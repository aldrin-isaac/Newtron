// File: cmd/newtron/utils.go
// Chat Session: newtron-20250706-01
// Session Timestamp: 2025-07-06T15:24:00-07:00
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

const termWidth = 76

// printMenuLine prints a separator line for menus.
func printMenuLine(char, title string) {
	if title != "" {
		padding := (termWidth - len(title) - 2) / 2
		if padding < 0 {
			padding = 0
		}
		rightPadding := termWidth - len(title) - padding - 2
		if rightPadding < 0 {
			rightPadding = 0
		}
		fmt.Println(strings.Repeat(char, padding), title, strings.Repeat(char, rightPadding))
	} else {
		fmt.Println(strings.Repeat(char, termWidth))
	}
}

// readInput reads a single line of text from stdin.
func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// askAdminState prompts the user for an enable/disable action.
func askAdminState() (string, error) {
	fmt.Print("Enable or Disable Status (enable/disable): ")
	input := readInput()
	if input != "enable" && input != "disable" {
		return "", errors.New("invalid state, must be 'enable' or 'disable'")
	}
	return input, nil
}

