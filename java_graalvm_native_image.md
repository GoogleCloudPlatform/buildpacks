# (Experimental) Java GraalVM Native Image Support

## Table of Contents

* [Enabling GraalVM Native Image](#enabling-graalvm-native-image)
* [Memory Requirements](#memory-requirements)
* [Supported Project Types](#supported-project-types)
* [Buildpacks Environment Variables](#buildpacks-environment-variables)

## Enabling GraalVM Native Image

Google Cloud Buildpacks support GraalVM [ahead-of-time native image](https://www.graalvm.org/reference-manual/native-image/) compilation for Java. (Note the "native image" here refers to a compiled native binary, not a container image.) To enable, set the `GOOGLE_JAVA_USE_NATIVE_IMAGE` environment variable to true.

```
pack build my-image --builder gcr.io/buildpacks/builder:v1 --env GOOGLE_JAVA_USE_NATIVE_IMAGE=true
```

## Memory Requirements

Native image compilation [requires a lot of RAM](https://repo.spring.io/milestone/org/springframework/experimental/spring-graalvm-native-docs/0.7.1/spring-graalvm-native-docs-0.7.1.zip!/reference/index.html#_out_of_memory_error_when_building_the_native_image). Insufficient memory can slow down compilation or cause a failure.

> `native-image` consumes a lot of RAM. We have most success on 32G RAM desktop machines. 16G is possible for smaller samples but 8G machines are likely to hit problems more often

Symptoms and error messages due to out-of-memory may not be obvious and include `native-image` hanging and JVM crashing. Some possible error messages:
```
Caused by: java.lang.OutOfMemoryError: GC overhead limit exceeded
```
```
Error: Image build request failed with exit status 137
```
```
Error: Image build request failed with exit status 1
```
```
Error: Image build request failed with exit status 134
```
```
Error: Image building with exit status 137
```

If you want to specify more than 14GB of memory, you may need to pass, say, [`-J-Xmx32G`](https://github.com/oracle/graal/issues/2130#issuecomment-593513004) to `native-image` (combined with [`--no-server`](https://github.com/oracle/graal/issues/2598) if you are on an old GraalVM version). See [Buildpacks Environment Variables](#buildpacks-environment-variables) for providing additional build arguments for Google Cloud Buildpacks.

If you run Buildpacks inside a Docker or a CI/CD environment, you may also need extra configuration for the platform to ensure sufficient memory. For example, to increase memory for Docker on Windows or Mac, see this [Stack Overflow answer](https://stackoverflow.com/questions/44533319/how-to-assign-more-memory-to-docker-container/44533437#44533437). For [Google Cloud Build](README.md#using-with-google-cloud-build) (`gcloud builds submit --pack ...`), you may want to add `--machine-type=e2-highcpu-32` ([32GB-RAM machine](https://cloud.google.com/build/pricing)).

## Supported Project Types

* Maven Projects: A Maven project where `./mvnw package` generates a standard executable JAR runnable by `java -jar`. (That is, a JAR whose `MANIFEST.MF` defines `Main-Class:` and (if applicable) `Class-Path:`).

   For a thin JAR that specifies dependencies with `Class-Path:`, often you will configure your `pom.xml` so that `./mvn package` copies the dependencies to the correct location relative to your executable JAR. For example, with `Class-Path: lib/my-dep-0.1.0.jar ...`,

   ```xml
   <build>
     <plugins>
       <plugin>
         <groupId>org.apache.maven.plugins</groupId>
         <artifactId>maven-jar-plugin</artifactId>
         <version>3.2.0</version>
         <configuration>
           <archive>
             <manifest>
               <mainClass>com.example.demo.Main</mainClass>
               <addClasspath>true</addClasspath>
               <classpathPrefix>lib/</classpathPrefix>
             </manifest>
           </archive>
         </configuration>
       </plugin>

       <plugin>
         <groupId>org.apache.maven.plugins</groupId>
         <artifactId>maven-dependency-plugin</artifactId>
         <version>3.2.0</version>
         <executions>
           <execution>
             <id>copy-dependencies</id>
             <phase>package</phase>
             <goals>
               <goal>copy-dependencies</goal>
             </goals>
             <configuration>
               <outputDirectory>${project.build.directory}/lib</outputDirectory>
             </configuration>
           </execution>
         </executions>
       </plugin>
       ...
     </plugins>
     ...
   </build>
   ```

* <a name="maven-plugin-profile"/>Maven Projects that define a GraalVM [`native-image-maven-plugin`](https://www.graalvm.org/reference-manual/native-image/NativeImageMavenPlugin/) under a [Maven profile](https://maven.apache.org/guides/introduction/introduction-to-profiles.html).

   Note that GraalVM [`native-maven-plugin`](https://github.com/graalvm/native-build-tools/blob/master/native-maven-plugin/README.md) is a successor to `native-image-maven-plugin`, which is not yet supported. (Also, don't be confused with another [`native-maven-plugin`](https://www.mojohaus.org/maven-native/native-maven-plugin/) for compiling C/C++.)

   Buildpacks simply trigger the first profile where the plugin is configured while running the `package` goal (for example, `./mvnw -Pnative package`). This assumes that the `native-image` goal of the plugin is bound to the [`package` Maven phase](https://maven.apache.org/guides/introduction/introduction-to-the-lifecycle.html#a-build-lifecycle-is-made-up-of-phases). For example,

   ```xml
   <profiles>
     <profile>
       <id>native</id>
       <build>
         <plugins>
           <plugin>
             <groupId>org.graalvm.nativeimage</groupId>
             <artifactId>native-image-maven-plugin</artifactId>
             <version>${graalvm.version}</version>
             <executions>
               <execution>
                 <id>native-image</id>
                 <goals>
                   <goal>naive-image</goal>
                 </goals>
                 <phase>package</phase>
               </execution>
             </executions>
             <configuration>
               ...
             </configuration>
               ...
           </plugin>
           ...
         </plugins>
       </build>
     </profile>
     ...
   </profiles>
   ```
* Spring Boot: Maven Projects that builds their fat JAR using the [`maven-spring-boot-plugin`](https://docs.spring.io/spring-boot/docs/current/maven-plugin/reference/htmlsingle/).

   The project should be configured with [Spring Native (beta)](https://github.com/spring-projects-experimental/spring-native). As a demo,
   1. Initialize a project with Spring Native (beta).

       ```
       curl https://start.spring.io/starter.tgz -d dependencies=web,native -d baseDir=spring-app | tar -xzvf -
       cd spring-app
       ```
   2. Run Buildpacks with the environment variable set to true.

       ```
       pack build my-image --builder gcr.io/buildpacks/builder:v1 --env GOOGLE_JAVA_USE_NATIVE_IMAGE=true
       ```
   3. Run the image and check the app at http://localhost:8080.

       ```
       docker run -p8080:8080 my-image
       ```

   Note that Google Cloud Buildpacks ignores [Spring's way of building an image](https://docs.spring.io/spring-native/docs/current/reference/htmlsingle/#_enable_native_image_support) via their Cloud Native Buildpacks support. Following their instructions is actually configuring `./mvnw spring-boot:build-image` to run [Paketo Buildpacks](https://paketo.io/). Therefore, their instructions to set Paketo-specific environment variables (for example, [`BP_NATIVE_IMAGE` and `BP_NATIVE_IMAGE_BUILD_ARGUMENTS`](https://paketo.io/docs/buildpacks/language-family-buildpacks/java-native-image/)) have no effect for Google Cloud Buildpacks.

## Buildpacks Environment Variables

* `GOOGLE_JAVA_USE_NATIVE_IMAGE`: `true` to enable experimental GraalVM native image compilation. Currently, only supported are Maven projects.

* `GOOGLE_JAVA_NATIVE_IMAGE_ARGS`: Additional build arguments to pass to the [`native-image`](https://www.graalvm.org/reference-manual/native-image/Options/#options-to-native-image-builder) generation tool.

   Note that, for [Maven projects that define a GraalVM `native-image-maven-plugin`](#maven-plugin-profile), this environment variable is ignored, as the entire work to generate a native image is delegated to the plugin. In this case, put your build arguments to the plugin configuration. For example,

   ```xml
   <plugin>
     <groupId>org.graalvm.nativeimage</groupId>
     <artifactId>native-image-maven-plugin</artifactId>
     <version>${graalvm.version}</version>
     ...
     <configuration>
       <buildArgs>
         --no-fallback --no-server
         -H:+StaticExecutableWithDynamicLibC
       </buildArgs>
     </configuration>
   </plugin>
   ```
