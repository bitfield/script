package main

import (
	"github.com/bitfield/script"
	"log"
	"net/http"
	"regexp"
)

// This program reads the content of the first JSON file given as arguements and post their contents to the httpbin.org service.
// This service response with the oberserved request. The response is printed to STDOUT.
// A typical usecase of this pattern is for example in administrating something like elasticsearch. You want to push your
// index templates from you local file system to the cluster via the cluster REST API.
//
// Example:
/*
go run examples/http/main.go examples/http/test.json
{
  "args": {},
  "data": "{\n  \"policy\": {\n    \"attribute1\": \"value1\",\n    \"attribute2\": \"value2\"\n  }\n}",
  "files": {},
  "form": {},
  "headers": {
    "Accept-Encoding": "gzip",
    "Content-Length": "76",
    "Content-Type": "application/json",
    "Host": "httpbin.org",
    "Myorganisationsrequiredheader": "foo",
    "User-Agent": "Go-http-client/2.0",
    "X-Amzn-Trace-Id": "Root=1-600338b5-0aa55ba02f873b797bcd49b1"
  },
  "json": {
    "policy": {
      "attribute1": "value1",
      "attribute2": "value2"
    }
  },
  "origin": "1.2.3.4",
  "url": "https://httpbin.org/put"
}


*/
func main() {
	_, err := script.Args().
		MatchRegexp(regexp.MustCompile(".*\\.json")).
		First(1).
		Concat().
		HTTP("https://httpbin.org/put").
		WithHeader(http.Header{"Content-Type": []string{"application/json"}, "MyOrganisationsRequiredHeader": []string{"foo"}}).
		WithMethod(http.MethodPut).
		Stdout()
	if err != nil {
		log.Fatal(err)
	}
}
