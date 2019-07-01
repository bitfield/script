/*
This program reads an Apache logfile in Common Log Format, like this:

212.205.21.11 - - [30/Jun/2019:17:06:15 +0000] "GET / HTTP/1.1" 200 2028 "https://example.com/ "Mozilla/5.0 (Linux; Android 8.0.0; FIG-LX1 Build/HUAWEIFIG-LX1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.156 Mobile Safari/537.36"

It extracts the first column of each line (the visitor IP address), counts the
frequency of each unique IP address in the log, and outputs the 10 most frequent
visitors in the log. Example output:

16 176.182.2.191
7 212.205.21.11
1 190.253.121.1
1 90.53.111.17

*/
package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.Stdin().Column(1).Freq().First(10).Stdout()
}
