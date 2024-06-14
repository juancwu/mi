package models

import (
	"errors"
	"fmt"
	"net/mail"

	"github.com/charmbracelet/bubbles/textinput"
)

// validateEmail validates if the given input is a valid email.
// This function uses net/mail that implements RFC 5322.
func validateEmail(s string) error {
	_, err := mail.ParseAddress(s)
	if err != nil {
		return errors.New("Invalid email")
	}
	return nil
}

// validateMinLen validates the minimum of length of the input s against the defined length.
func validateMinLen(l int) textinput.ValidateFunc {
	return func(s string) error {
		if len(s) < l {
			return fmt.Errorf("Minimum length of %d not satisfied", l)
		}
		return nil
	}
}
