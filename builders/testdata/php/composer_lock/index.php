<?php
// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

require_once __DIR__ . '/vendor/autoload.php';

// As of when this test was written, 8.5.22 was the current version of phpunit.
// composer.lock specifies 8.5.21 and composer.json specifies ^8.5.22, so we
// expect composer.lock to ensure we get 8.5.21.
$want = '8.5.21';

$got = PHPUnit\Runner\Version::id();

if ($got !== $want) {
  echo 'Unexpected phpunit version: got ' . $got . ', want ' . $want;
} else {
  echo 'PASS';
}
