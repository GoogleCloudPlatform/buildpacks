# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import asyncio
from typing import Optional

class Config:
    def __init__(self):
        self.http_status: int = 0
        self.response_file: str = ""
        self.response_json: str = ""
        self.mock_url: Optional[str] = None
        self.handler: Optional[asyncio.Future] = None

class Option:
    def __init__(self, func):
        self.func = func

    async def configure(self, config: Config) -> Config:
        return self.func(config)

class TestServer:
    def __init__(self, loop=None):
        if loop is None:
            loop = asyncio.get_event_loop()
        self.loop = loop
        self.config = Config()

    async def with_status(self, http_status: int) -> "TestServer":
        self.config.http_status = http_status
        return self

    async def with_json(self, json: str) -> "TestServer":
        self.config.response_json = json
        return self

    async def with_file(self, path: str) -> "TestServer":
        self.config.response_file = path
        return self

    async def with_mock_url(self, url: str) -> "TestServer":
        self.config.mock_url = url
        return self

    async def with_handler(self, handler: asyncio.Future) -> "TestServer":
        self.config.handler = handler
        return self

    async def new(self, t=None, opts=()) -> Optional[str]:
        if t is None:
            t = unittest.TestCase()
        options = [o.configure(self.config) for o in opts]
        config = self.config
        server = asyncio.Future()

        async def handler(rw, r):
            if config.handler is not None:
                await config.handler.send(rw, r)
                return
            if config.http_status != 0:
                rw.write_status(config.http_status)
            if config.response_file != "":
                await http.server.serve_file(rw, r, config.response_file)
                return
            if config.response_json != "":
                rw.headers["Content-Type"] = "application/json"
            if not await rw.write(config.response_json):
                t.fail(f"Sending stubbed HTTP response: {r.get_exception()}")

        self.loop.run_in_executor(None, lambda: server.set_result(
            asyncio.create_server(handler, '127.0.0.1', 0).serve_forever()
        ))
        address = self.loop.run_until_complete(server)
        t.addCleanup(self.loop.run_until_complete(self.server_close()))
        if config.mock_url is not None:
            original_value = config.mock_url
            t.addCleanup(lambda: setattr(config, "mock_url", original_value))
            config.mock_url = f"{address.getsockname()[0]}:{address.getsockname()[1]}?p1=%s&p2=%s&p3=%s"

        return server.result()

    async def close(self):
        self.loop.run_until_complete(self.server_close())

class TestServerOptions:
    @classmethod
    def with_status(cls, http_status: int) -> "TestServerOptions":
        return cls(lambda config: config.with_status(http_status))

    @classmethod
    def with_json(cls, json: str) -> "TestServerOptions":
        return cls(lambda config: config.with_json(json))

    @classmethod
    def with_file(cls, path: str) -> "TestServerOptions":
        return cls(lambda config: config.with_file(path))

    @classmethod
    def with_mock_url(cls, url: str) -> "TestServerOptions":
        return cls(lambda config: config.with_mock_url(url))

    @classmethod
    def with_handler(cls, handler: asyncio.Future) -> "TestServerOptions":
        return cls(lambda config: config.with_handler(handler))
