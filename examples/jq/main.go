package main

import (
	"github.com/bitfield/script"
)

// Get all the slide titles from the httpsbin JSON demo and count the uniq titles.
func main() {
	// Golang only with `JQ` function
	script.HTTP("https://httpbin.org/json").
		JQ(".slideshow.slides[].title").
		Freq().
		Stdout()

	// Call the jq binary. This requires the jq binary to be installed on our system
	// if we want to generate a portable script this can be a hassle since we can not
	// specify the jq binary as dependency for different platforms (Windows, Linux, MacOS, *BSD)
	script.HTTP("https://httpbin.org/json").
		Exec("jq -r '.slideshow.slides[].title'").
		Freq().
		Stdout()
}
