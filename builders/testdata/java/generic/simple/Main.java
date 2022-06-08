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
import java.util.HashMap;
import java.util.Map;

/** Toy server for acceptance testing purposes */
@SuppressWarnings("DefaultPackage")
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
          String got = Integer.toString(getVersion());
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

  /**
   * Returns the Java version at runtime. java.version is a system property that exists in all JVMs.
   * There are two possible formats for it:
   *
   * <p>Java 8 or lower: 1.6.0_23, 1.7.0, 1.7.0_80, 1.8.0_211, Java 9 or higher: 9.0.1, 11.0.4,
   * 12.0.1
   *
   * @return int Java version
   */
  private static int getVersion() {
    String version = System.getProperty("java.version");
    if (version.startsWith("1.")) {
      version = version.substring(2, 3);
    } else {
      int dot = version.indexOf(".");
      if (dot != -1) {
        version = version.substring(0, dot);
      }
    }
    return Integer.parseInt(version);
  }
}
