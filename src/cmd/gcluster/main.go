package main

import (
	"fmt"
	"os"
)

var commands = map[string]func(args []string){
	"apply":  cmdApply,
	"master": cmdMaster,
	"steer":  cmdSteer,
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
	}

	cmd(os.Args[2:])
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gcluster <command> [args...]\n\ncommands:\n  apply    Apply agent definitions from a .p file\n  master   Start the cluster control plane\n  steer    Open the steering TUI\n")
	os.Exit(1)
}

func cmdApply(args []string) {
	fmt.Println("gcluster apply: not implemented yet")
}

func cmdMaster(args []string) {
	fmt.Println("gcluster master: not implemented yet")
}

func cmdSteer(args []string) {
	fmt.Println("gcluster steer: not implemented yet")
}
