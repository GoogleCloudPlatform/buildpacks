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

$endpoints = [
  'custom',
  'version',
  'env',
];

// Get the request URL and parse it.
$url = $_SERVER['REQUEST_URI'];
$parsed_urls = parse_url($url);
$path = $parsed_urls['path'];

// Figure out which script to load.
$f = '';
foreach ($endpoints as $ep) {
  if ($path == '/' . $ep) {
    $f = $ep . '.php';
  }
}

if ($f == '') {
  $f = 'hello.php';
}

// Load the script.
error_log(sprintf('test router: requiring %s', $f));
require_once $f;
