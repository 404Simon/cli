package login

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type HostingOption string

const (
	HostingOptionCloud      HostingOption = "cloud"
	HostingOptionSelfHosted HostingOption = "self-hosted"
)

type LoginMethod string

const (
	LoginMethodCredentials LoginMethod = "credentials"
	LoginMethodWeb         LoginMethod = "web"
)

func loginWithCredentials(hostname string) (string, error) {
	// Build base URL for login (use hostname as-is, LoginWithCookie will add /api/v1/auth/login)
	baseURL := hostname

	// Create a temporary API client for login (without auth)
	loginClient, err := api.NewClient(api.ClientConfig{
		BaseURL:           baseURL,
		AgentName:         "pangolin-cli",
		SessionCookieName: "p_session_token",
		CSRFToken:         "x-csrf-protection",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	// Prompt for email and password
	var email, password string
	credentialsForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Email").
				Placeholder("your.email@example.com").
				Value(&email),
			huh.NewInput().
				Title("Password").
				Placeholder("Enter your password").
				EchoMode(huh.EchoModePassword).
				Value(&password),
		),
	)

	if err := credentialsForm.Run(); err != nil {
		return "", fmt.Errorf("error collecting credentials: %w", err)
	}

	// Perform login
	loginReq := api.LoginRequest{
		Email:    email,
		Password: password,
	}

	loginResp, sessionToken, err := api.LoginWithCookie(loginClient, loginReq)
	if err != nil {
		return "", err
	}

	// Handle nil response (shouldn't happen, but be safe)
	if loginResp == nil {
		if sessionToken != "" {
			// If we got a token, consider it successful
			return sessionToken, nil
		}
		return "", fmt.Errorf("login failed - no response received")
	}

	// Handle different response scenarios
	if loginResp.TwoFactorSetupRequired {
		return "", fmt.Errorf("two-factor authentication setup is required. Please complete setup in the web interface")
	}

	if loginResp.UseSecurityKey {
		return "", fmt.Errorf("security key authentication is required. This is not yet supported in the CLI")
	}

	if loginResp.CodeRequested {
		// Prompt for 2FA code
		var code string
		codeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Two-factor authentication code").
					Placeholder("Enter your 2FA code").
					Value(&code),
			),
		)

		if err := codeForm.Run(); err != nil {
			return "", fmt.Errorf("error collecting 2FA code: %w", err)
		}

		// Retry login with code
		loginReq.Code = code
		loginResp, sessionToken, err = api.LoginWithCookie(loginClient, loginReq)
		if err != nil {
			return "", err
		}
	}

	if loginResp.EmailVerificationRequired {
		utils.Info("Email verification is required. Please check your email and verify your account.")
		// Still save the token if we got one
		if sessionToken != "" {
			return sessionToken, nil
		}
		return "", fmt.Errorf("email verification required but no session token received")
	}

	return sessionToken, nil
}

func loginWithWeb(hostname string) (string, error) {
	// TODO: Implement web login
	return "", fmt.Errorf("web login is not yet implemented. Please use credentials login for now")
}

var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Pangolin",
	Long:  "Interactive login to select your hosting option and configure access.",
	Run: func(cmd *cobra.Command, args []string) {
		var hostingOption HostingOption
		var hostname string
		var loginMethod LoginMethod

		// First question: select hosting option
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[HostingOption]().
					Title("Select your hosting option").
					Options(
						huh.NewOption("Pangolin Cloud (app.pangolin.net)", HostingOptionCloud),
						huh.NewOption("Self-hosted or Dedicated instance", HostingOptionSelfHosted),
					).
					Value(&hostingOption),
			),
		)

		if err := form.Run(); err != nil {
			utils.Error("Error: %v", err)
			return
		}

		// If self-hosted, prompt for hostname
		if hostingOption == HostingOptionSelfHosted {
			hostnameForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter hostname URL").
						Placeholder("https://your-instance.example.com").
						Value(&hostname),
				),
			)

			if err := hostnameForm.Run(); err != nil {
				utils.Error("Error: %v", err)
				return
			}
		} else {
			// For cloud, set the default hostname
			hostname = "app.pangolin.net"
		}

		// Normalize hostname (preserve protocol, remove trailing slash)
		hostname = strings.TrimSuffix(hostname, "/")

		// If no protocol specified, default to https
		if !strings.HasPrefix(hostname, "http://") && !strings.HasPrefix(hostname, "https://") {
			hostname = "https://" + hostname
		}

		// Store hostname in viper config (with protocol)
		viper.Set("hostname", hostname)

		// Ensure config type is set and file path is correct
		if viper.ConfigFileUsed() == "" {
			// Config file doesn't exist yet, set the full path
			homeDir, err := os.UserHomeDir()
			if err == nil {
				viper.SetConfigFile(homeDir + "/.pangolin.yaml")
				viper.SetConfigType("yaml")
			}
		}

		if err := viper.WriteConfig(); err != nil {
			// If config file doesn't exist, create it
			if err := viper.SafeWriteConfig(); err != nil {
				utils.Warning("Failed to save hostname to config: %v", err)
			}
		}

		// Select login method
		loginMethodForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[LoginMethod]().
					Title("Select login method").
					Options(
						huh.NewOption("Login with web (recommended)", LoginMethodWeb),
						huh.NewOption("Login with credentials", LoginMethodCredentials),
					).
					Value(&loginMethod),
			),
		)

		if err := loginMethodForm.Run(); err != nil {
			utils.Error("Error: %v", err)
			return
		}

		// Branch based on login method
		var sessionToken string
		var err error

		if loginMethod == LoginMethodWeb {
			sessionToken, err = loginWithWeb(hostname)
		} else {
			sessionToken, err = loginWithCredentials(hostname)
		}

		if err != nil {
			utils.Error("%v", err)
			return
		}

		if sessionToken == "" {
			utils.Error("Login appeared successful but no session token was received.")
			return
		}

		// Save session token to keyring
		if err := api.SaveSessionToken(sessionToken); err != nil {
			utils.Error("Failed to save session token: %v", err)
			return
		}

		// Update the global API client (always initialized)
		// Update base URL and token (hostname already includes protocol)
		apiBaseURL := hostname + "/api/v1"
		api.GlobalClient.SetBaseURL(apiBaseURL)
		api.GlobalClient.SetToken(sessionToken)

		// Get user information
		var user *api.User
		user, err = api.GlobalClient.GetUser()
		if err != nil {
			utils.Warning("Failed to get user information: %v", err)
		} else {
			// Store userId and email in viper config
			viper.Set("userId", user.UserID)
			viper.Set("email", user.Email)
			if err := viper.WriteConfig(); err != nil {
				utils.Warning("Failed to save user information to config: %v", err)
			}
		}

		// List and select organization
		if user != nil {
			if _, err := utils.SelectOrg(user.UserID); err != nil {
				utils.Warning("%v", err)
			}
		}

		utils.Success("Login successful!")
	},
}
