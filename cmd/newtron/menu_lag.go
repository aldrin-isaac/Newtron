//File: cmd/newtron/menu_lag.go
// Chat Session: 20240706_1244_NewtronProject
// Session Timestamp: 2025-07-06T14:10:00-07:00
package main

import (
	"fmt"
	"os"

	"newtron/pkg/device"
)

func menuAEInterfaceLAGDetail(d *device.Device, aePort *device.Port) {
	for {
		if aePort.ParentCard.PortType != "ae" {
			fmt.Println(">> Not a Link Aggregation port.")
			return
		}
		if err := d.LoadLAGDetail(aePort); err != nil {
			fmt.Printf(">> Error loading LAG details: %v\n", err)
			return
		}

		showAEInterfaceMemberInfo(aePort)
		showAEInterfaceLACPDetail(aePort)

		fmt.Println("\nActions:")
		fmt.Println(" 1. Configure Link Aggregation Members")
		fmt.Println(" 2. Configure Link Aggregation Control Protocol (LACP)")
		printMenuLine("=", aePort.ParentCard.ParentNode.Name)

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
			menuAEInterfaceMemberDetail(d, aePort)
		case "2":
			fmt.Print("Enable or Disable LACP (enable/disable): ")
			action := readInput()
			if action == "enable" || action == "disable" {
				d.ConfigAEInterfaceLACP(aePort, action)
			} else {
				fmt.Println(">> Invalid action.")
			}
		default:
			fmt.Println(">> Action not implemented yet.")
		}
	}
}

func menuAEInterfaceMemberDetail(d *device.Device, aePort *device.Port) {
	for {
		showAEInterfaceMemberInfo(aePort)
		fmt.Println("\nActions:")
		fmt.Println(" 1. Add Interface to Bundle")
		fmt.Println(" 2. Remove Interface from Bundle")
		printMenuLine("=", aePort.ParentCard.ParentNode.Name)

		fmt.Print("Enter Selection: ")
		input := readInput()

		switch input {
		case "x":
			os.Exit(0)
		case "q":
			return
		case "":
			continue
		case "1": // Add member
			memberPort, err := askAEInterfaceMember(d, aePort, "add")
			if err != nil {
				fmt.Printf(">> Error: %v\n", err)
				continue
			}
			fmt.Printf("Add %s to bundle %s? (y/n) [n]: ", memberPort.IfName, aePort.IfName)
			if readInput() == "y" {
				d.ConfigAEInterfaceMember(aePort, memberPort, "add")
			}
		case "2": // Remove member
			memberPort, err := askAEInterfaceMember(d, aePort, "remove")
			if err != nil {
				fmt.Printf(">> Error: %v\n", err)
				continue
			}
			fmt.Printf("Remove %s from bundle %s? (y/n) [n]: ", memberPort.IfName, aePort.IfName)
			if readInput() == "y" {
				d.ConfigAEInterfaceMember(aePort, memberPort, "remove")
			}
		default:
			fmt.Println(">> Invalid selection.")
		}
	}
}

func askAEInterfaceMember(d *device.Device, aePort *device.Port, action string) (*device.Port, error) {
	fmt.Print("Enter Member Interface (e.g., xe-0/0/1): ")
	ifName := readInput()
	// This is a mock lookup. A real implementation would search all ports on the node.
	for _, card := range d.Node.Cards {
		for _, port := range card.Ports {
			if port.IfName == ifName {
				return port, nil
			}
		}
	}
	return nil, fmt.Errorf("interface %s not found", ifName)
}

func showAEInterfaceMemberInfo(aePort *device.Port) {
	fmt.Println()
	printMenuLine("=", fmt.Sprintf(" AE / %s ", aePort.IfName))
	if aePort.LAG == nil || len(aePort.LAG.Members) == 0 {
		fmt.Println("  No members in this bundle.")
		return
	}
	for _, member := range aePort.LAG.Members {
		fmt.Printf("  Member: %-15s (%s)\n", member.IfName, member.AdminStatus)
	}
}

func showAEInterfaceLACPDetail(aePort *device.Port) {
	printMenuLine("-", " LACP ")
	if aePort.LAG != nil && aePort.LAG.LACPEnabled {
		fmt.Printf("    LACP: %-8s    Mode: %-8s    Rate: %s\n", "enabled", aePort.LAG.LACPMode, aePort.LAG.LACPRate)
	} else {
		fmt.Println("    LACP: disabled")
	}
}

