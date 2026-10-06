/*
 * Copyright 2020 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 */
package functions;

import com.google.cloud.functions.HttpFunction;
import com.google.cloud.functions.HttpRequest;
import com.google.cloud.functions.HttpResponse;
import com.google.escapevelocity.Template;
import java.io.IOException;
import java.io.StringReader;
import java.util.Map;

/** A function that just prints out PASS. */
public class HelloWorld implements HttpFunction {
  private static final String TEMPLATE_TEXT = "$pass";

  @Override
  public void service(HttpRequest request, HttpResponse response) throws IOException {
    // This elaborate way of getting the string "PASS" proves that functions can have dependencies
    // that are correctly present at runtime.
    Template template = Template.parseFrom(new StringReader(TEMPLATE_TEXT));
    String text = template.evaluate(Map.of("pass", "PASS"));
    response.getWriter().write(text);
  }
}
