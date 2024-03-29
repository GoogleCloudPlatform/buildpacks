/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * @fileoverview Application that checks that a file exists.
 */

'use strict';

const fs = require('fs');
const http = require('http');
const isexe = require('isexe');

const server = http.createServer((request, response) => {
  response.writeHead(200, {"Content-Type": "text/plain"});
  try {
    isexe.sync('index.js');
    response.end("PASS");
  } catch (e) {
    response.end("Failed to use the isexe dependency which should be installed by Yarn");
  }
});

server.listen(process.env.PORT || 8080);
