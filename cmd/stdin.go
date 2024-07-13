package cmd

import (
	"bufio"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"
)

// readEmail is a helper function that will prompt the user to enter their email and read it.
func readEmail() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return email[:len(email)-1], nil
}

// readPassword is a helper function that will prompt the user to enter their password and read it.
// This method won't show any echo on the terminal.
func readPassword() (string, error) {
	fmt.Print("Enter your password:")
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Print("\n")
	password := string(bytePassword)
	return password, nil
}
