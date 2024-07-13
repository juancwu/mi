package cmd

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/juancwu/konbini-cli/config"
	"github.com/spf13/cobra"
)

// newBentoCmd creates a new bento cmd and all of its subcommands.
func newBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bento",
		Short: "Bento related commands. Get, update, delete, all in here.",
	}
	cmd.AddCommand(newOrderBentoCmd())
	return cmd
}

// newOrderBentoCmd creates a new command to order a prepared bento.
func newOrderBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Ordered a bento that was previously prepared.",
		Long:  "Use this command when you need to get the contents of a bento that was previously prepared. You will need the private key that was used for the prepared bento.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfiguration()
			if err != nil {
				return err
			}
			// create challenge
			challenge, err := createChallenge()
			if err != nil {
				return err
			}
			// hash the challenge with sha256
			hashed := sha256.Sum256(challenge)
			// read in the private key
			block, err := readPEMKey(cfg.PrivateKeyPath)
			if err != nil {
				return err
			}
			// parse the private key so that the challenge can be signed
			pk, err := parsePrivateKey(block)
			if err != nil {
				return err
			}
			// sign the challenge
			signature, err := signChallenge(pk, hashed)
			if err != nil {
				return err
			}
			// encode both challenge and signature to send to Konbini API
			encodedSign := hex.EncodeToString(signature)
			encodedChallenge := hex.EncodeToString(challenge)
			// preprare request query params
			values := url.Values{}
			values.Add("signature", encodedSign)
			values.Add("challenge", encodedChallenge)
			q := values.Encode()
			// prepare http request
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/bento/%s?%s", config.GetServiceURL(), cfg.BentoId, q), nil)
			if err != nil {
				return err
			}
			// prepare client
			client := http.Client{}
			res, err := client.Do(req)
			switch res.StatusCode {
			case http.StatusOK:
				fmt.Println("success")
			default:
				fmt.Println("failed")
			}
			return nil
		},
	}

	cmd.Flags().String("key-path", "private.pem", "Optional: The path of the PEM encoded private key. Use this flag if you would like to use a different private key aside from the one described in the configuration file.")
	cmd.Flags().StringP("bento", "b", "", "Optional: The prepared bento id that to order. Use this flag if you would like to use a different bento id aside from the one described in the configuration file.")

	return cmd
}

// createChallenge generates a challenge that can later be used for signing.
// Most bento related actions required a signed challenge to gain access.
func createChallenge() ([]byte, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	return randomBytes, nil
}

// signChallenge signs the given challenge with the private key.
func signChallenge(pk *rsa.PrivateKey, challenge [32]byte) ([]byte, error) {
	return rsa.SignPKCS1v15(nil, pk, crypto.SHA256, challenge[:])
}

// parsePrivateKey parses a private key in PEM format.
func parsePrivateKey(block *pem.Block) (*rsa.PrivateKey, error) {
	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("Unsupported private key. Expected type: 'PRIVATE KEY', but found: '%s'", block.Type)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Failed to parse private key.")
	}
	return pk, nil
}

// parsePublicKey parses a public key in PEM format.
func parsePublicKey(block *pem.Block) (*rsa.PublicKey, error) {
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("Unsupported public key. Expected type: 'PUBLIC KEY', but found: '%s'", block.Type)
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Failed to parse public key.")
	}
	return pk, nil
}

// readPEMKey will try to read the pem encoded key in the given path and return the block.
func readPEMKey(path string) (*pem.Block, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(fb)
	return block, nil
}
