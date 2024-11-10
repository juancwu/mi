package cmd

import (
	"fmt"
	"os"
	"testing"
)

func TestReadEnvFile(t *testing.T) {
	inputEnvStr := `TEST=1

# comment
                TEST_2=2
    `
	tmpEnvFile, err := os.CreateTemp("", "tmpfile-")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpEnvFile.Close()
	defer os.Remove(tmpEnvFile.Name())
	_, err = tmpEnvFile.Write([]byte(inputEnvStr))
	if err != nil {
		t.Fatal(err)
	}
	values, err := readEnvFile(tmpEnvFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		"TEST":   "1",
		"TEST_2": "2",
	}
	for _, value := range values {
		if expected_value, ok := expected[value.Name]; ok {
			if expected_value != value.Value {
				t.Error(fmt.Errorf("Unexpected value from parsed env file. Expected: %s but received %s", expected_value, value.Value))
			}
		} else {
			t.Error(fmt.Errorf("Unexpected env key from parsed env file. Got key: %s", value.Name))
		}
	}
}
