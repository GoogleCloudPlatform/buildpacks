package functions.jar;

import com.google.cloud.functions.HttpFunction;
import com.google.cloud.functions.HttpRequest;
import com.google.cloud.functions.HttpResponse;
import java.io.IOException;

/** Trivial function that just responds with {@code PASS}. */
public class HelloWorld implements HttpFunction {
  @Override public void service(HttpRequest request, HttpResponse response) throws IOException {
    response.getWriter().write("PASS");
  }
}
