package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/juancwu/konbini-cli/shared/form"
	"github.com/juancwu/konbini-cli/utils"
	"github.com/spf13/cobra"
)

var createBentoCmd = &cobra.Command{
	Use:   "bento [name]",
	Short: "Commands to manage personal bentos.",
	Args:  cobra.ExactArgs(1),
	Run:   createBentoRun,
}

func createBentoRun(cmd *cobra.Command, args []string) {
	creds, err := utils.PromptCredentials()
	if err != nil {
		log.Errorf("Failed to get credentials from prompt: %v\n", err)
		return
	}

	// check if env file is present in the current directory
	envData, err := utils.GetEnvContent()
	if err != nil {
		log.Errorf("Failed to read .env file in current directory: %v\n", err)
		return
	}

	if len(envData) == 0 {
		log.Error("Env file is empty.")
		return
	}

	// authenticate user
	log.Info("Authenticating...")
	accessToken, err := utils.Auth(creds.Email, creds.Password)
	if err != nil {
		log.Errorf("Failed to authenticate user: %v\n", err)
		return
	}

	// get new keys
	log.Info("Generating keys...")
	keys, err := utils.GenerateKey()
	if err != nil {
		log.Errorf("Failed to generate private key: %v\n", err)
		return
	}

	// encrypt env envData
	log.Info("Encrypting env data...")
	encrypted, err := utils.Encrypt(envData, keys)
	if err != nil {
		log.Errorf("Failed to encrypt env data: %v\n", err)
		return
	}

	log.Info("Encoding keys to PEM...")
	privatePEM, publicPEM, err := utils.Keys2PEM(keys)
	if err != nil {
		log.Errorf("Failed to encode keys to PEM: %v\n", err)
		return
	}

	log.Info("Saving public key in disk as 'public.pem'")
	err = os.WriteFile("public.pem", publicPEM, 0644)
	if err != nil {
		log.Errorf("Failed to save public key to disk: %v\n", err)
		return
	}

	log.Info("Saving private key in disk as 'private.pem'")
	err = os.WriteFile("private.pem", privatePEM, 0644)
	if err != nil {
		log.Errorf("Failed to save private key to disk: %v\n", err)
		return
	}

	log.Info("Gathering bento ingridients...")
	content := base64.StdEncoding.EncodeToString(encrypted)
	log.Info(content)
	bentoForm := form.BentoForm{
		Name:      args[0],
		PublicKey: string(publicPEM),
		Content:   base64.StdEncoding.EncodeToString(encrypted),
	}
	reqBodyBytes, err := json.Marshal(bentoForm)
	if err != nil {
		log.Errorf("Failed to gather bento ingridients: %v\n", err)
		return
	}

	// make bento request
	log.Info("Preheating pans...")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bento/personal/new", utils.GetServiceURL()), bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		log.Errorf("Failed to preheat pans: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(reqBodyBytes)))

	// make client to do the request
	log.Info("Cooking bento...")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Something went wrong when cooking bento: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		log.Info("Bento cooked and safely stored in Konbini.")
		bentoIdBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error reading response body: %v\n", err)
			return
		}
		log.Info("Saving bento recipe...")
		cwd, err := os.Getwd()
		if err != nil {
			log.Warn("Failed to get current working directory, using fallback method.")
			cwd = ""
		}
		bentoConfig := utils.CreateBentoConfig(bentoForm.Name, string(bentoIdBytes), filepath.Join(cwd, "public.pem"), filepath.Join(cwd, "private.pem"))
		err = os.WriteFile(filepath.Join(cwd, DEFAULT_CFG_FILE_PATH), []byte(bentoConfig), 0644)
		if err != nil {
			log.Errorf("Failed to save bento recipe in disk: %v\n", err)
			return
		}
	} else {
		// show error message
		errMsgBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error reading response body: %v\n", err)
			return
		}
		log.Errorf("Failed to cook bento (code: %d): %s\n", resp.StatusCode, string(errMsgBytes))
	}
}

var getBentoCmd = &cobra.Command{
	Use:   "bento",
	Short: "Gets a bento based on the .konbini.yaml file.",
	Run:   getBentoRun,
}

type PersonalBento struct {
	Id        string    `json:"id"`
	OwnerId   string    `json:"owner_id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	PubKey    string    `json:"pub_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func getBentoRun(cmd *cobra.Command, args []string) {
	log.Infof("Loading bento configuration from '%s'", cfgFilePath)
	bentoConfig, err := utils.GetBentoConfig(cfgFilePath)
	if err != nil {
		log.Errorf("Failed to get bento config: %v\n", err)
		return
	}

	log.Info("Getting bento keys...")
	keys, err := utils.LoadKeys(bentoConfig)
	if err != nil {
		log.Errorf("Failed to get bento keys: %v\n", err)
		return
	}

	// check if there is an existing .env file and prompt for confirmation if there is
	_, err = os.Stat(".env")
	if err != nil && !os.IsNotExist(err) {
		var confirmation string
		log.Warnf("Failed to check for existing .env file. Continueing will overwrite any existing .env file in the current working directory. Continue? (y/n) ")
		_, err := fmt.Scanln(&confirmation)
		if err != nil {
			log.Errorf("Failed to read confirmation: %v\n", err)
			log.Error("ABORT")
			return
		}
		if strings.ToLower(confirmation) != "y" {
			return
		}
	} else if err == nil {
		var confirmation string
		log.Warnf("Existing .env file found. Do you want to overwrite it? (y/n) ")
		_, err := fmt.Scanln(&confirmation)
		if err != nil {
			log.Errorf("Failed to read confirmation: %v\n", err)
			log.Error("ABORT")
			return
		}
		if strings.ToLower(confirmation) != "y" {
			return
		}
	}

	log.Info("Authenticating...")
	hashed, signature, err := utils.GetSignature(keys)
	if err != nil {
		log.Errorf("Failed to get authentication signature: %v\n", err)
		return
	}

	log.Info("Preparing bento order...")
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/bento/personal/%s", utils.GetServiceURL(), bentoConfig.BentoId), nil)
	if err != nil {
		log.Errorf("Failed to prepare bento order: %v\n", err)
		return
	}
	req.Header.Set("X-Bento-Hashed", base64.StdEncoding.EncodeToString(hashed))
	req.Header.Set("X-Bento-Signature", base64.StdEncoding.EncodeToString(signature))

	log.Info("Placing bento order...")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to place bento order: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read response body: %v\n", err)
		return
	}
	if resp.StatusCode == http.StatusOK {
		var bento PersonalBento
		err = json.Unmarshal(respBodyBytes, &bento)
		if err != nil {
			log.Errorf("Failed to unmarshal response body: %v\n", err)
			return
		}
		log.Info("Got bento! Opening bento...")

		// open bento by decrypting the content
		plaintext, err := utils.Decrypt(bento.Content, keys)
		if err != nil {
			log.Errorf("Failed to decrypt bento: %v\n", err)
			return
		}

		log.Info("Saving bento into disk...")
		err = os.WriteFile(".env", plaintext, 0644)
		if err != nil {
			log.Errorf("Failed to save bento to disk: %v\n", err)
			return
		}
	} else {
		log.Errorf("Error (%d): %s\n", resp.StatusCode, string(respBodyBytes))
	}
}

var listBentosCmd = &cobra.Command{
	Use:   "bentos",
	Short: "Get a list of bentos stored in Konbini.",
	Long:  "Get a list of bentos stored in Konbini showing id, name, created at, and updated at.",
	Run:   listBentosRun,
}

type PersonalBentoListItem struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func listBentosRun(cmd *cobra.Command, args []string) {
	// get credentials
	creds, err := utils.PromptCredentials()
	if err != nil {
		log.Errorf("Failed to get credentials: %v\n", err)
		return
	}

	// get access token
	log.Info("Authentication...")
	token, err := utils.Auth(creds.Email, creds.Password)
	if err != nil {
		log.Error(err)
		return
	}

	// make request to get list of bentos
	log.Info("Preparing order...")
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/bento/personal/list", utils.GetServiceURL()), nil)
	if err != nil {
		log.Error(err)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	log.Info("Placing order...")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		var data []PersonalBentoListItem
		err = json.Unmarshal(respBodyBytes, &data)
		if err != nil {
			log.Error(err)
			return
		}
		prettyJson, err := json.MarshalIndent(data, "", "   ")
		if err != nil {
			log.Error(err)
			return
		}
		log.Print(string(prettyJson))
	} else {
		log.Errorf("Failed to get list of bentos: %s\n", string(respBodyBytes))
	}
}
