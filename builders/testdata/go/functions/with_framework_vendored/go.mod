module example.com/myfunc

replace example.com/htmlreturn => ./local/example.com/htmlreturn

require (
  github.com/GoogleCloudPlatform/functions-framework-go v1.2.0
  example.com/htmlreturn v1.0.0
)

