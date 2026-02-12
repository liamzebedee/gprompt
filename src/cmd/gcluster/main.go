package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"p2p/cluster"
	"p2p/parser"
	"p2p/registry"
	"p2p/sexp"
	"p2p/stdlib"
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

// cmdMaster starts the cluster control plane: loads persisted state,
// starts the TCP server, and waits for SIGINT/SIGTERM to shut down.
func cmdMaster(args []string) {
	addr := cluster.DefaultAddr
	statePath := cluster.DefaultStatePath()

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "--addr requires an argument\n")
				os.Exit(1)
			}
			addr = args[i+1]
			i++
		case "--state":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "--state requires an argument\n")
				os.Exit(1)
			}
			statePath = args[i+1]
			i++
		}
	}

	// Create store and load persisted state
	store := cluster.NewStore()
	cluster.LoadState(store, statePath)

	// Create and start server
	srv := cluster.NewServer(store, addr)

	// Handle shutdown signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")

		// Persist state before exit
		if err := cluster.SaveState(store, statePath); err != nil {
			log.Printf("warning: failed to save state: %v", err)
		} else {
			log.Printf("state saved to %s", statePath)
		}

		srv.Stop()
	}()

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// cmdApply parses a .p file, extracts agent- prefixed definitions,
// compiles them to S-expressions, computes stable IDs, and sends
// them to the master. Prints a summary of what changed.
func cmdApply(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: gcluster apply <file.p>\n")
		os.Exit(1)
	}

	addr := cluster.DefaultAddr
	filename := ""

	// Parse flags and positional args
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "--addr requires an argument\n")
				os.Exit(1)
			}
			addr = args[i+1]
			i++
		default:
			if filename == "" {
				filename = args[i]
			}
		}
	}

	if filename == "" {
		fmt.Fprintf(os.Stderr, "usage: gcluster apply <file.p>\n")
		os.Exit(1)
	}

	// Parse the .p file
	nodes, err := parser.Parse(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	// Build registry with stdlib and all methods (agents reference non-agent methods)
	reg := registry.New()
	loadStdlib(reg, filename)

	fileDir := filepath.Dir(filename)
	for _, node := range nodes {
		switch node.Type {
		case parser.NodeMethodDef:
			reg.Register(node.Name, node.Params, node.Body)
		case parser.NodeImport:
			importPath := resolveImport(node.ImportPath, fileDir)
			importNodes, err := parser.Parse(importPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "import error (%s): %v\n", node.ImportPath, err)
				os.Exit(1)
			}
			for _, n := range importNodes {
				if n.Type == parser.NodeMethodDef {
					reg.Register(n.Name, n.Params, n.Body)
				}
			}
		}
	}

	// Extract agent- prefixed definitions and compile to AgentDefs
	var agentDefs []cluster.AgentDef
	for _, node := range nodes {
		if node.Type != parser.NodeMethodDef {
			continue
		}
		if !strings.HasPrefix(node.Name, "agent-") {
			continue
		}

		// Emit S-expression for this agent definition
		sexpr := sexp.EmitProgram(nodes, reg, node.Name)
		if sexpr == "" {
			fmt.Fprintf(os.Stderr, "error: could not compile agent %q to S-expression\n", node.Name)
			os.Exit(1)
		}

		agentName := strings.TrimPrefix(node.Name, "agent-")
		stableID := sexp.StableID(sexpr)

		agentDefs = append(agentDefs, cluster.AgentDef{
			Name:       agentName,
			Definition: sexpr,
			ID:         stableID,
		})
	}

	if len(agentDefs) == 0 {
		fmt.Println("0 agents applied (no agent- definitions found)")
		return
	}

	// Connect to master and send apply request
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to master at %s — is `gcluster master` running?\n", addr)
		os.Exit(1)
	}
	defer conn.Close()

	// Send apply_request
	env, err := cluster.NewEnvelope(cluster.MsgApplyRequest, cluster.ApplyRequest{Agents: agentDefs})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	data, err := json.Marshal(env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "error sending to master: %v\n", err)
		os.Exit(1)
	}

	// Read apply_response
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		fmt.Fprintf(os.Stderr, "error: no response from master\n")
		os.Exit(1)
	}

	var respEnv cluster.Envelope
	if err := json.Unmarshal(scanner.Bytes(), &respEnv); err != nil {
		fmt.Fprintf(os.Stderr, "error: malformed response: %v\n", err)
		os.Exit(1)
	}

	var resp cluster.ApplyResponse
	if err := respEnv.DecodePayload(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "error from master: %s\n", resp.Error)
		os.Exit(1)
	}

	// Print summary
	printApplySummary(resp.Summary)
}

func printApplySummary(s cluster.ApplySummary) {
	total := len(s.Created) + len(s.Updated) + len(s.Unchanged)
	fmt.Printf("%d agent(s) applied: %d created, %d updated, %d unchanged\n",
		total, len(s.Created), len(s.Updated), len(s.Unchanged))

	for _, name := range s.Created {
		fmt.Printf("  + %s (created)\n", name)
	}
	for _, name := range s.Updated {
		fmt.Printf("  ~ %s (updated)\n", name)
	}
	for _, name := range s.Unchanged {
		fmt.Printf("  = %s (unchanged)\n", name)
	}
}

func cmdSteer(args []string) {
	fmt.Println("gcluster steer: not implemented yet")
}

// loadStdlib loads the standard library into the registry, searching
// disk first then falling back to the embedded copy.
func loadStdlib(reg *registry.Registry, inputFile string) {
	inputDir := filepath.Dir(inputFile)
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	paths := []string{
		filepath.Join(inputDir, "stdlib.p"),
		"stdlib.p",
		filepath.Join(exeDir, "stdlib.p"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			nodes, err := parser.Parse(p)
			if err != nil {
				continue
			}
			for _, n := range nodes {
				if n.Type == parser.NodeMethodDef {
					reg.Register(n.Name, n.Params, n.Body)
				}
			}
			return
		}
	}

	// Fallback: embedded stdlib
	nodes, err := parser.ParseString(stdlib.Source)
	if err != nil {
		return
	}
	for _, n := range nodes {
		if n.Type == parser.NodeMethodDef {
			reg.Register(n.Name, n.Params, n.Body)
		}
	}
}

func resolveImport(importPath string, baseDir string) string {
	if filepath.IsAbs(importPath) {
		return importPath
	}
	return filepath.Join(baseDir, importPath)
}
