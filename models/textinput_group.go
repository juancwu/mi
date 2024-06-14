package models

import "github.com/charmbracelet/bubbles/textinput"

// inputGroup represents a more complete textinput from bubbles.
// It holds the textinput itself, its label, and an error field for input validation.
type textinputGroup struct {
	Input    textinput.Model
	Label    string
	Err      error
	Validate textinput.ValidateFunc
}

func newTextinputGroup(input textinput.Model, label string) textinputGroup {
	return textinputGroup{
		Input: input,
		Label: label,
		Err:   nil,
	}
}
