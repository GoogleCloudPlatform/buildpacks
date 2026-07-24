// Copyright 2025 Google LLC
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

// This script is used to generate bytecode cache for a Node.js application.
// It is intended to be run by the Node.js functions-framework buildpack.
//
// The script takes the absolute path to the user's entrypoint as an argument.
// It then enables bytecode caching, requires the entrypoint, and flushes the
// cache to disk.
//
// The script is designed to be run in a Node.js environment with the
// functions-framework installed. Any dependencies of the entrypoint must be
// installed in the Node.js environment.

const { enableCompileCache, flushCompileCache } = require('node:module');
const path = require('node:path');
const fs = require('node:fs');

// The buildpack passes the absolute path to the user's entrypoint as the first argument.
const entrypoint = process.argv[2];
if (!entrypoint) {
  console.error('Internal error: Application entrypoint not provided to cache generator.');
  process.exit(1);
}

// The buildpack passes the cache directory name as the second argument.
const cacheDirName = process.argv[3];
if (!cacheDirName) {
  console.error('Internal error: Cache directory name not provided to cache generator.');
  process.exit(1);
}


console.log('--- Starting bytecode cache generation ---');

try {
  const cachePath = path.join(process.cwd(), cacheDirName);

  if (fs.existsSync(cachePath)) {
    fs.rmSync(cachePath, { recursive: true, force: true });
  }
  fs.mkdirSync(cachePath);

  enableCompileCache(cachePath);

  console.log('Requiring application entrypoint to trigger compilation...');
  require(entrypoint);

  console.log('Flushing compile cache to disk...');
  flushCompileCache();

  console.log('--- Cache generation complete. ---');
} catch (error) {
  console.error(' Warning: Error during cache generation, build will continue without it.', error);
  process.exit(1);
}
