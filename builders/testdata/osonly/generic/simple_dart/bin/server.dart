/**
 * Copyright 2022 Google LLC
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

// Simple Dart web service using shelf that always responds with "PASS".

import 'dart:io';

import 'package:shelf/shelf.dart';
import 'package:shelf/shelf_io.dart' as shelf_io;

Future main() async {
  final port = int.parse(Platform.environment['PORT'] ?? '8080');
  await shelf_io.serve(
    _helloWorldHandler,
    InternetAddress.anyIPv4,
    port,
  );
}

Response _helloWorldHandler(Request request) => Response.ok('PASS');
