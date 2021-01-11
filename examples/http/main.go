package main

import (
	"github.com/bitfield/script"
	"log"
	"net/http"
)

// This program reads the content of the files given as arguements and post their contents to the httpbin.org service.
// This service response with the oberserved request. This program checks if the response is 200 (HTTP OK) and prints
// the content of the body to STDOUT.
func main() {
	req, err := script.Args().Concat().HTTPRequest(http.MethodPost, "https://httpbin.org/post")
	if err != nil {
		log.Fatal(err)
	}
	script.HTTP(req, script.AssertingHTTPProcessor(http.StatusOK)).Stdout()
}
