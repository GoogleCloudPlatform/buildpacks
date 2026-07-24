/**
 * Copyright 2020 Google LLC
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
 * @fileoverview Application that verifies the installed version of Express.
 *
 * The installed version should match the one specified in yarn.lock.
 */

'use strict';

const express = require('express');
const epkg = require('express/package.json');
const http = require('http');

const server = http.createServer((request, response) => {
  response.writeHead(200, {"Content-Type": "text/plain"});
  let want = "4.1.0";
  let got = epkg.version;
  if (want != got) {
    response.end(`Unexpected express version: got ${got}, want ${want}`);
  } else {
    response.end("PASS");
  }
});

server.listen(process.env.PORT);
