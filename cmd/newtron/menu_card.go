// File: cmd/newtron/menu_card.go
// Chat Session: 20240706_1244_NewtronProject
// Session Timestamp: 2025-07-06T14:10:00-07:00
package main

import (
	"fmt"
	"sort"

	"newtron/pkg/device"
)

// menuCard is the top-level menu for selecting a card (chassis-level view).
func menuCard(d *device.Device) {
	for {
		if err := d.LoadCards(); err != nil {
			fmt.Printf("Error loading cards: %v. Please try again.\n", err)
			continue
		}
		showCards(d.Node)
		fmt.Print("Enter Selection: ")
		input := readInput()

		switch input {
		case "x", "q":
			return
		case "":
			continue
		default:
			selectedCard, ok := d.Node.Cards[input]
			if !ok {
				fmt.Println(">> Invalid selection.")
				continue
			}
			menuPort(d, selectedCard)
		}
	}
}

// showCards displays the list of available cards on the device.
func showCards(node *device.Node) {
	fmt.Println()
	printMenuLine("=", " Cards ")
	fmt.Printf("Node: %-20s   Ver: %-10s Chassis: %s (%s)\n", node.Name, node.Version, node.Chassis, node.ConfigClass)
	printMenuLine("-", "")

	ids := make([]string, 0, len(node.Cards))
	for id := range node.Cards {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		card := node.Cards[id]
		fmt.Printf(" %-3s. %s (%s)\n", id, card.Description, card.Model)
	}
	printMenuLine("=", node.Name)
}

