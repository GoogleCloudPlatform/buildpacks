<?php
// Copyright 2022 Google LLC
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

$gotIniPath = php_ini_loaded_file();
$wantIniPath = '/layers/google.php.runtime/php/etc/php.ini';

if (!$gotIniPath) {
  echo 'FAIL. php.ini file is not loaded';
  return;
}

if ($gotIniPath != $wantIniPath) {
  echo 'FAIL. Unexpected php.ini path: got ' . $gotIniPath . ', want ' . $wantIniPath;
  return;
}

echo 'PASS_PHP_INI';
