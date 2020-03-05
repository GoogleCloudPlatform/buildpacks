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
 * Responds 'PASS' to any HTTP requests, used in GCF builder acceptance tests.
 *
 * @param {!Object} req request context.
 * @param {!Object} res response context.
 */
exports.testFunction = (req, res) => {
  res.send('PASS');
};

// process.env.FOO is only available at runtime.
console.log(process.env.FOO.toLowerCase());


