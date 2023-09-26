# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Simple web server used to validate that pip vendored deps are installed.
"""
from http.server import BaseHTTPRequestHandler, HTTPServer
from sample_pip_dependency import sample


""" Simple web server to respond to HTTP GET.
"""
class MyServer(BaseHTTPRequestHandler):
    def _set_headers(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()

    def _html(self, message):
        """This just generates an HTML document that includes `message`
        in the body.
        """
        content = f"{message}"
        return content.encode("utf8")  # NOTE: must return a bytes object!

    def do_GET(self):
        self._set_headers()
        message = sample.helloworld()
        if message == "hello world - from pip dependency":
          self.wfile.write(self._html("PASS"))
        else:
          self.wfile.write(self._html("FAIL"))

if __name__ == "__main__":
    webServer = HTTPServer(("0.0.0.0", 8080), MyServer)

    try:
        webServer.serve_forever()
    except KeyboardInterrupt:
        pass

    webServer.server_close()
