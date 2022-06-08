package java11_compat_webapp;

import java.io.IOException;
import javax.servlet.annotation.WebServlet;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

/** Simplest of AppEngine-Java servlets, yielding a simple Hello World response. */
@WebServlet(
    name = "hello",
    urlPatterns = {"/*"})
public class HelloServlet extends HttpServlet {
  @Override
  public void doGet(HttpServletRequest request, HttpServletResponse response) throws IOException {
    response.setContentType("text/plain");
    response.setCharacterEncoding("UTF-8");
    response.getWriter().println("PASS");
  }
}
