package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/juancwu/mi/config"
)

// Shortcut function to make a sign-in request. It won't parse the response but
// will make the repetitive and boring request setup better.
func SignIn(email, password string) (*http.Response, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/auth/signin", config.GetServiceURL()), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(b)))
	client := http.Client{}
	res, err := client.Do(req)
	return res, err
}

// Utility function to update the email of an account given a valid access token.
func UpdateEmail(newEmail, accessToken string) (*http.Response, error) {
	body := map[string]string{
		"new_email": newEmail,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/auth/email/update", config.GetServiceURL()), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(b)))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := http.Client{}
	res, err := client.Do(req)
	return res, err
}
