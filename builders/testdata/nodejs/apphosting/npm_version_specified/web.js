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
 * @fileoverview A hello world application with a specific NPM version specified
 * in package.json.
 */


const { exec } = require("child_process");
const express = require('express');
const app = express();

app.get('/version', (req, res) => {
  exec("npm --version", (error, stdout, stderr) => {
    const want = req.query.want;
    const got = stdout.trim();
    if (!want) {
      res.end('FAIL: ?want must be set to a version');
    } else if (got !== want) {
      res.end(`FAIL: current version: ${got}; want ${want}`);
    } else {
      res.end('PASS');
    }
  });
});

app.listen(process.env.PORT || 8080);
