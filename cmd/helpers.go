package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/juancwu/mi/config"
	"github.com/juancwu/mi/util"
)

// getNewAccessToken makes a request to get a new access token with a stored refresh token.
func getNewAccessToken(c *config.Credentials) error {
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/auth/refresh", config.GetServiceURL()), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.RefreshToken))
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		var body map[string]string
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &body)
		if err != nil {
			return err
		}
		at, ok := body["access_token"]
		if !ok {
			return errors.New("No access token found in response body.")
		}
		c.AccessToken = at
		err = config.SaveCredentials(c)
		if err != nil {
			return err
		}
	case http.StatusUnauthorized:
		return newErrExpiredCreds()
	default:
		return errors.New("Failed to get new access token.")
	}
	return nil
}

// readApiResponseBody tries to read a response body of type JSON and returns the apiResponse struct.
func readApiResponseBody(body io.ReadCloser) (*apiResponse, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var resBody apiResponse
	if err := json.Unmarshal(b, &resBody); err != nil {
		return nil, err
	}
	return &resBody, nil
}

// logApiResponseBody handles formatted logging of an api response body.
func logApiResponseBody(resBody *apiResponse) {
	if resBody.Message != "" {
		fmt.Printf("Message: %s\n", resBody.Message)
	}
	if resBody.RequestId != "" {

		fmt.Printf("Request ID: %s\n", resBody.RequestId)
	}
	if len(resBody.Errs) > 0 {
		util.LogApiResponseErrs(resBody.Errs)
	}
}
