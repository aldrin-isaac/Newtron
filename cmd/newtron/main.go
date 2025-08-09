// File: cmd/newtron/main.go
// Chat Session: newtron-20250706-01
// Session Timestamp: 2025-07-06T15:24:00-07:00
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"newtron/pkg/device"
	"newtron/pkg/intent" // Import the intent package

	"golang.org/x/term"
)

const version = "1.0.2 (gNMI-backend-init)"

func main() {
	opts := parseFlags()
	if opts.showHelp {
		printHelp()
		return
	}

	fmt.Println()
	fmt.Printf("Newtron v%s\n\n", version)
	// ... (menu help text) ...

	nodename := getNodeName(opts.jsonInputFile)
	if nodename == "" {
		fmt.Println("\n>> No node name provided. Exiting.")
		os.Exit(1)
	}

	// --- Intent Loading and Resolution ---
	fmt.Println("  [Main] Loading and resolving intent...")
	globalIntent, err := intent.LoadGlobalIntent("./pkg/intent") // Assuming YAMLs are here
	if err != nil {
		fmt.Printf(">> FATAL: Could not load global intent: %v\n", err)
		os.Exit(1)
	}

	resolvedIntent, err := globalIntent.Resolve(nodename)
	if err != nil {
		fmt.Printf(">> FATAL: Could not resolve intent for device '%s': %v\n", nodename, err)
		os.Exit(1)
	}
	fmt.Println("  [Main] Intent loaded successfully.")
	// --- End Intent Loading ---

	user, pass, err := askLoginInfo()
	if err != nil {
		fmt.Printf("\n>> Error getting credentials: %v\n", err)
		os.Exit(1)
	}

	// Create a new device instance with the resolved intent
	d, err := device.NewDevice(nodename, resolvedIntent)
	if err != nil {
		fmt.Printf(">> Failed to initialize device: %v\n", err)
		os.Exit(1)
	}

	// Connect to the device using the backend API
	err = d.Connect(user, pass)
	if err != nil {
		fmt.Printf(">> Failed to connect to %s (%s): %v\n", nodename, d.Intent.MgmtIP, err)
		os.Exit(1)
	}
	defer d.Close() // Ensure connection is closed on exit

	fmt.Printf("Successfully connected to %s\n", d.Node.Name)

	// Start the main interactive loop
	menuCard(d)

	fmt.Println("\n--- Exiting Newtron ---")
}

// --- Flag Parsing and Help Functions ---

type options struct {
	showHelp      bool
	jsonInputFile string
}

func parseFlags() *options {
	opts := &options{}
	flag.BoolVar(&opts.showHelp, "?", false, "Show help message")
	flag.BoolVar(&opts.showHelp, "h", false, "Show help message")
	flag.StringVar(&opts.jsonInputFile, "f", "", "Use JSON input file or node profile name.")
	flag.Parse()
	return opts
}

func printHelp() {
	fmt.Println("\n  syntax: newtron [-h] [-f <file>] [node-name]")
	fmt.Println("\n    -h, ?  : Show this help message.")
	fmt.Println("    -f     : Use json input file or node profile name.")
	fmt.Println()
}

func getNodeName(jsonFile string) string {
	if jsonFile != "" {
		return strings.TrimSuffix(jsonFile, ".json")
	}
	if flag.NArg() > 0 {
		return flag.Arg(0)
	}
	fmt.Print("Node: ")
	return readInput()
}

func askLoginInfo() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	user, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	fmt.Print("Password: ")
	bytePass, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", err
	}
	fmt.Println()
	return strings.TrimSpace(user), string(bytePass), nil
}

