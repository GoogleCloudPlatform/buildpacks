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
 * Responds 'PASS' if the 'server' local dependency was installed correctly.
 *
 * @param {!Object} req request context.
 * @param {!Object} res response context.
 */
exports.testFunction = (req, res) => {
  const pkg = require('server/package.json');
  const got = pkg.version;
  const want = '0.5.2';
  if (got == want) {
    res.send('PASS');
  } else {
    res.send(`Unexpected server version: got ${got}, want ${want}`);
  }
};
