/**
 * Transforms a DependencyLogList .textproto message into valid JSON for BigQuery.
 * This is integrated as a transformation UDF in the "serverless-runtimes-qa" project's
 * "bq-intelligence-sub" Pub/Sub push subscriber which writes to the
 * "serverless-runtimes-qa.build_intelligence_dataset.dependencies" table. 
 * @param {!Object<string, string|!Object<string, string>>} message Pub/Sub message
 * @param {!Object<string, *>} metadata Pub/Sub message metadata
 * @return {?Object<string, string|!Object<string, string>>}
 *
 * Input structure received from Pub/Sub (outer layer):
 * {
 *   "message": {
 *     "data": "<base64_encoded_bytes_of_.textproto>"
 *   }
 * }
 * Where decoding `message.data` yields the .textproto payload:
 *   runtime: "nodejs"
 *   language: "javascript"
 *   dependencies { package_name: "express" package_version: "4.18.2" ... }
 *
 * Output structure returned by this UDF (outer layer intact with JSON data) which is compatible with BigQuery schema:
 * {
 *   "message": {
 *     "data": "{\"runtime\":\"nodejs\",\"language\":\"javascript\",\"region\":null,\"dependencies\":[...]}"
 *   }
 * }
 */
function transformDependencyLog(message, metadata) {
  if (!message || !message.data) {
    return message;
  }

  let rawText = message.data;

  // 1. Decode base64 if REST transmitted without decoding
  if (rawText.indexOf(':') === -1 && rawText.indexOf('{') === -1) {
    try {
      if (typeof atob === 'function') {
        rawText = atob(rawText);
      } else if (typeof Buffer !== 'undefined') {
        rawText = Buffer.from(rawText, 'base64').toString('utf8');
      }
    } catch (e) {}
  }

  // 2. Extract top-level scalar string fields (runtime, language, region)
  function extractString(key) {
    let re = new RegExp(key + '\\s*:\\s*"([^"]+)"');
    let match = rawText.match(re);
    return match ? match[1] : null;
  }

  let runtime = extractString("runtime");
  let language = extractString("language");
  let region = extractString("region");
  
  
  // Extract strings inside this specific dependency block
  function getBlockString(block, key) {
    let re = new RegExp(key + '\\s*:\\s*"([^"]+)"');
    let m = block.match(re);
    return m ? m[1] : null;
  }

  // 4. Dependency Block Parser:
  // Matches: dependencies: { ... } OR dependencies { ... } OR dependencies: < ... >
  // The :? allows for an optional colon. The (?:\{([^}]*)\}|<([^>]*)>) captures content inside either {} or <>
  let depRegex = /dependencies\s*:?\s*(?:\{([^}]*)\}|<([^>]*)>)/g;
  let depMatch;
  let depArray = [];

  while ((depMatch = depRegex.exec(rawText)) !== null) {
    // depMatch[1] holds {} content, depMatch[2] holds <> content
    let block = depMatch[1] || depMatch[2] || "";
    let depObj = {
      package_name: null,
      package_version: null,
      explicit_dep: false,
      dep_type: null
    };

    depObj.package_name = getBlockString(block, "package_name");
    depObj.package_version = getBlockString(block, "package_version");
    depObj.dep_type = getBlockString(block, "dep_type");

    // Extract explicit_dep (handles true, false, 1, or 0)
    let expM = block.match(/explicit_dep\s*:\s*(true|false|1|0)/i);
    if (expM) {
      let val = expM[1].toLowerCase();
      depObj.explicit_dep = (val === 'true' || val === '1');
    }

    depArray.push(depObj);
  }

  // 5. Construct the clean BigQuery record and replace message data with stringified JSON
  let cleanRow = {
    runtime: runtime,
    language: language,
    region: region,
    dependencies: depArray
  };

  message.data = JSON.stringify(cleanRow);

  return message;
}