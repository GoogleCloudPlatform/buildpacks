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
 * Responds 'PASS' if the runtime configs are correctly loaded.
 *
 * @param {!Object} req request context.
 * @param {!Object} res response context.
 */
const functions = require('firebase-functions');
const admin = require('firebase-admin');
admin.initializeApp();

exports.testFunction = functions.https.onRequest((req, res) => {
  let want = 'THE API KEY';
  let got = functions.config().someservice.key;
  if (got != want) {
    res.send(`Unexpected key: got ${got}, want ${want}`);
  }

  res.send('PASS');
});
