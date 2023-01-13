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

package hello;

import com.sun.jna.Library;
import com.sun.jna.Native;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

@Path("/")
/** PASS text handler */
public class Hello {
  public interface CLibrary extends Library {
    // Loading libjpeg native library to test JNI
    CLibrary INSTANCE =
        (CLibrary) Native.loadLibrary("/usr/lib/x86_64-linux-gnu/libjpeg.so.8", CLibrary.class);

    void printf(String format, Object... args);
  }

  @GET
  @Produces(MediaType.TEXT_PLAIN)
  public String pass() {
    try {
      CLibrary.INSTANCE.printf("Hello, World\n");
    } catch (Exception e) {
      return "FAIL";
    }
    return "PASS";
  }
}
