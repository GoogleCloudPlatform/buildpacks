// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// node.js server that runs user's code (extracted from a zip file from GCS) on
// HTTP request. HTTP response is sent once user's function has completed.
// The server accepts following HTTP requests:
//   - GET '/check' for checking if the server is ready.
//   - GET '/load' for loading the client function.
//   - POST '/*' for executing functions (only for servers handling functions
//     with non-HTTP trigger).
//   - ANY (all methods) '/*' for executing functions (only for servers handling
//     functions with HTTP trigger).
// The server requires the following environment variable:
//   - X_GOOGLE_FUNCTION_NAME - the name of the client function.
//   - X_GOOGLE_WORKER_PORT - defines the port on which this server listens to
//   all HTTP
//     requests.
//   - X_GOOGLE_ENTRY_POINT - defines the name of the function within user's
//   node module
//     to execute. If such a function is not defined, then falls back to
//     'function' name.
//   - X_GOOGLE_CODE_LOCATION - defines path to directory that contains user's
//   node
//     module with function to execute and all the libraries used by the
//     function.
//   - X_GOOGLE_SUPERVISOR_HOSTNAME and X_GOOGLE_SUPERVISOR_INTERNAL_PORT -
//   define address and
//     port to which logs are written. If not specified, logs are written to
//     stdout and stderr.
//   - X_GOOGLE_FUNCTION_TRIGGER_TYPE - the trigger type of the client function.

var FUNCTION_NAME = process.env.X_GOOGLE_FUNCTION_NAME;
var FUNCTION_VERSION = process.env.X_GOOGLE_FUNCTION_VERSION;

var bodyParser = require('body-parser');
var domain = require('domain');
var express = require('express');
var fs = require('fs');
var http = require('http');
var onFinished = require('on-finished');
var util = require('util');

var CODE_LOCATION_DIR = process.env.X_GOOGLE_CODE_LOCATION;
var PACKAGE_JSON_FILE = CODE_LOCATION_DIR + '/package.json';
var ENTRY_POINT = process.env.X_GOOGLE_ENTRY_POINT ?
    process.env.X_GOOGLE_ENTRY_POINT :
    'function';
var SUPERVISOR_HOSTNAME = process.env.X_GOOGLE_SUPERVISOR_HOSTNAME ?
    process.env.X_GOOGLE_SUPERVISOR_HOSTNAME :
    null;
var SUPERVISOR_INTERNAL_PORT = process.env.X_GOOGLE_SUPERVISOR_INTERNAL_PORT ?
    process.env.X_GOOGLE_SUPERVISOR_INTERNAL_PORT :
    null;
var FUNCTION_TRIGGER_TYPE = process.env.X_GOOGLE_FUNCTION_TRIGGER_TYPE;
var FUNCTION_TIMEOUT_SEC = process.env.X_GOOGLE_FUNCTION_TIMEOUT_SEC;
var WORKER_PORT = process.env.X_GOOGLE_WORKER_PORT;
var NEW_FUNCTION_SIGNATURE = process.env.X_GOOGLE_NEW_FUNCTION_SIGNATURE &&
    process.env.X_GOOGLE_NEW_FUNCTION_SIGNATURE == 'true';
var LOAD_ON_START = process.env.X_GOOGLE_LOAD_ON_START &&
    process.env.X_GOOGLE_LOAD_ON_START == 'true';
var CONTAINER_LOGGING_ENABLED =
    process.env.X_GOOGLE_CONTAINER_LOGGING_ENABLED &&
    process.env.X_GOOGLE_CONTAINER_LOGGING_ENABLED == 'true';
// HTTP header field that is added to Worker response to signalize problems with
// executing the client function.
var FUNCTION_STATUS_HEADER_FIELD = 'X-Google-Status';
// If set then the request is coming from Fetcher and shouldn't be parsed in
// Worker.
var FETCHER_ORIGIN = 'X-Google-Fetcher-Origin';

// URL path prefix which is used for sending user requests. This prefix should
// be removed from express request object before passing it to function.
var EXECUTE_PREFIX = '/execute';

var GOOGLE_CLOUD_SPAN_SAMPLED_HEADER_FIELD = 'X-Google-Cloud-Span-Sampled';

var MAX_LOG_LENGTH = 5000;
var MAX_LOG_BATCH_ENTRIES = 1500;
var MAX_LOG_BATCH_LENGTH = 150000;
var SUPERVISOR_LOG_TIMEOUT_MS = Math.max(60, FUNCTION_TIMEOUT_SEC || 0) * 1000;
var SUPERVISOR_KILL_TIMEOUT_MS = 5000;

// App to use for function executions.
var app = express();
var logDebug;
var logError;

var logReportingInProgress = false;
var currentLogBatch;
var currentLogBatchLength;
var logBatchesToReport = [];

var userFunction = null;
var userCodeError = null;
var currentRes = null;

// Function execution data.
var functionExecutionId;
var functionExecutionFinished;

/**
 * Creates a function which writes logs to stream `out` with the specified
 * severity.
 * @param {!Stream} out
 * @param {string} severity
 * @return {!Function}
 */
var writeToStream = function(out, severity) {
  var originalWrite = out.write;
  var newLine = {isNewLine: true};
  return function(chunk, encoding, callback) {
    if (chunk !== null) {
      var severityPrefix = '[' + severity + ']';
      var timestampPrefix = '[' + new Date().toISOString() + ']';
      var executionIdPrefix =
          functionExecutionId ? '[' + functionExecutionId + ']' : '';
      var logPrefix = severityPrefix + timestampPrefix + executionIdPrefix;
      if (newLine.isNewLine) {
        chunk = logPrefix + ' ' + chunk;
        newLine.isNewLine = false;
      }
      if (chunk.slice(-1) == '\n') {
        chunk =
            chunk.slice(0, -1).replace(/\n/g, '\n' + logPrefix + ' ') + '\n';
        newLine.isNewLine = true;
      } else {
        chunk = chunk.replace(/\n/g, '\n' + logPrefix + ' ');
      }
    }
    originalWrite.call(out, chunk, encoding, callback);
  };
};

/**
 * Creates a function which queues logs to be written to the supervisor with the
 * specified severity.
 * @param {string} severity
 * @return {!Function}
 */
var logToSupervisor = function(severity) {
  return function() {
    // Prepare a log entry.
    var entry = {
      TextPayload:
          util.format.apply(util, arguments).substring(0, MAX_LOG_LENGTH),
      Severity: severity,
      Time: new Date().toISOString(),
    };
    if (functionExecutionId) {
      entry.ExecutionID = functionExecutionId;
    }
    // Start a new batch if the current one would grow too much.
    if (currentLogBatch.Entries.length + 1 > MAX_LOG_BATCH_ENTRIES ||
        currentLogBatchLength + entry.TextPayload.length >
            MAX_LOG_BATCH_LENGTH) {
      startNewLogBatch();
    }
    // Add the entry to the current batch.
    currentLogBatch.Entries.push(entry);
    currentLogBatchLength += entry.TextPayload.length;
    triggerLogReporting();
  };
};

/**
 * Creates a new empty log batch to be filled in by the upcoming logs.
 */
var startNewLogBatch = function() {
  currentLogBatch = {Entries: []};
  currentLogBatchLength = 0;
  logBatchesToReport.push(currentLogBatch);
};

/**
 * Triggers log reporting if it is not running yet.
 */
var triggerLogReporting = function() {
  if (!logReportingInProgress) {
    logReportingInProgress = true;
    reportNextLogBatch();
  }
};

/**
 * Sends the next available log batch to the supervisor, or executes a callback
 * added to the log batch queue.
 * @param {!Function} err
 */
var reportNextLogBatch = function(err) {
  // Kill the instance on unexpected errors.
  if (err) {
    killInstance();
    return;
  }
  if (logBatchesToReport.length > 0 && logBatchesToReport[0].Entries &&
      logBatchesToReport[0].Entries.length > 0) {
    // If the next batch is a regular non-empty log batch, report it.
    var postData = JSON.stringify(logBatchesToReport[0]);
    logBatchesToReport.shift();
    if (logBatchesToReport.length == 0) {
      startNewLogBatch();
    }
    postToSupervisor(
        '/_ah/log', postData, SUPERVISOR_LOG_TIMEOUT_MS, reportNextLogBatch);
  } else if (
      logBatchesToReport.length > 0 &&
      typeof logBatchesToReport[0] == 'function') {
    // If the next batch is actually a callback, run it.
    logBatchesToReport[0]();
    logBatchesToReport.shift();
    process.nextTick(reportNextLogBatch);
  } else {
    // Stop reporting.
    logReportingInProgress = false;
  }
};

/**
 * Kills the current instance by sending a kill signal to supervisor and exiting
 * the current process.
 */
var killInstance = function() {
  postToSupervisor('/_ah/kill', '', SUPERVISOR_KILL_TIMEOUT_MS, function() {
    // Use an exit code which is unused by Node.js:
    // https://nodejs.org/api/process.html#process_exit_codes
    process.exit(16);
  });
};

/**
 * Sends a POST request to the supervisor.
 * @param {string} path
 * @param {string} postData
 * @param {number} timeoutMs
 * @param {!Function} callback
 */
var postToSupervisor = function(path, postData, timeoutMs, callback) {
  var postOptions = {
    hostname: SUPERVISOR_HOSTNAME,
    port: SUPERVISOR_INTERNAL_PORT,
    path: path,
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Content-Length': Buffer.byteLength(postData),
      // Force outgoing requests not to be traced by Stackdriver Trace. We have
      // to use this workaround as Trace registers all outgoing requests.
      'x-cloud-trace-agent-request': 'cloudfunctions-internal-call'
    },
    timeout: timeoutMs  // Set connection and read timeout.
  };
  var req = http.request(postOptions, function(res) {
    // Response data must be consumed to fire the 'end' event and free up
    // memory: https://nodejs.org/api/http.html#http_class_http_clientrequest.
    res.resume();
    if (res.statusCode < 200 || res.statusCode >= 300) {
      process.stderr.write(
          'Incorrect response code when calling supervisor: ' + res.statusCode +
          '\n');
      // Unexpected error, supervisor may be unhealthy.
      callback(new Error('Incorrect response code from supervisor'));
    } else {
      callback();
    }
  });
  req.on('timeout', function() {
    process.stderr.write('Timeout when calling supervisor\n');
    // Unexpected error, supervisor may be unhealthy.
    callback(new Error('Timeout when calling supervisor'));
  });
  req.on('error', function(err) {
    process.stderr.write(
        'Error when calling supervisor: ' + getErrorDetails(err) + '\n');
    // Unexpected error, supervisor may be unhealthy.
    callback(new Error('Error when calling supervisor'));
  });
  req.write(postData);
  req.end();
};

/**
 * Installs custom handlers for writing logs.
 */
var hookIntoOutput = function() {
  if (SUPERVISOR_HOSTNAME === null || SUPERVISOR_INTERNAL_PORT === null) {
    // TODO: Remove the branch which works without the supervisor.
    var debugWrite = writeToStream(process.stdout, 'D');
    logDebug = function() {
      debugWrite(util.format.apply(util, arguments) + '\n');
    };
    var errorWrite = writeToStream(process.stderr, 'E');
    logError = function() {
      errorWrite(util.format.apply(util, arguments) + '\n');
    };
    process.stdout.write = writeToStream(process.stdout, 'I');
    process.stderr.write = writeToStream(process.stderr, 'E');
  } else {
    if (CONTAINER_LOGGING_ENABLED) {
      logDebug = console.log;
      logError = function() {
        return console.error('\n' + util.format.apply(util, arguments));
      };
    } else {
      startNewLogBatch();
      logDebug = logToSupervisor('DEBUG');
      logError = logToSupervisor('ERROR');
      console.log = logToSupervisor('INFO');
      console.info = console.log;
      console.error = logToSupervisor('ERROR');
      console.warn = console.error;
    }
  }
};

/**
 * Returns absolute path for the given relative code path.
 * @param {string} codeRelativePath
 * @return {string}
 */
var getCodeAbsolutePath = function(codeRelativePath) {
  return CODE_LOCATION_DIR + '/' + codeRelativePath;
};

/**
 * Returns detailed error message.
 * @param {!Error} err
 * @return {!Object}
 */
var getErrorDetails = function(err) {
  return err.stack || err;
};

/**
 * Returns general error message.
 * @param {!Error} err
 * @return {!Object}
 */
var getErrorMessage = function(err) {
  return err.message || err;
};

/**
 * Handles an error which happened when loading user's code.
 * @param {string} errorMessage
 */
var processUserCodeError = function(errorMessage) {
  if (errorMessage !== userCodeError) {
    logDebug(errorMessage);
    userCodeError = errorMessage;
  }
};

/**
 * Processes definition of user's node module specifying function.
 * Returns relative path to the javascript file defining module.
 * Return null in case of error in definition of the module.
 * @return {?string}
 */
var processNodeModuleDefinition = function() {
  // There are a couple of options for user to specify a function file:
  // - define 'main' field in package.json,
  // - otherwise: the default is index.js, but if this file is not present,
  //   then we fall back to function.js.
  if (fs.existsSync(PACKAGE_JSON_FILE)) {
    try {
      var packageJson = require(PACKAGE_JSON_FILE);
      if (packageJson.main) {
        return packageJson.main;
      }
    } catch (ex) {
      processUserCodeError(
          'package.json file can\'t be parsed. ' +
          'Please check whether syntax is correct:\n' + ex.message);
      return null;
    }
  }
  if (fs.existsSync(CODE_LOCATION_DIR + '/index.js')) {
    return 'index.js';
  }
  if (fs.existsSync(CODE_LOCATION_DIR + '/function.js')) {
    return 'function.js';
  }
  processUserCodeError(
      'File index.js or function.js that is expected to ' +
      'define function doesn\'t exist in the root directory.');
  return null;
};

/**
 * Returns user's function from function file.
 * Returns null if function can't be retrieved.
 * @param {string} functionFileRelativePath
 * @return {?Function}
 */
var getUserFunction = function(functionFileRelativePath) {
  if (!fs.existsSync(getCodeAbsolutePath(functionFileRelativePath))) {
    processUserCodeError(
        'File ' + functionFileRelativePath + ' that is ' +
        'expected to define function doesn\'t exist.');
    return null;
  }

  try {
    var functionCode = require(getCodeAbsolutePath(functionFileRelativePath));
    var userFunction =
        ENTRY_POINT.split('.').reduce(function(code, entryPointPart) {
          if (typeof code === 'undefined') {
            return undefined;
          } else {
            return code[entryPointPart];
          }
        }, functionCode);

    if (typeof userFunction === 'undefined') {
      if (functionCode.hasOwnProperty('function')) {
        userFunction = functionCode['function'];
      } else {
        processUserCodeError(
            'Node.js module defined by file ' + functionFileRelativePath +
            ' is expected to export function named ' + ENTRY_POINT);
        return null;
      }
    }

    if (typeof userFunction != 'function') {
      processUserCodeError(
          'The function exported from file ' + functionFileRelativePath +
          ' as ' + ENTRY_POINT +
          ' needs to be of type function. Got: ' + typeof userFunction);
      return null;
    }

    return userFunction;
  } catch (ex) {
    var errorDetails = getErrorDetails(ex);
    var additionalHint = '';
    if (errorDetails.includes('Cannot find module')) {
      additionalHint =
          'Did you list all required modules in the package.json ' +
          'dependencies?\n';
    } else {
      additionalHint = 'Is there a syntax error in your code?\n';
    }
    processUserCodeError(
        'Code in file ' + functionFileRelativePath + ' can\'t be loaded.\n' +
        additionalHint + 'Detailed stack trace: ' + errorDetails);
    return null;
  }
};

/**
 * Loads user's code.
 * @return {boolean} Whether user's code has been loaded successfully.
 */
var loadUserCode = function() {
  var functionFileRelativePath = processNodeModuleDefinition();
  if (!functionFileRelativePath) {
    return false;
  }
  userFunction = getUserFunction(functionFileRelativePath);
  return !!userFunction;
};

/**
 * Executes the callback only after flushing logs. If a log queue is used, then
 * the queued logs are processed and sent out before the callback is executed.
 * @param {function()} callback
 */
var callAfterFlushingLogs = function(callback) {
  if (SUPERVISOR_HOSTNAME === null || SUPERVISOR_INTERNAL_PORT === null ||
      CONTAINER_LOGGING_ENABLED) {
    // TODO: Remove the branch which works without the supervisor.
    callback();
  } else {
    if (currentLogBatch.Entries.length == 0) {
      // Skip an empty batch.
      logBatchesToReport.pop();
    }
    logBatchesToReport.push(callback);
    startNewLogBatch();
    triggerLogReporting();
  }
};

/**
 * Sends back a response to the incoming request.
 * @param {*} result
 * @param {?Error} err
 * @param {!express.Response} res
 */
var sendResponse = function(result, err, res) {
  if (err) {
    res.set(FUNCTION_STATUS_HEADER_FIELD, 'error');
    // Sending error message back is fine for Pub/Sub-based functions as they do
    // not reach the caller anyway.
    res.send(getErrorMessage(err));
    return;
  }
  if (typeof result === 'undefined' || result === null) {
    res.sendStatus(204);  // No Content
  } else if (typeof result == 'number') {
    // This isn't technically compliant but numbers otherwise cause us to set
    // the status code to that number instead of sending the number as a body.
    res.json(result);
  } else {
    try {
      res.send(result);
    } catch (sendErr) {
      // If a customer passes a non-serializeable object (e.g. one with a cycle)
      // then res.send will throw. Customers don't always put a lot of thought
      // into the return value because it's currently only used for
      // CallFunction. The most sensible resolution here is to succeed the
      // function (this was the customer's clear intent) but send a 204
      // (NO CONTENT) and log an error message explaining why their content
      // wasn't sent.
      console.error('Error serializing return value: ' + sendErr.toString());
      res.sendStatus(204);  // No Content
    }
  }
};

/**
 * Handles the end of a single function execution.
 * @param {?Error} err
 * @param {*} result
 * @param {!express.Response} res
 */
var finishCallback = function(err, result, res) {
  if (err) {
    logError(getErrorDetails(err));
  }
  callAfterFlushingLogs(function() {
    sendResponse(result, err, res);
  });
};

/**
 * Logs an error message and sends back an error response to the incoming
 * request.
 * @param {!Error} err
 * @param {?express.Response} res
 * @param {?Function} callback
 */
var logAndSendError = function(err, res, callback) {
  logError(getErrorDetails(err));
  callAfterFlushingLogs(function() {
    // If user function has already sent response headers, the response with
    // error message cannot be sent. This check is done inside the callback,
    // right before sending the response, to make sure that no concurrent
    // execution sends the response between the check and 'send' call below.
    if (res && !res.headersSent) {
      res.set(FUNCTION_STATUS_HEADER_FIELD, 'crash');
      res.send(getErrorMessage(err));
    }
    if (callback) {
      callback();
    }
  });
};

/**
 * If a given str starts with prefix this prefix is removed.
 * @param {string} str
 * @param {string} prefix
 * @return {string}
 */
var removePrefix = function(str, prefix) {
  if (typeof str == 'string' && str.startsWith(prefix)) {
    return str.slice(prefix.length);
  }
  return str;
};

/**
 * Adds a leading slash to a given str, if it does not start with a slash
 * already.
 * @param {string} str
 * @return {string}
 */
var ensureLeadingSlash = function(str) {
  if (typeof str == 'string' && !str.startsWith('/')) {
    return '/' + str;
  }
  return str;
};

/**
 * Adjusts request path before it's ready to be passed to user function.
 * @param {string} str
 * @return {string}
 */
var adjustUserReqPath = function(str) {
  str = removePrefix(str, EXECUTE_PREFIX);
  // Maintain legacy behavior for Node.js 6, in which leading slash is not added
  // in case of an empty path.
  if (!process.version.startsWith('v6.')) {
    str = ensureLeadingSlash(str);
  }
  return str;
};

/**
 * Adjusts req object before it's ready to be passed to user function.
 * @param {!express.Request} req
 */
var adjustUserReq = function(req) {
  req.url = adjustUserReqPath(req.url);
  req.path = adjustUserReqPath(req.path);
  req.baseUrl = removePrefix(req.baseUrl, EXECUTE_PREFIX);
  req.originalUrl = adjustUserReqPath(req.originalUrl);
  if (req.route != null) {
    req.route.path = removePrefix(req.route.path, EXECUTE_PREFIX);
    req.route.path = ensureLeadingSlash(req.route.path);
  }
};

/**
 * Adjusts res object before it's ready to be passed back.
 * @param {!express.Response} res
 */
var setTraceHeader = function(res) {
  try {
    res.set(GOOGLE_CLOUD_SPAN_SAMPLED_HEADER_FIELD, '0');
  } catch (ex) {
    console.error('Error when setting trace header ' + ex.toString());
  }
};

hookIntoOutput();

// Set request-specific values in the very first middleware.
app.use(EXECUTE_PREFIX + '*', function(req, res, next) {
  currentRes = res;
  functionExecutionId = req.get('Function-Execution-Id');
  functionExecutionFinished = false;
  next();
});
// Set limit to a value larger than 32MB, which is maximum limit of higher level
// layers anyway.
var requestLimit = '1024mb';

/**
 * Retains a reference to the raw body buffer to allow access to the raw body
 * for things like request signature validation.  This is used as the "verify"
 * function in body-parser options.
 * @param {!express.Request} req
 * @param {!express.Response} res
 * @param {Buffer} buf
 */
var rawBodySaver = function(req, res, buf) {
  req.rawBody = buf;
};

var defaultBodySavingOptions = {limit: requestLimit, verify: rawBodySaver};

// The parser will process ALL content types so must come last.
var rawBodySavingOptions = {
  limit: requestLimit,
  verify: rawBodySaver,
  type: '*/*'
};

// Use extended query string parsing for URL-encoded bodies.
var urlEncodedOptions = {
  limit: requestLimit,
  verify: rawBodySaver,
  extended: true
};

app.use(bodyParser.json(defaultBodySavingOptions));
app.use(bodyParser.text(defaultBodySavingOptions));
app.use(bodyParser.urlencoded(urlEncodedOptions));

// MUST be last in the list of body parsers as subsequent parsers will be
// skipped when one is matched.
app.use(bodyParser.raw(rawBodySavingOptions));

app.get('/load', function(req, res) {
  var ready = userFunction || loadUserCode();
  callAfterFlushingLogs(function() {
    if (ready) {
      res.send('User function is ready');
    } else {
      // A non-transient error occurred when loading user code.
      res.set(FUNCTION_STATUS_HEADER_FIELD, 'load_error');
      res.status(500).send(userCodeError);
    }
  });
});

app.get('/check', function(req, res) {
  res.status(200).send('OK');
});

/**
 * Exposes a function which handles function execution request.
 * @param {!Function} execute Runs user's function.
 * @constructor
 */
var Handler = function(execute) {
  this.handle = function(req, res) {
    adjustUserReq(req);
    var d = domain.create();
    // Catch unhandled errors originating from this request.
    d.on('error', function(err) {
      if (functionExecutionFinished) {
        logDebug('Ignoring exception from a finished function');
      } else {
        functionExecutionFinished = true;
        logAndSendError(err, res);
      }
    });
    d.run(function() {
      if (!userFunction) {
        functionExecutionFinished = true;
        logDebug('User function not ready!');
        res.set(FUNCTION_STATUS_HEADER_FIELD, 'load_error');
        res.send('User function not ready!');
        return;
      }
      Promise.resolve().then(d.bind(() => {
        process.nextTick(function() {
          setTraceHeader(res);
          execute(req, res);
        });
      }));
    });
  };
};

if (FUNCTION_TRIGGER_TYPE == 'HTTP_TRIGGER') {
  app.use(EXECUTE_PREFIX + '*', function(req, res, next) {
    onFinished(res, function(err, res) {
      functionExecutionFinished = true;
    });
    next();
  });

  app.all(EXECUTE_PREFIX + '*', function(req, res) {
    var handler = new Handler(function(req, res) {
      userFunction(req, res);
    });
    handler.handle(req, res);
  });
} else {
  app.post(EXECUTE_PREFIX + '*', function(req, res) {
    var handler = new Handler(function(req, res) {
      var event = req.body;
      var callback = process.domain.bind(function(err, result) {
        if (functionExecutionFinished) {
          logDebug('Ignoring extra callback call');
        } else {
          functionExecutionFinished = true;
          finishCallback(err, result, res);
        }
      });
      var data = event.data;
      var context = event.context;
      // Support legacy events with context properties represented as an event
      // properties.
      if (NEW_FUNCTION_SIGNATURE && context == undefined) {
        // Context is everything but data.
        context = event;
        // Clear the property before removing field so the data object
        // is not deleted.
        context.data = undefined;
        delete context.data;
      }
      // Callback style if user function has multiple arguments (old signature,
      // for Node.js v6) or more than 2 arguments (new signature, Node.js 8).
      if (NEW_FUNCTION_SIGNATURE) {
        if (userFunction.length > 2) {
          return userFunction(data, context, callback);
        }
      } else if (userFunction.length > 1) {
        return userFunction(event, callback);
      }

      Promise.resolve()
          .then(function() {
            var result;
            if (NEW_FUNCTION_SIGNATURE) {
              result = userFunction(data, context);
            } else {
              result = userFunction(event);
            }
            return result;
          })
          .then(
              function(result) {
                callback(null, result);
              },
              function(err) {
                callback(err);
              });
    });
    handler.handle(req, res);
  });
}

/**
 * Registers handlers for uncaught exceptions and other unhandled errors.
 */
var errorHandler = function() {
  process.on('uncaughtException', function(err) {
    logError('Uncaught exception');
    logAndSendError(err, currentRes, killInstance);
  });

  process.on('unhandledRejection', function(err) {
    logError('Unhandled rejection');
    logAndSendError(err, currentRes, killInstance);
  });

  process.on('exit', function(code) {
    logAndSendError('Process exited with code ' + code, currentRes);
  });

  process.on('SIGTERM', () => {
    server.close(() => {
      process.exit();
    });
  });
};
if (LOAD_ON_START) {
  loadUserCode();
}
app.enable('trust proxy');  // To respect X-Forwarded-For header.
var server = app.listen(WORKER_PORT, errorHandler);
server.timeout = 0;  // Disable automatic timeout on incoming connections.

module.exports = app;  // for testing
