module example.com/hello

go 1.13

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.7.0
	rsc.io/quote v1.5.2
)

replace rsc.io/quote => ./quote-forked
