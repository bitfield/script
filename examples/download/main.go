package main

import (
	"encoding/base64"
	"github.com/bitfield/script"
	"log"
	"net/http"
)

func basicAuth(username, password string) http.Header {

	auth := username + ":" + password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	h := http.Header{}
	h.Set("Authorization", "Basic "+enc)
	return h
}

// This programm downloads a file from a basic auth protected server and saves it to the local file sytem
// Example:
// go run examples/download/main.go && cat  download.txt
//{
//  "authenticated": true,
//  "user": "admin"
//}
func main() {
	user, password := "admin", "supersecret"
	_, err := script.HTTP("https://httpbin.org/basic-auth/admin/supersecret").
		WithHeader(basicAuth(user, password)).
		WriteFile("download.txt")
	if err != nil {
		log.Fatal(err)
	}
}
