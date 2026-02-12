package main

import (
	"fmt"
	"os"
)

var commands = map[string]func(args []string){
	"apply": cmdApply,
	"run":   cmdRun,
	"steer": cmdSteer,
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
	fmt.Fprintf(os.Stderr, "usage: gcluster <command> [args...]\n\ncommands:\n  apply   Apply a cluster configuration\n  run     Run a cluster\n  steer   Steer a running cluster\n")
	os.Exit(1)
}

func cmdApply(args []string) {
	fmt.Println("gcluster apply: not implemented yet")
}

func cmdRun(args []string) {
	fmt.Println("gcluster run: not implemented yet")
}

func cmdSteer(args []string) {
	fmt.Println("gcluster steer: not implemented yet")
}
