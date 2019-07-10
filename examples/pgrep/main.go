/* Get pids for processes with name "ssh"
 */
package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.Processes().Match("ssh").Stdout()
}
