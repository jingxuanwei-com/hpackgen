package main

import (
	"1szt/hpackgen"
	"1szt/motd"
)

func main() {
	motd.Run()

	hpackgen.Run()

	select {}
}
