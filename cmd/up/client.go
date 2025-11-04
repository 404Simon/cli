package up

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	"github.com/fosrl/newt/logger"
	olmpkg "github.com/fosrl/olm/olm"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagID            string
	flagSecret        string
	flagEndpoint      string
	flagMTU           int
	flagDNS           string
	flagInterfaceName string
	flagLogLevel      string
	flagEnableAPI     bool
	flagHTTPAddr      string
	flagSocketPath    string
	flagPingInterval  string
	flagPingTimeout   string
	flagHolepunch     bool
	flagTlsClientCert string
	flagVersion       string
	flagDetached      bool
	flagLogFile       string
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start a client connection",
	Long:  "Bring up a client tunneled connection",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if a client is already running
		olmClient := olm.NewClient("")
		if olmClient.IsRunning() {
			// Try to get status to confirm it's actually running
			status, err := olmClient.GetStatus()
			if err == nil && status != nil {
				utils.Info("A client is already running")
				os.Exit(1)
			}
			// If status check fails but socket exists, still warn
			utils.Error("A client appears to be running (socket exists)")
			os.Exit(1)
		}

		// Validate required flags before detached mode check
		if flagID == "" {
			utils.Error("--id is required")
			os.Exit(1)
		}
		if flagSecret == "" {
			utils.Error("--secret is required")
			os.Exit(1)
		}

		// Handle log file setup - if detached mode, always use log file
		logFile := flagLogFile
		if flagDetached && logFile == "" {
			logFile = utils.GetDefaultLogPath()
		}

		// Handle detached mode - subprocess self without -d flag
		if flagDetached {
			executable, err := os.Executable()
			if err != nil {
				utils.Error("Error: failed to get executable path: %v", err)
				os.Exit(1)
			}

			// Build command arguments, excluding -d flag
			cmdArgs := []string{"up", "client"}

			// Add all flags that were set (except -d)
			// Required flags are always included
			cmdArgs = append(cmdArgs, "--id", flagID)
			cmdArgs = append(cmdArgs, "--secret", flagSecret)

			// Optional flags - only include if they were explicitly set
			if cmd.Flags().Changed("endpoint") {
				cmdArgs = append(cmdArgs, "--endpoint", flagEndpoint)
			}
			if cmd.Flags().Changed("mtu") {
				cmdArgs = append(cmdArgs, "--mtu", fmt.Sprintf("%d", flagMTU))
			}
			if cmd.Flags().Changed("dns") {
				cmdArgs = append(cmdArgs, "--dns", flagDNS)
			}
			if cmd.Flags().Changed("interface-name") {
				cmdArgs = append(cmdArgs, "--interface-name", flagInterfaceName)
			}
			if cmd.Flags().Changed("log-level") {
				cmdArgs = append(cmdArgs, "--log-level", flagLogLevel)
			}
			if cmd.Flags().Changed("enable-api") {
				if flagEnableAPI {
					cmdArgs = append(cmdArgs, "--enable-api")
				}
				// Note: If enable-api is set to false, we can't pass --no-enable-api
				// The subprocess will use its default (true) if not specified
			}
			if cmd.Flags().Changed("http-addr") {
				cmdArgs = append(cmdArgs, "--http-addr", flagHTTPAddr)
			}
			if cmd.Flags().Changed("socket-path") {
				cmdArgs = append(cmdArgs, "--socket-path", flagSocketPath)
			}
			if cmd.Flags().Changed("ping-interval") {
				cmdArgs = append(cmdArgs, "--ping-interval", flagPingInterval)
			}
			if cmd.Flags().Changed("ping-timeout") {
				cmdArgs = append(cmdArgs, "--ping-timeout", flagPingTimeout)
			}
			if cmd.Flags().Changed("holepunch") {
				if flagHolepunch {
					cmdArgs = append(cmdArgs, "--holepunch")
				}
			}
			if cmd.Flags().Changed("tls-client-cert") {
				cmdArgs = append(cmdArgs, "--tls-client-cert", flagTlsClientCert)
			}
			if cmd.Flags().Changed("version") {
				cmdArgs = append(cmdArgs, "--version", flagVersion)
			}
			// Always add log-file when detached (use default if not explicitly set)
			cmdArgs = append(cmdArgs, "--log-file", logFile)

			// Add positional args if any
			cmdArgs = append(cmdArgs, args...)

			// Create command
			procCmd := exec.Command(executable, cmdArgs...)
			procCmd.Stdin = nil
			procCmd.Stdout = nil
			procCmd.Stderr = nil

			// Set process group for proper detachment (Unix-like systems)
			if runtime.GOOS != "windows" {
				procCmd.SysProcAttr = &syscall.SysProcAttr{
					Setpgid: true,
					Pgid:    0,
				}
			}

			// Start the process
			if err := procCmd.Start(); err != nil {
				utils.Error("Error: failed to start detached process: %v", err)
				os.Exit(1)
			}

			// Show live log preview and status
			statusIconStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(utils.ColorSuccess))
			statusStartingIconStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(utils.ColorWarning))
			completed, err := tui.NewLogPreview(tui.LogPreviewConfig{
				LogFile: logFile,
				Header:  "Starting up client...",
				ExitCondition: func(client *olm.Client, status *olm.StatusResponse) (bool, bool) {
					// Exit when interface is registered
					if status != nil && status.Registered {
						return true, true
					}
					return false, false
				},
				OnEarlyExit: func(client *olm.Client) {
					// Kill the subprocess if user exits early
					if client.IsRunning() {
						client.Exit()
					}
				},
				StatusFormatter: func(isRunning bool, status *olm.StatusResponse) string {
					if !isRunning || status == nil {
						icon := statusStartingIconStyle.Render("○")
						return fmt.Sprintf("%s Starting", icon)
					} else if status.Registered {
						icon := statusIconStyle.Render("✓")
						return fmt.Sprintf("%s Registered", icon)
					}
					icon := statusStartingIconStyle.Render("○")
					return fmt.Sprintf("%s Starting", icon)
				},
			})
			if err != nil {
				utils.Error("Error: %v", err)
				os.Exit(1)
			}

			// Clear the TUI lines (header + blank + 5 logs + blank + status = 9 lines)
			// Move cursor up 9 lines, then clear from cursor to end of screen
			fmt.Print("\033[9A\r\033[0J") // Move up 9 lines, go to start of line, clear to end

			// Check if the process completed successfully or was killed
			if !completed {
				// User exited early - subprocess was killed
				utils.Info("Client process killed")
			} else {
				// Completed successfully
				utils.Success("Client interface created successfully")
			}
			os.Exit(0)
		}

		// Helper function to get value with precedence: CLI flag > config > default
		getString := func(flagValue, flagName, configKey, defaultValue string) string {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			// Check config file (if key exists)
			configPath := "olm_defaults." + configKey
			if viper.IsSet(configPath) {
				return viper.GetString(configPath)
			}
			return defaultValue
		}

		getInt := func(flagValue int, flagName, configKey string, defaultValue int) int {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			// Check config file (if key exists)
			configPath := "olm_defaults." + configKey
			if viper.IsSet(configPath) {
				return viper.GetInt(configPath)
			}
			return defaultValue
		}

		getBool := func(flagValue bool, flagName, configKey string, defaultValue bool) bool {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			// Check config file (if key exists)
			configPath := "olm_defaults." + configKey
			if viper.IsSet(configPath) {
				return viper.GetBool(configPath)
			}
			return defaultValue
		}

		// Parse duration strings to time.Duration
		parseDuration := func(durationStr string, defaultDuration time.Duration) time.Duration {
			if durationStr == "" {
				return defaultDuration
			}
			d, err := time.ParseDuration(durationStr)
			if err != nil {
				utils.Warning("Invalid duration format '%s', using default: %v", durationStr, defaultDuration)
				return defaultDuration
			}
			return d
		}

		// Get endpoint from hostname (not from olm_defaults)
		endpoint := flagEndpoint
		if endpoint == "" {
			endpoint = viper.GetString("hostname")
		}

		// Get values with precedence: CLI flag > config > default
		mtu := getInt(flagMTU, "mtu", "mtu", 1280)
		dns := getString(flagDNS, "dns", "dns", "8.8.8.8")
		interfaceName := getString(flagInterfaceName, "interface-name", "interface_name", "olm")
		logLevel := getString(flagLogLevel, "log-level", "log_level", "info")
		enableAPI := getBool(flagEnableAPI, "enable-api", "enable_api", true)
		httpAddr := getString(flagHTTPAddr, "http-addr", "http_addr", "")
		socketPath := getString(flagSocketPath, "socket-path", "socket_path", "/var/run/olm.sock")
		pingInterval := getString(flagPingInterval, "ping-interval", "ping_interval", "5s")
		pingTimeout := getString(flagPingTimeout, "ping-timeout", "ping_timeout", "5s")
		holepunch := getBool(flagHolepunch, "holepunch", "holepunch", false)
		tlsClientCert := getString(flagTlsClientCert, "tls-client-cert", "tls_client_cert", "")
		version := getString(flagVersion, "version", "version", "1")

		// Parse durations
		pingIntervalDuration := parseDuration(pingInterval, 5*time.Second)
		pingTimeoutDuration := parseDuration(pingTimeout, 5*time.Second)

		// Setup log file if specified
		if logFile != "" {
			if err := setupLogFile(logFile); err != nil {
				utils.Error("Error: failed to setup log file: %v", err)
				os.Exit(1)
			}
		}

		olmConfig := olmpkg.Config{
			Endpoint:             endpoint,
			ID:                   flagID,
			Secret:               flagSecret,
			MTU:                  mtu,
			DNS:                  dns,
			InterfaceName:        interfaceName,
			LogLevel:             logLevel,
			EnableAPI:            enableAPI,
			HTTPAddr:             httpAddr,
			SocketPath:           socketPath,
			PingInterval:         pingInterval,
			PingTimeout:          pingTimeout,
			Holepunch:            holepunch,
			TlsClientCert:        tlsClientCert,
			PingIntervalDuration: pingIntervalDuration,
			PingTimeoutDuration:  pingTimeoutDuration,
			Version:              version,
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		olmpkg.Run(ctx, olmConfig)
	},
}

func init() {
	// Required flags
	clientCmd.Flags().StringVar(&flagID, "id", "", "Client ID (required)")
	clientCmd.Flags().StringVar(&flagSecret, "secret", "", "Client secret (required)")

	// Optional flags
	clientCmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "Client endpoint (defaults to hostname from config)")
	clientCmd.Flags().IntVar(&flagMTU, "mtu", 0, "MTU (default: 1280)")
	clientCmd.Flags().StringVar(&flagDNS, "dns", "", "DNS server (default: 8.8.8.8)")
	clientCmd.Flags().StringVar(&flagInterfaceName, "interface-name", "", "Interface name (default: olm)")
	clientCmd.Flags().StringVar(&flagLogLevel, "log-level", "", "Log level (default: info)")
	clientCmd.Flags().BoolVar(&flagEnableAPI, "enable-api", false, "Enable API (default: true)")
	clientCmd.Flags().StringVar(&flagHTTPAddr, "http-addr", "", "HTTP address")
	clientCmd.Flags().StringVar(&flagSocketPath, "socket-path", "", "Socket path (default: /var/run/olm.sock)")
	clientCmd.Flags().StringVar(&flagPingInterval, "ping-interval", "", "Ping interval (default: 5s)")
	clientCmd.Flags().StringVar(&flagPingTimeout, "ping-timeout", "", "Ping timeout (default: 5s)")
	clientCmd.Flags().BoolVar(&flagHolepunch, "holepunch", false, "Enable holepunching (default: false)")
	clientCmd.Flags().StringVar(&flagTlsClientCert, "tls-client-cert", "", "TLS client certificate path")
	clientCmd.Flags().StringVar(&flagVersion, "version", "", "Version (default: 1)")
	clientCmd.Flags().BoolVarP(&flagDetached, "detached", "d", false, "Run in detached mode (background)")
	clientCmd.Flags().StringVar(&flagLogFile, "log-file", "", "Path to log file (required when using -d)")

	UpCmd.AddCommand(clientCmd)
}

// setupLogFile sets up file logging with rotation
func setupLogFile(logPath string) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Rotate log file if needed
	err = rotateLogFile(logDir, logPath)
	if err != nil {
		// Log warning but continue
		log.Printf("Warning: failed to rotate log file: %v", err)
	}

	// Open log file for appending
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Set the logger output
	logger.GetLogger().SetOutput(file)

	log.Printf("Logging to file: %s", logPath)
	return nil
}

// rotateLogFile handles daily log rotation
func rotateLogFile(logDir string, logFile string) error {
	// Get current log file info
	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No current log file to rotate
		}
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	// Check if log file is from today
	now := time.Now()
	fileTime := info.ModTime()

	// If the log file is from today, no rotation needed
	if now.Year() == fileTime.Year() && now.YearDay() == fileTime.YearDay() {
		return nil
	}

	// Create rotated filename with date
	rotatedName := fmt.Sprintf("client-%s.log", fileTime.Format("2006-01-02"))
	rotatedPath := filepath.Join(logDir, rotatedName)

	// Rename current log file to dated filename
	err = os.Rename(logFile, rotatedPath)
	if err != nil {
		return fmt.Errorf("failed to rotate log file: %v", err)
	}

	// Clean up old log files (keep last 30 days)
	cleanupOldLogFiles(logDir, 30)
	return nil
}

// cleanupOldLogFiles removes log files older than specified days
func cleanupOldLogFiles(logDir string, daysToKeep int) {
	cutoff := time.Now().AddDate(0, 0, -daysToKeep)
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "client-") && strings.HasSuffix(file.Name(), ".log") {
			filePath := filepath.Join(logDir, file.Name())
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filePath)
			}
		}
	}
}

