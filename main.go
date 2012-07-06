package main

import "os"
import "github.com/nsf/torgo/view"
import "github.com/nsf/torgo/make"
import "github.com/nsf/torgo/client"

func dispatch_command() {
	switch os.Args[1] {
	case "make":
		make.Tool()
	case "view":
		view.Tool()
	case "client":
		client.Tool()
	case "-h", "--help", "help":
	default:
		println("unknown command: " + os.Args[1])
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) == 1 {
		client.Tool()
	}
	dispatch_command()
}