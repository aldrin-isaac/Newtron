// File: cmd/newtron/menu_port.go
// Chat Session: 20250706_1244_NewtronProject
// Session Timestamp: 2025-07-06T14:10:00-07:00
package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"newtron/pkg/device"
)

// menuPort is the second-level menu for selecting a port on a specific card.
func menuPort(d *device.Device, card *device.Card) {
	for {
		if err := d.LoadPorts(card); err != nil {
			fmt.Printf("Error loading ports: %v. Please try again.\n", err)
			continue
		}
		showPorts(card)
		fmt.Print("Enter Selection: ")
		input := readInput()

		switch input {
		case "x":
			os.Exit(0)
		case "q":
			return
		case "":
			continue
		default:
			selection, err := strconv.Atoi(input)
			if err != nil || selection < 1 || selection > len(card.Ports) {
				fmt.Println(">> Invalid selection.")
				continue
			}
			selectedPort := card.Ports[selection-1]

			// Decide which third-level menu to enter based on port type
			if card.PortType == "ae" {
				menuAEInterfaceLAGDetail(d, selectedPort)
			} else {
				menuInterfaceDetail(d, selectedPort)
			}
		}
	}
}

// showPorts displays the list of available ports on a card.
func showPorts(card *device.Card) {
	fmt.Println()
	printMenuLine("=", fmt.Sprintf(" Ports on Card %s ", card.ID))
	if len(card.Ports) == 0 {
		fmt.Println("  No ports found on this card.")
	}
	sort.Slice(card.Ports, func(i, j int) bool {
		return card.Ports[i].IfName < card.Ports[j].IfName
	})
	for i, port := range card.Ports {
		fmt.Printf(" %2d. %-15s Status: %s\n", i+1, port.IfName, port.OperStatus)
	}
	printMenuLine("=", card.ParentNode.Name)
}

// menuInterfaceDetail is the third-level menu for actions on a specific physical port.
func menuInterfaceDetail(d *device.Device, port *device.Port) {
	for {
		// Load details every time to get fresh state
		if err := d.LoadPortDetail(port); err != nil {
			fmt.Printf("Error loading port details: %v\n", err)
			return
		}
		showPortDetail(port)

		fmt.Println("\nActions:")
		fmt.Println(" 1. Configure Sub-Interfaces")
		fmt.Println(" 2. Configure Bridge Interface")
		fmt.Println(" 3. Change Interface Admin State")
		fmt.Println(" 4. Configure Link Aggregation (LAG)")
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
			menuSubInterfaceDetail(d, port)
		case "2":
			menuBridgeInterfaceDetail(d, port)
		case "3":
			state, err := askAdminState()
			if err != nil {
				fmt.Printf(">> Error: %v\n", err)
				continue
			}
			d.ConfigInterfaceAdminStatus(port, state)
		case "4":
			fmt.Println(">> To configure LAG, select an 'ae' card from the main menu.")
		default:
			fmt.Println(">> Invalid selection.")
		}
	}
}

func showPortDetail(port *device.Port) {
	fmt.Println()
	printMenuLine("=", fmt.Sprintf(" %s / %s ", port.ParentCard.PortType, port.IfName))
	fmt.Printf("    Status: %-12s Admin Status: %s\n", port.OperStatus, port.AdminStatus)
	// Add more details here as needed
}

