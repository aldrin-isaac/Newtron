//File: cmd/newtron/menu_subinterface.go
// Chat Session: 20240706_1244_NewtronProject
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

type menuAction struct {
	Key         string
	Description string
}

func menuSubInterfaceDetail(d *device.Device, port *device.Port) {
	for {
		if err := d.LoadPortDetail(port); err != nil {
			fmt.Printf("Error loading port details: %v\n", err)
			return
		}

		showInterfaceIpInfo(port)
		actions := menuSubInterfaceActions()
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
			if err != nil || selection < 1 || selection > len(actions) {
				fmt.Println(">> Invalid selection.")
				continue
			}
			actionKey := actions[selection-1].Key
			handleSubInterfaceAction(d, port, actionKey)
		}
	}
}

func menuSubInterfaceIPVPN(d *device.Device, subInterface *device.SubInterface) {
	for {
		if subInterface.VRF == nil {
			fmt.Println(">> Sub-interface is not part of a VRF.")
			return
		}
		d.LoadVRFDetail(subInterface.VRF)

		showSubInterfaceIPVPNDetail(subInterface)

		availableVPNs, err := d.GetAvailableVPNs(subInterface)
		if err != nil {
			fmt.Printf(">> Error getting available VPNs: %v\n", err)
			return
		}

		fmt.Println("\nAvailable VPNs:")
		for i, vpn := range availableVPNs {
			fmt.Printf(" %d. %s\n", i+1, vpn.Description)
		}
		printMenuLine("=", subInterface.ParentPort.ParentCard.ParentNode.Name)

		fmt.Print("Enter IP VPN to configure: ")
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
			if err != nil || selection < 1 || selection > len(availableVPNs) {
				fmt.Println(">> Invalid VPN selection.")
				continue
			}
			selectedVPN := availableVPNs[selection-1]

			fmt.Print("Add or Delete (add/delete): ")
			action := readInput()
			if action == "add" || action == "delete" {
				d.ConfigSubInterfaceIPVPN(subInterface, selectedVPN, action)
			} else {
				fmt.Println(">> Invalid action.")
			}
		}
	}
}

func menuSubInterfaceActions() []menuAction {
	actions := map[string]string{
		"addip":      "Add IP Interface",
		"delip":      "Remove IP Interface",
		"modipstate": "Enable/Disable IP Interface State",
		"modvpn":     "IP VPN Membership",
		"modbw":      "Set SubInterface Bandwidth (Shaping)",
		"modfilter":  "Update Interface Firewall Policy",
	}

	keys := make([]string, 0, len(actions))
	for k := range actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	menuItems := []menuAction{}
	fmt.Println("\nActions:")
	for i, key := range keys {
		fmt.Printf(" %d. %s\n", i+1, actions[key])
		menuItems = append(menuItems, menuAction{Key: key, Description: actions[key]})
	}
	printMenuLine("=", "Sub-Interface Actions")
	return menuItems
}

func handleSubInterfaceAction(d *device.Device, port *device.Port, actionKey string) {
	var subInterface *device.SubInterface
	if len(port.SubInterfaces) > 0 {
		subInterface = port.SubInterfaces[0]
	}

	if actionKey != "addip" && subInterface == nil {
		fmt.Println(">> No sub-interface to act upon. Please add one first.")
		return
	}

	switch actionKey {
	case "addip":
		ip, service, err := askIPAddress()
		if err != nil {
			fmt.Printf(">> Error: %v\n", err)
			return
		}
		subIfID, err := askSubInterface(port)
		if err != nil {
			fmt.Printf(">> Error: %v\n", err)
			return
		}
		fmt.Printf("Create %s %s.%d? (y/n) [n]: ", service, port.IfName, subIfID)
		if readInput() == "y" {
			d.ConfigSubInterface(port, subIfID, ip, service, "add")
		}
	case "delip":
		fmt.Printf("Delete sub-interface %s? (y/n) [n]: ", subInterface.IfName)
		if readInput() == "y" {
			d.ConfigSubInterface(subInterface.ParentPort, subInterface.ID, "", "", "delete")
		}
	case "modipstate":
		state, err := askAdminState()
		if err != nil {
			fmt.Printf(">> Error: %v\n", err)
			return
		}
		d.ConfigSubInterfaceIPAdminStatus(subInterface, state)
	case "modbw":
		bw, err := askSubInterfaceBandwidth()
		if err != nil {
			fmt.Printf(">> Error: %v\n", err)
			return
		}
		//d.ConfigSubInterfaceBandwidth(subInterface, bw)
	case "modfilter":
		fmt.Printf("Update Firewall Policy on %s? [n]: ", subInterface.IfName)
		if readInput() == "y" {
			d.ConfigFirewallPolicy(subInterface)
		}
	case "modvpn":
		menuSubInterfaceIPVPN(d, subInterface)
	default:
		fmt.Println(">> Action not implemented yet.")
	}
}

func askSubInterface(port *device.Port) (int, error) {
	fmt.Print("Enter SubInterface Number: ")
	input := readInput()
	if input == "" {
		return 0, errors.New("no sub-interface entered")
	}
	id, err := strconv.Atoi(input)
	if err != nil {
		return 0, errors.New("invalid number")
	}
	return id, nil
}

func askIPAddress() (string, string, error) {
	fmt.Print("Enter IP Address/Mask (e.g., 1.1.1.1/30): ")
	input := readInput()
	if !strings.Contains(input, "/") {
		return "", "", errors.New("invalid format, requires CIDR mask (e.g., /30)")
	}
	return input, "vce", nil
}

func askSubInterfaceBandwidth() (string, error) {
	fmt.Print("Enter SubInterface Bandwidth (mbps) [e.g., 10, 50, 100]: ")
	input := readInput()
	_, err := strconv.Atoi(input)
	if err != nil {
		return "", errors.New("invalid bandwidth value, must be a number")
	}
	return input + "m", nil
}

func showInterfaceIpInfo(port *device.Port) {
	fmt.Println()
	printMenuLine("=", fmt.Sprintf(" IP SubInterfaces / %s ", port.IfName))
	if len(port.SubInterfaces) == 0 {
		fmt.Println("  No IP sub-interfaces configured.")
	}
	for _, su := range port.SubInterfaces {
		filterInfo := ""
		if su.FilterName != "" {
			filterInfo = fmt.Sprintf(" Filter: %s", su.FilterName)
		}
		fmt.Printf("SubIf: %-5d  IP: %-18s (%s) (%s)%s\n", su.ID, su.IPAddress, su.Service, su.AdminStatus, filterInfo)
	}
}

func showSubInterfaceIPVPNDetail(subInterface *device.SubInterface) {
	vrf := subInterface.VRF
	fmt.Println()
	printMenuLine("=", fmt.Sprintf(" IP VPNs / %s ", subInterface.IfName))
	fmt.Printf("   VRF Name: %-12s  RD: %s\n", vrf.Name, vrf.RouteDistinguisher)

	importRTs := "none"
	if len(vrf.ImportRouteTargets) > 0 {
		importRTs = strings.Join(vrf.ImportRouteTargets, ", ")
	}
	fmt.Printf("   Import RTs: %s\n", importRTs)

	exportRTs := "none"
	if len(vrf.ExportRouteTargets) > 0 {
		exportRTs = strings.Join(vrf.ExportRouteTargets, ", ")
	}
	fmt.Printf("   Export RTs: %s\n", exportRTs)

	fmt.Println("\n   Associated VPNs:")
	if len(vrf.AssociatedVPNs) == 0 {
		fmt.Println("    - None")
	} else {
		for _, vpn := range vrf.AssociatedVPNs {
			fmt.Printf("    - %s\n", vpn.Description)
		}
	}
}

