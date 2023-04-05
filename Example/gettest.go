package main

import (
	"github.com/rmasci/script"
)

func main() {
	script.Get("https://httpbin.org/headers", "Metadata:true", "User-Agent:Go-http-client/bitfield-script", "one:one", "Two:two").Stdout()
}
