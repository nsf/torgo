package main

import "os"

func dispatch_command() {
	switch os.Args[1] {
	case "make":
		make_tool()
	case "view":
		view_tool()
	case "client":
		client_tool()
	case "-h", "--help", "help":
		// TODO
	default:
		println("unknown command: " + os.Args[1])
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) == 1 {
		client_tool()
	}
	dispatch_command()
}