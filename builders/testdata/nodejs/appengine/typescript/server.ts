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
 * @fileoverview Simple application that uses Polka.
 */

const PORT = Number(process.env.PORT) || 8080;
import * as polka from 'polka';

polka()
    .get(
        '/',
        (req, res) => {
          if (process.env.NODE_ENV == 'production') {
            res.end('PASS');
          }
        })
    .listen(PORT, () => {
      console.log(`App listening on port ${PORT}`);
    });
