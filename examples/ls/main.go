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

	"github.com/bitfield/script"
)

func main() {
	script.ListFiles(os.Args[1]).Stdout()
}
