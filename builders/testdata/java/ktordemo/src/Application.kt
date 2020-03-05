package com.google

import io.ktor.application.*
import io.ktor.client.*
import io.ktor.client.engine.jetty.*
import io.ktor.html.*
import io.ktor.http.*
import io.ktor.request.*
import io.ktor.response.*
import io.ktor.routing.*
import kotlinx.css.*
import kotlinx.html.*

fun main(args: Array<String>): Unit = io.ktor.server.netty.EngineMain.main(args)

@Suppress("unused") // Referenced in application.conf
@kotlin.jvm.JvmOverloads
fun Application.module(testing: Boolean = false) {
  val client = HttpClient(Jetty) {
  }

  routing {
    get("/") {
      call.respondText("PASS", contentType = ContentType.Text.Plain)
    }

    get("/html-dsl") {
      call.respondHtml {
        body {
          h1 { +"HTML" }
          ul {
            for (n in 1..10) {
              li { +"$n" }
            }
          }
        }
      }
    }

    get("/styles.css") {
      call.respondCss {
        body {
          backgroundColor = Color.red
        }
        p {
          fontSize = 2.em
        }
        rule("p.myclass") {
          color = Color.blue
        }
      }
    }
  }
}

fun FlowOrMetaDataContent.styleCss(builder: CSSBuilder.() -> Unit) {
  style(type = ContentType.Text.CSS.toString()) {
    +CSSBuilder().apply(builder).toString()
  }
}

fun CommonAttributeGroupFacade.style(builder: CSSBuilder.() -> Unit) {
  this.style = CSSBuilder().apply(builder).toString().trim()
}

suspend inline fun ApplicationCall.respondCss(builder: CSSBuilder.() -> Unit) {
  this.respondText(CSSBuilder().apply(builder).toString(), ContentType.Text.CSS)
}
