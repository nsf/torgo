package main

import "os"
import "github.com/nsf/torgo/view"
import "github.com/nsf/torgo/make"
import "github.com/nsf/torgo/client"
import "runtime"
import "fmt"

const main_help = `This is a set of BitTorrent tools, which currently includes:

torgo make:
    Utility for making .torrent files. Type "torgo make -h" for usage info.

torgo view:
    Torrent file viewer. Type "torgo view -h" for usage info.

torgo client:
    Fully featured BitTorrent client. Not implemented yet.
`

func print_version() {
	fmt.Printf("TorGo version 0.1, compiled with %s (Go version: %s)\n",
		runtime.Compiler, runtime.Version())
}

func dispatch_command() {
	switch os.Args[1] {
	case "make":
		make.Tool()
	case "view":
		view.Tool()
	case "client":
		client.Tool()
	case "-h", "--help", "help":
		print_version()
		fmt.Print(main_help)
	case "-v", "--version", "version":
		print_version()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s", os.Args[1])
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) == 1 {
		client.Tool()
	}
	dispatch_command()
}