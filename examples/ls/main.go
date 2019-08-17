/*Example for ListFiles function.
`ls` binary will list content of directory provided via args

Example of runing this binary at `examples/` directory:
./examples/ls/ls examples/
examples/cat
examples/cat2
examples/echo
examples/grep

*/
package main

import (
	"os"
	"regexp"

	"github.com/bitfield/script"
)

func main() {
	var listPath string
	var err error
	if len(os.Args) > 1 {
		listPath = os.Args[1]
	} else {
		listPath, err = os.Getwd()
	}
	if err != nil {
		panic(err)
	}
	//dont show files starting with dot
	reg, err := regexp.Compile("(/[^/])*/\\.[^/]*$")
	if err != nil {
		panic(err)
	}
	script.ListFiles(listPath).RejectRegexp(reg).Stdout()
}
