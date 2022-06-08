import com.github.jengelman.gradle.plugins.shadow.tasks.ShadowJar

plugins {
    application
    kotlin("jvm") version "1.3.21"
    kotlin("kapt") version "1.3.21"
    id("com.github.johnrengelman.shadow") version "4.0.4"
}

repositories {
    mavenLocal()
    mavenCentral()
    jcenter()
}

dependencies {
    compile(kotlin("stdlib"))

    compile("io.micronaut:micronaut-runtime:1.1.0")
    compile("io.micronaut:micronaut-http-client:1.1.0")
    compile("io.micronaut:micronaut-http-server-netty:1.1.0")
    compile("ch.qos.logback:logback-classic:1.2.3")

    kapt("io.micronaut:micronaut-inject-java:1.1.0")
    kapt("io.micronaut:micronaut-validation:1.1.0")
}

application {
    mainClassName = "hello.WebAppKt"
}

tasks.withType<ShadowJar> {
    mergeServiceFiles()
}

tasks.create("stage") {
    dependsOn("shadowJar")
}
