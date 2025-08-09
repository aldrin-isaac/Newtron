// File: cmd/newtron/menu_bridg.go
// Chat Session: 20240706_1244_NewtronProject
// Session Timestamp: 2025-07-06T14:10:00-07:00
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"newtron/pkg/device"
)

func menuBridgeInterfaceDetail(d *device.Device, port *device.Port) {
	for {
		showBridgeInterfaceInfo(port)
		fmt.Println("\nActions:")
		fmt.Println(" 1. Add Bridge Interface")
		fmt.Println(" 2. Remove Bridge Interface")
		fmt.Println(" 3. Modify VLAN Membership")
		printMenuLine("=", port.ParentCard.ParentNode.Name)

		fmt.Print("Enter Selection: ")
		input := readInput()
		switch input {
		case "x":
			os.Exit(0)
		case "q":
			return
		case "":
			continue
		case "1":
			mode, vlans, err := askBridgeInterfaceInfo(port)
			if err != nil {
				fmt.Printf(">> Error: %v\n", err)
				continue
			}
			fmt.Printf("Create %s bridge-interface on %s? (y/n) [n]: ", mode, port.IfName)
			if readInput() == "y" {
				d.ConfigBridgeInterface(port, "add", mode, vlans)
			}
		default:
			fmt.Println(">> Action not implemented yet.")
		}
	}
}

func askBridgeInterfaceInfo(port *device.Port) (string, []string, error) {
	fmt.Print("Enter Interface Bridge Mode (access/trunk) [access]: ")
	mode := readInput()
	if mode == "" {
		mode = "access"
	}
	if mode != "access" && mode != "trunk" {
		return "", nil, errors.New("invalid mode")
	}

	fmt.Print("Enter VLAN IDs (comma-separated): ")
	vlanInput := readInput()
	vlans := strings.Split(vlanInput, ",")
	for i := range vlans {
		vlans[i] = strings.TrimSpace(vlans[i])
	}
	return mode, vlans, nil
}

func showBridgeInterfaceInfo(port *device.Port) {
	fmt.Println()
	printMenuLine("=", fmt.Sprintf(" Bridge Interface / %s ", port.IfName))
	// Mock display
	fmt.Println("  Mode: Trunk")
	fmt.Println("  VLANs: 100, 200, 300")
}

