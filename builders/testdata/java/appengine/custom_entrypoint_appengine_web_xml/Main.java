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
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.stream.Collectors;
import java.util.stream.Stream;

/** Simple Java Web Server which tests the presence of Jetty injected at build time. */
public class Main {
  public static void main(String[] args) throws IOException {
    // Create an instance of HttpServer bound to port defined by the
    // PORT environment variable when present, otherwise on 8080.
    int port = Integer.parseInt(System.getenv().getOrDefault("PORT", "8080"));
    HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
    server.createContext(
        "/",
        (HttpExchange t) -> {
          String commandOutput = getDirectoryListing("/layers/google.java.appengine/java_runtime/");
          String responseString = "PASS\n " + commandOutput;
          byte[] response = responseString.getBytes(UTF_8);
          t.sendResponseHeaders(200, response.length);
          try (OutputStream os = t.getResponseBody()) {
            os.write(response);
          }
        });
    server.setExecutor(null);
    server.start();
  }

  private static String getDirectoryListing(String path) {
    try {
      Path dir = Paths.get(path);
      try (Stream<Path> stream = Files.list(dir)) {
        return stream
            .map(p -> p.getFileName().toString())
            .sorted()
            .collect(Collectors.joining(";"));
      }
    } catch (Exception e) {
      return "Error listing directory: " + e.getMessage();
    }
  }

  private Main() {}
}
