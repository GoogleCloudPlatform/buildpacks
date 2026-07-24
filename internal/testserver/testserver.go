# -*- coding: utf-8 -*-
"""
testserver.py - Utility functions for stubbing HTTP requests in tests.

Copyright 2023 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

from typing import Optional
from fastapi import FastAPI, Response
from pydantic import BaseModel

class Config(BaseModel):
    http_status: int
    response_file: str = ""
    response_json: str = ""
    mock_url: Optional[str] = None
    handler: Optional[FastAPI] = None

class Option:
    def __init__(self, config: Config):
        self.config = config

    @staticmethod
    def with_status(http_status: int) -> "Option":
        return Option(
            Config(
                http_status=http_status,
            )
        )

    @staticmethod
    def with_json(json: str) -> "Option":
        return Option(
            Config(
                response_json=json,
            )
        )

    @staticmethod
    def with_file(path: str) -> "Option":
        return Option(
            Config(
                response_file=path,
            )
        )

    @staticmethod
    def with_mock_url(url: Optional[str]) -> "Option":
        return Option(
            Config(
                mock_url=url,
            )
        )

    @staticmethod
    def with_handler(handler: FastAPI) -> "Option":
        return Option(
            Config(
                handler=handler,
            )
        )

async def new_test_server(t, opts):
    config = Config()
    for opt in opts:
        opt.config = config

    app = FastAPI()

    if config.handler is not None:
        app.include_router(config.handler)
    else:
        async def stub_handler(request: Request):
            if config.http_status != 0:
                return Response(status_code=config.http_status)

            if config.response_file != "":
                with open(config.response_file, "rb") as f:
                    content = await f.read()
                    headers = {"Content-Type": "application/octet-stream"}
                    return Response(content=content, media_type=headers["Content-Type"])

            if config.response_json != "":
                response = JSONResponse(
                    content={"message": config.response_json},
                    media_type="application/json",
                )
                return response

        app.add_route("/", stub_handler)

    server = await asyncio.start_server(app.serve, "localhost", 0)
    address = server.sockets[0].getsockname()
    t.addCleanup(server.close)

    if config.mock_url is not None:
        orig_val = config.mock_url
        t.addCleanup(lambda: setattr(config.mock_url, orig_val))
        config.mock_url = f"{address[0]}:{address[1]}?p1=%s&p2=%s&p3=%s"

    return server.sockets[0].getsockname()
