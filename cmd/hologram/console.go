package main

import (
	"encoding/json"
	"fmt"
	"github.com/AdRoll/hologram/log"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func init() {
	consoleCmd.Flags().Bool("new-session", false, "Start a new Google Chrome session. This allows use of multiple roles simultaneously.")
	consoleCmd.Flags().Bool("show-url", false, "Show the federation URL used for sign-in.")
	consoleCmd.Flags().Bool("no-launch", false, "Don't launch the browser.")
	rootCmd.AddCommand(consoleCmd)
}

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Open the AWS console in the default browser",
	Run: func(cmd *cobra.Command, args []string) {
		newSession, err :=  cmd.Flags().GetBool("new-session")
		showUrl, err :=  cmd.Flags().GetBool("show-url")
		noLaunch, err :=  cmd.Flags().GetBool("no-launch")
		if err == nil {
			err = launchConsole(newSession, showUrl, noLaunch)
		}

		if err != nil {
			log.Errorf("%s", err)
			os.Exit(1)
		}
	},
}

type HttpHologramCredentials struct {
	Code string
	LastUpdated string
	Type string
	AccessKeyId string
	SecretAccessKey string
	Token string
	Expiration string
}

type HttpAwsCredentials struct {
	SessionId string `json:"sessionId"`
	SessionKey string `json:"sessionKey"`
	SessionToken string `json:"sessionToken"`
}

type HttpFederationSigninToken struct {
	SigninToken string
}

func launchConsole(newSession bool, showUrl bool, noLaunch bool) error {
	federationUrlBase := "https://signin.aws.amazon.com/federation"
	profileUrl := "http://169.254.169.254/latest/meta-data/iam/security-credentials/"
	awsConsoleUrl := "https://console.aws.amazon.com/"

	// Get the profile name from the metadata service
	response, err := http.Get(profileUrl)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	profileBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	profile := string(profileBytes)

	// Get the credentials from the metadata service
	metadataUrl := fmt.Sprintf("%v%v", profileUrl, profile)
	response, err = http.Get(metadataUrl)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("error getting credentials. Try running 'hologram me'")
	}
	metadataBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	credentials := HttpHologramCredentials{}
	err = json.Unmarshal(metadataBytes, &credentials)
	if err != nil {
		return err
	}

	// Get the federation signin token
	awsCreds := HttpAwsCredentials{
		SessionId: credentials.AccessKeyId,
		SessionKey: credentials.SecretAccessKey,
		SessionToken: credentials.Token,
	}
	awsCredsJson, err := json.Marshal(awsCreds)
	signinTokenUrl := fmt.Sprintf("%v?Action=getSigninToken&SessionDuration=43200&Session=%v", federationUrlBase, url.QueryEscape(string(awsCredsJson)))
	response, err = http.Get(signinTokenUrl)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	signinToken_bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	signinToken := HttpFederationSigninToken{}
	err = json.Unmarshal(signinToken_bytes, &signinToken)
	if err != nil {
		return err
	}

	// Get the federation login URL
	federationUrl := fmt.Sprintf("%v?Action=login&Issuer=Hologram&Destination=%v&SigninToken=%v", federationUrlBase, url.QueryEscape(awsConsoleUrl), signinToken.SigninToken)

	// if --show-url is set, print the URL
	if showUrl {
		fmt.Println(federationUrl)
	}
	// if --no-launch is set, stop here
	if noLaunch {
		return nil
	}

	// Open the URL in the browser
	var openArgs []string
	switch runtime.GOOS {
	case "darwin":
		if newSession {
			dateSeconds := time.Now().Unix()
			userDataDir := fmt.Sprintf("/tmp/hologram_session_%v/", dateSeconds)
			err := os.MkdirAll(userDataDir, 0755)
			if err != nil {
				return err
			}
			_, err = os.Create(fmt.Sprintf("%v/First Run", userDataDir))
			if err != nil {
				return err
			}
			openArgs = append(openArgs, "-na", "Google Chrome", "--args", "--user-data-dir="+userDataDir)
		}
		openArgs = append(openArgs, federationUrl)
		err = exec.Command("open", openArgs...).Run()
	case "linux":
		if newSession {
			fmt.Println("Warning: --new-session is not currently supported on Linux")
		}
		openArgs = append(openArgs, federationUrl)
		err = exec.Command("xdg-open", openArgs...).Run()
	default:
		return fmt.Errorf("unsupported OS: %v", runtime.GOOS)
	}

	return err
}