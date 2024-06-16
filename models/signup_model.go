package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type signupModel struct {
	inputs     []textinputGroup
	focusIndex int
	cursor     cursor.Mode
}

func InitSignupModel() signupModel {
	emailInput := textinput.New()
	emailInputGroup := newTextinputGroup(emailInput, "Email")
	emailInputGroup.Validate = validateEmail

	passwordInput := textinput.New()
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInputGroup := newTextinputGroup(passwordInput, "Password")
	passwordInputGroup.Validate = validateMinLen(12)

	firstNameInput := textinput.New()
	firstNameInput.Placeholder = "Optional"
	firstNameInputGroup := newTextinputGroup(firstNameInput, "First Name")

	lastNameInput := textinput.New()
	lastNameInput.Placeholder = "Optional"
	lastNameInputGroup := newTextinputGroup(lastNameInput, "Last Name")

	m := signupModel{
		inputs: []textinputGroup{
			emailInputGroup,
			passwordInputGroup,
			firstNameInputGroup,
			lastNameInputGroup,
		},
		focusIndex: 0,
	}

	m.inputs[0].Input.Focus()

	return m
}

func (m signupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m signupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab, tea.KeyEnter, tea.KeyDown:
			// 4 inputs, last idx is 3
			if msg.Type == tea.KeyEnter && m.focusIndex == 3 {
				hasErrors := false
				firstErrIdx := -1
				for i := range m.inputs {
					if m.inputs[i].Validate != nil {
						m.inputs[i].Err = m.inputs[i].Validate(m.inputs[i].Input.Value())
						if m.inputs[i].Err != nil {
							hasErrors = true
							if firstErrIdx == -1 {
								firstErrIdx = i
							}
						}
					}
				}

				if hasErrors {
					m.inputs[m.focusIndex].Input.Blur()
					m.focusIndex = firstErrIdx
					cmds := []tea.Cmd{
						m.inputs[firstErrIdx].Input.Focus(),
						m.updateInputs(msg),
					}
					return m, tea.Batch(cmds...)
				}
				return m, tea.Quit
			}

			// move to next text input
			currIdx := m.focusIndex
			nextIdx := m.findNextFocusIdx()

			// blur current input
			m.inputs[currIdx].Input.Blur()

			// focux new input
			cmd := m.inputs[nextIdx].Input.Focus()

			m.focusIndex = nextIdx

			return m, cmd
		case tea.KeyShiftTab, tea.KeyUp:
			// get new indeces
			currIdx := m.focusIndex
			prevIdx := m.findPrevFocusIdx()

			// blur current input
			m.inputs[currIdx].Input.Blur()

			// focus on prev input
			cmd := m.inputs[prevIdx].Input.Focus()

			m.focusIndex = prevIdx

			return m, cmd
		}
	}

	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m signupModel) View() string {
	var builder strings.Builder

	for i := range m.inputs {
		if m.inputs[i].Err != nil {
			if i < len(m.inputs)-1 {
				builder.WriteString(fmt.Sprintf("%s %s %s\n", m.inputs[i].Label, m.inputs[i].Input.View(), m.inputs[i].Err.Error()))
			} else {
				builder.WriteString(fmt.Sprintf("%s %s %s", m.inputs[i].Label, m.inputs[i].Input.View(), m.inputs[i].Err.Error()))
			}
			continue
		}

		if i < len(m.inputs)-1 {
			builder.WriteString(fmt.Sprintf("%s %s\n", m.inputs[i].Label, m.inputs[i].Input.View()))
		} else {
			builder.WriteString(fmt.Sprintf("%s %s", m.inputs[i].Label, m.inputs[i].Input.View()))
		}
	}

	return builder.String()
}

func (m *signupModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i].Input, cmds[i] = m.inputs[i].Input.Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m *signupModel) findNextFocusIdx() int {
	// loop
	if m.focusIndex == len(m.inputs)-1 {
		return 0
	}

	return m.focusIndex + 1
}

func (m *signupModel) findPrevFocusIdx() int {
	// loop
	if m.focusIndex == 0 {
		return len(m.inputs) - 1
	}

	return m.focusIndex - 1
}
