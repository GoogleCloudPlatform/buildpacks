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

import static java.nio.charset.StandardCharsets.UTF_8;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.util.Map;
import java.util.HashMap;

/** Simple Java Web Server */
public class Main {
  public static void main(String[] args) throws IOException {
    // Create an instance of HttpServer bound to port defined by the
    // PORT environment variable when present, otherwise on 8080.
    int port = Integer.parseInt(System.getenv().getOrDefault("PORT", "8080"));
    HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
    server.createContext(
        "/",
        (HttpExchange t) -> {
          byte[] response = "PASS".getBytes(UTF_8);
          t.sendResponseHeaders(200, response.length);
          try (OutputStream os = t.getResponseBody()) {
            os.write(response);
          }
        });

    server.createContext(
        "/version",
        (HttpExchange t) -> {
          String wantVersion = queryToMap(t.getRequestURI().getQuery()).get("want");
          String got = Runtime.version().toString();
          byte[] response;
          if (wantVersion.equals(got)) {
            response = "PASS".getBytes(UTF_8);
          } else {
            response = "FAIL".getBytes(UTF_8);
          }
          t.sendResponseHeaders(200, response.length);
          try (OutputStream os = t.getResponseBody()) {
            os.write(response);
          }
        });
    server.setExecutor(null);
    server.start();
  }

  private Main() {}

  /**
   * returns the url parameters in a map
   *
   * @param query
   * @return map
   */
  private static Map<String, String> queryToMap(String query) {
    Map<String, String> result = new HashMap<String, String>();
    for (String param : query.split("&")) {
      String pair[] = param.split("=");
      if (pair.length > 1) {
        result.put(pair[0], pair[1]);
      } else {
        result.put(pair[0], "");
      }
    }
    return result;
  }
}
