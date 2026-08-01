import os
from flask import Flask, request, Response
from typing import Any, Optional, Callable
import json
import logging
import sys
import time
from datetime import datetime
from urllib.parse import unquote
import traceback

app = Flask(__name__)
log_debug: Optional[Callable] = None
log_error: Optional[Callable] = None
current_log_batch = []
current_log_batch_length = 0
log_batches_to_report = []

FUNCTION_NAME = os.environ.get("X_GOOGLE_FUNCTION_NAME")
FUNCTION_VERSION = os.environ.get("X_GOOGLE_FUNCTION_VERSION")
CODE_LOCATION_DIR = os.environ.get("X_GOOGLE_CODE_LOCATION")
PACKAGE_JSON_FILE = f"{CODE_LOCATION_DIR}/package.json"
ENTRY_POINT = os.environ.get("X_GOOGLE_ENTRY_POINT", "function")
SUPERVISOR_HOSTNAME = os.environ.get("X_GOOGLE_SUPERVISOR_HOSTNAME")
SUPERVISOR_INTERNAL_PORT = int(os.environ.get("X_GOOGLE_SUPERVISOR_INTERNAL_PORT")) if SUPERVISOR_HOSTNAME else None
FUNCTION_TRIGGER_TYPE = os.environ.get("X_GOOGLE_FUNCTION_TRIGGER_TYPE")
WORKER_PORT = int(os.environ.get("X_GOOGLE_WORKER_PORT"))

EXECUTE_PREFIX = "/execute"
GOOGLE_CLOUD_SPAN_SAMPLED_HEADER_FIELD = "X-Google-Cloud-Span-Sampled"

MAX_LOG_LENGTH = 5000
MAX_LOG_BATCH_ENTRIES = 1500
MAX_LOG_BATCH_LENGTH = 150000
SUPERVISOR_LOG_TIMEOUT_MS = max(60, int(os.environ.get("X_GOOGLE_FUNCTION_TIMEOUT_SEC", 0))) * 1000
SUPERVISOR_KILL_TIMEOUT_MS = 5000

user_function: Optional[Callable] = None
user_code_error: Optional[str] = None
current_res: Optional[Response] = None
function_execution_id: Optional[str] = None
function_execution_finished = False

def write_to_stream(out, severity):
    def new_write(chunk, encoding=None, lock=False):
        nonlocal out
        if chunk is not None:
            timestamp = datetime.now().isoformat()
            log_prefix = f"[{severity}] [{timestamp}]"
            if function_execution_id:
                log_prefix += f" [{function_execution_id}]"
            lines = str(chunk).split('\n')
            for i in range(len(lines)):
                if lines[i]:
                    lines[i] = f"{log_prefix} {lines[i]}"
            chunk = '\n'.join(lines)
        out.write(chunk, encoding, lock)
    return new_write

def log_to_supervisor(severity):
    def log_function(*args):
        nonlocal severity
        entry = {
            "TextPayload": str(args).strip()[:MAX_LOG_LENGTH],
            "Severity": severity,
            "Time": datetime.now().isoformat()
        }
        if function_execution_id:
            entry["ExecutionID"] = function_execution_id
        global current_log_batch, current_log_batch_length
        if len(current_log_batch) + 1 > MAX_LOG_BATCH_ENTRIES or \
           current_log_batch_length + len(entry["TextPayload"]) > MAX_LOG_BATCH_LENGTH:
            start_new_log_batch()
        current_log_batch.append(entry)
        current_log_batch_length += len(entry["TextPayload"])
        trigger_log_reporting()
    return log_function

def start_new_log_batch():
    global current_log_batch, current_log_batch_length
    if current_log_batch:
        log_batches_to_report.append(current_log_batch)
    current_log_batch = []
    current_log_batch_length = 0

def trigger_log_reporting():
    if not hasattr(trigger_log_reporting, "in_progress"):
        trigger_log_reporting.in_progress = False
    if not trigger_log_reporting.in_progress:
        trigger_log_reporting.in_progress = True
        report_next_log_batch()

def report_next_log_batch(error=None):
    global log_batches_to_report
    if error:
        kill_instance()
        return
    if log_batches_to_report and len(log_batches_to_report[0]) > 0:
        post_data = json.dumps(log_batches_to_report.pop(0))
        start_new_log_batch()
        post_to_supervisor("/_ah/log", post_data, SUPERVISOR_LOG_TIMEOUT_MS, report_next_log_batch)
    elif log_batches_to_report and callable(log_batches_to_report[0]):
        callback = log_batches_to_report.pop(0)
        callback()
        report_next_log_batch()
    else:
        trigger_log_reporting.in_progress = False

def kill_instance():
    post_to_supervisor("/_ah/kill", "", SUPERVISOR_KILL_TIMEOUT_MS, lambda: sys.exit(16))

def post_to_supervisor(path, data, timeout_ms, callback):
    import requests
    try:
        start_time = time.time()
        response = requests.post(
            f"http://{SUPERVISOR_HOSTNAME}:{SUPERVISOR_INTERNAL_PORT}{path}",
            data=data,
            headers={
                "Content-Type": "application/json",
                "x-cloud-trace-agent-request": "cloudfunctions-internal-call"
            },
            timeout=timeout_ms / 1000
        )
        if response.status_code < 200 or response.status_code >= 300:
            raise Exception(f"Incorrect response code: {response.status_code}")
        callback()
    except requests.exceptions.RequestException as e:
        logging.error(f"Error calling supervisor: {str(e)}")
        kill_instance()

def hook_into_output():
    global log_debug, log_error
    if SUPERVISOR_HOSTNAME is None or SUPERVISOR_INTERNAL_PORT is None:
        debug_write = write_to_stream(sys.stdout, "D")
        error_write = write_to_stream(sys.stderr, "E")
        log_debug = lambda *args: debug_write(" ".join(map(str, args)) + "\n")
        log_error = lambda *args: error_write(" ".join(map(str, args)) + "\n")
    else:
        start_new_log_batch()
        log_debug = log_to_supervisor("DEBUG")
        log_error = log_to_supervisor("ERROR")
        logging.getLogger().handlers = []
        logging.basicConfig(level=logging.INFO)
        logging.info = log_to_supervisor("INFO")
        logging.error = log_to_supervisor("ERROR")

hook_into_output()

@app.route(f"{EXECUTE_PREFIX}/*", methods=["GET"])
def check():
    return "OK"

@app.route("/load")
def load():
    if user_function:
        current_res.send("User function is ready")
    else:
        current_res.status(500).send(user_code_error)

if __name__ == "__main__":
    app.run(port=WORKER_PORT)
