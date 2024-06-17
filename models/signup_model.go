package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type apiResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

// signupRequestBody represents the JSON body that the signup endpoint requires for a signup.
type signupRequestBody struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

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
	firstNameInputGroup := newTextinputGroup(firstNameInput, "First Name")

	lastNameInput := textinput.New()
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

				email := m.inputs[0].Input.Value()
				password := m.inputs[1].Input.Value()
				firstName := m.inputs[2].Input.Value()
				lastName := m.inputs[3].Input.Value()

				reqBody := signupRequestBody{
					Email:     email,
					Password:  password,
					FirstName: firstName,
					LastName:  lastName,
				}
				reqBodyBytes, err := json.Marshal(reqBody)
				if err != nil {
					log.Fatal(err)
					return m, tea.Quit
				}

				reqBodyReader := bytes.NewReader(reqBodyBytes)
				req, err := http.NewRequest(http.MethodPost, "http://localhost:3000/api/v1/account/signup", reqBodyReader)
				if err != nil {
					log.Fatal(err)
					return m, tea.Quit
				}
				req.Header.Add("Content-Type", "application/json")

				client := http.Client{}
				res, err := client.Do(req)
				if err != nil {
					log.Fatal(err)
					return m, tea.Quit
				}
				defer res.Body.Close()

				resBodyBytes, err := io.ReadAll(res.Body)
				if err != nil {
					log.Fatal(err)
					return m, tea.Quit
				}
				var resBody apiResponse
				err = json.Unmarshal(resBodyBytes, &resBody)
				if err != nil {
					log.Fatal(err)
					return m, tea.Quit
				}

				if res.StatusCode != http.StatusCreated {
					log.Fatal(errors.New(resBody.Message))
					return m, tea.Quit
				}

				fmt.Println(resBody.Message)

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

	builder.WriteString("Use tab, enter, or arrow keys to navigate the inputs.\n\n")
	for i := range m.inputs {
		if m.inputs[i].Err != nil {
			builder.WriteString(fmt.Sprintf(
				"%s %s %s\n",
				m.inputs[i].Label,
				m.inputs[i].Input.View(),
				errStyle.Render(m.inputs[i].Err.Error()),
			))
			continue
		}

		builder.WriteString(
			fmt.Sprintf("%s %s\n",
				m.inputs[i].Label,
				m.inputs[i].Input.View(),
			))
	}
	builder.WriteString("\nPress ESC to quit the program.\n")

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
