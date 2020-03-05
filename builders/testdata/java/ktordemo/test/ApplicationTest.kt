package com.google

import io.ktor.application.*
import io.ktor.client.*
import io.ktor.client.engine.jetty.*
import io.ktor.html.*
import io.ktor.http.*
import io.ktor.request.*
import io.ktor.response.*
import io.ktor.routing.*
import io.ktor.server.testing.*
import kotlin.test.*
import kotlinx.css.*
import kotlinx.html.*

class ApplicationTest {
  @Test
  fun testRoot() {
    withTestApplication({ module(testing = true) }) {
      handleRequest(HttpMethod.Get, "/").apply {
        assertEquals(HttpStatusCode.OK, response.status())
        assertEquals("PASS", response.content)
      }
    }
  }
}
