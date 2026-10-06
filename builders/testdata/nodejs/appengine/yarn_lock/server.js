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
 * @fileoverview Application that verifies the installed version of Polka.
 *
 * The installed version should match the one specified in yarn.lock.
 */

'use strict';

const polka = require('polka');
const ppkg = require('polka/package.json');

polka()
  .get('/', (req, response) => {
    response.writeHead(200, {"Content-Type": "text/plain"});
    let want = "0.5.2";
    let got = ppkg.version;
    if (want != got) {
      response.end(`Unexpected polka version: got ${got}, want ${want}`);
    } else {
      response.end("PASS");
    }
  })
  .listen(process.env.PORT);
