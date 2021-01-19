package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bitfield/script"
	"github.com/itchyny/gojq"
)

func JQ(query string) func(in *script.Pipe) *script.Pipe {
	return func(in *script.Pipe) *script.Pipe {
		q, err := gojq.Parse(query)
		if err != nil {
			return in.WithError(err)
		}
		var input map[string]interface{}
		json.NewDecoder(in).Decode(&input)
		output := bytes.NewBuffer(nil)
		iter := q.Run(input) // or query.RunWithContext
		for {
			v, ok := iter.Next()
			if !ok {
				return in.WithReader(output)
			}
			if err, ok := v.(error); ok {
				return in.WithError(err).WithReader(output)
			}
			fmt.Fprintf(output, "%v\n", v)
		}
		return in.WithReader(output)
	}
}

// Get all the slide titles from the httpsbin JSON demo and count the uniq titles.
func main() {
	// Golang only with `JQ` function
	script.HTTP("https://httpbin.org/json").
		ExternalFilter(JQ(".slideshow.slides[].title")).
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
