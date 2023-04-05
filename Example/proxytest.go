package main

import (
	"fmt"
	"os"

	"github.com/rmasci/script"
)

func main() {
	os.Setenv("https_proxy", "")
	os.Setenv("http_proxy", "")
	//curl -s 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net' -H Metadata:true
	script.Get("http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fvault.azure.net", "Metadata:true").JQ(".access_token").Stdout()
	fmt.Println("")
}
