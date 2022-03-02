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

if (!isset($_GET['want'])) {
  echo 'FAIL: ?want must not be empty';
  return;
}

$got = phpversion();
if(!isset($got)) {
  echo 'FAIL: phpversion() must not be empty';
  return;
}

if ($got != $_GET['want']) {
  echo sprintf('FAIL: phpversion() got "%s", want "%s"', $got, $_GET['want']);
  return;
}

echo 'PASS_VERSION';
