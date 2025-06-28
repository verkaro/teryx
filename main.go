// main.go
//
// This is the main implementation of the Teryx CLI tool.
// Teryx is a Go application designed to simplify common workflows with Fossil SCM.
//
// To build and run this tool:
// 1. Make sure you have Go installed (https://golang.org/doc/install).
// 2. Save this code as `main.go` in a new directory.
// 3. Open a terminal in that directory.
// 4. Initialize a Go module:
//    go mod init teryx
// 5. Get the cobra dependency:
//    go get github.com/spf13/cobra@latest
// 6. Build the executable:
//    go build -o teryx .
// 7. Run the tool:
//    ./teryx --help

package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// --- Helper Functions ---

// executeCommand runs an external command and connects it to the user's terminal.
// This allows for interactive prompts (like password entry for scp/sftp) and
// displays real-time output.
// It takes an optional workingDir, which, if specified, runs the command from that directory.
func executeCommand(workingDir string, commandName string, args ...string) error {
	cmd := exec.Command(commandName, args...)

	// Set the command's working directory if one is provided
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Connect the command's stdin, stdout, and stderr to the parent process.
	// This is crucial for interactive password prompts and seeing output.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("▶️  Executing: %s\n", cmd.String())

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("❌ command failed: %s", err)
	}
	return nil
}

// executeCommandWithOutput is similar to executeCommand but captures the stdout
// of the command instead of printing it directly. Used for commands like 'whoami'.
func executeCommandWithOutput(commandName string, args ...string) (string, error) {
	cmd := exec.Command(commandName, args...)
	fmt.Printf("▶️  Executing: %s\n", cmd.String())
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("❌ command failed: %s", err)
	}
	return strings.TrimSpace(string(output)), nil
}


// --- Cobra Command Definitions ---

// rootCmd is the base command when no subcommands are provided.
var rootCmd = &cobra.Command{
	Use:   "teryx",
	Short: "Teryx is a CLI tool to simplify Fossil SCM workflows.",
	Long: `A streamlined command-line tool written in Go to manage
the initialization, cloning, and transfer of Fossil SCM repositories.`,
}

// initCmd handles the 'teryx init' command.
var initCmd = &cobra.Command{
	Use:   "init <repository-name>",
	Short: "Initializes a new Fossil repository and sets up an admin user.",
	Long:  `Creates a new Fossil repository file, and a checkout directory for it. Also creates a new admin user with the specified password.`,
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the repository name.
	Run: func(cmd *cobra.Command, args []string) {
		repoArg := args[0]
		password, _ := cmd.Flags().GetString("password")
		username, _ := cmd.Flags().GetString("user")

		if password == "" {
			log.Fatal("❌ --password flag is required.")
		}
		
		// Auto-append .fossil if not present
		repoName := repoArg
		if !strings.HasSuffix(repoName, ".fossil") {
			repoName += ".fossil"
			fmt.Printf("ℹ️  Appending .fossil extension. Repository file will be: %s\n", repoName)
		}

		// If user flag is not set, get username from 'whoami'
		if username == "" {
			var err error
			username, err = executeCommandWithOutput("whoami")
			if err != nil {
				log.Fatalf("❌ Failed to get current user with 'whoami': %v", err)
			}
			fmt.Printf("ℹ️  No --user specified. Defaulting to current user: %s\n", username)
		}

		fmt.Printf("🚀 Initializing new repository '%s' for user '%s'...\n", repoName, username)

		// Create the repo file in the current directory.
		// The 'fossil new' command automatically creates an admin user with the same name as the
		// current system user and assigns a random password.
		if err := executeCommand("", "fossil", "new", repoName); err != nil {
			log.Fatalf("❌ Failed to create new repository: %v", err)
		}

		// Create a clean checkout directory
		checkoutDirName := strings.TrimSuffix(repoName, ".fossil")
		if err := os.MkdirAll(checkoutDirName, 0755); err != nil {
			log.Fatalf("❌ Failed to create checkout directory: %v", err)
		}
		
		// Path to the repo file relative to the checkout directory
		repoFilePath := filepath.Join("..", repoName)

		// Open the repository from within the new checkout directory
		if err := executeCommand(checkoutDirName, "fossil", "open", repoFilePath); err != nil {
			log.Fatalf("❌ Failed to open repository: %v", err)
		}

		// Since 'fossil new' already created the admin user, we just need to change their password.
		if err := executeCommand(checkoutDirName, "fossil", "user", "password", username, password); err != nil {
			log.Fatalf("❌ Failed to set user password: %v", err)
		}
		
		// Set the user as default for future CLI commands within this checkout.
		if err := executeCommand(checkoutDirName, "fossil", "user", "default", username); err != nil {
			log.Fatalf("❌ Failed to set default user: %v", err)
		}

		cwd, _ := os.Getwd()
		fmt.Printf("✅ Success! Repository initialized and opened in: %s\n", filepath.Join(cwd, checkoutDirName))
	},
}

// transferCmd handles the 'teryx transfer' command.
var transferCmd = &cobra.Command{
	Use:   "transfer <repository-name>",
	Short: "Transfers a repository file to a remote server using scp (or sftp fallback).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		destination, _ := cmd.Flags().GetString("destination")
		remoteUser, _ := cmd.Flags().GetString("remote-user")

		if destination == "" {
			log.Fatal("❌ --destination flag is required.")
		}

		fmt.Printf("🚀 Attempting to transfer '%s' to '%s' via scp...\n", repoName, destination)
		
		// 1. Try scp first
		err := executeCommand("", "scp", repoName, destination)
		if err != nil {
			fmt.Printf("⚠️ scp failed: %v\n", err)
			fmt.Println("ℹ️ Falling back to sftp...")

			// 2. Fallback to sftp
			// Parse destination to separate user@host from the path
			parts := strings.SplitN(destination, ":", 2)
			if len(parts) != 2 {
				log.Fatalf("❌ Invalid destination format. Expected user@host:path")
			}
			userHost := parts[0]
			remotePath := parts[1]
			
			// Construct the sftp command to run non-interactively
			// This approach pipes the 'put' command into sftp's standard input.
			sftpCommand := fmt.Sprintf("put %s %s", repoName, remotePath)
			sftpCmd := exec.Command("sftp", userHost)
			sftpCmd.Stdin = strings.NewReader(sftpCommand)
			sftpCmd.Stdout = os.Stdout
			sftpCmd.Stderr = os.Stderr

			fmt.Printf("▶️  Executing: echo \"%s\" | %s\n", sftpCommand, sftpCmd.String())
			
			if err := sftpCmd.Run(); err != nil {
				log.Fatalf("❌ sftp fallback also failed: %v", err)
			}
		}

		fmt.Println("✅ Success! Repository transferred.")
		fmt.Println("-----------------------------------------------------------------")
		fmt.Println("⚠️ IMPORTANT: Post-transfer steps required on the server!")
		fmt.Println("To allow the web server to write to the repository, you must update its permissions.")
		fmt.Println("Log into your server and run a command like the one below.")
		fmt.Printf("You may need to replace '%s' with your server's actual web user/group (e.g., 'apache', 'nginx').\n", remoteUser)
		fmt.Println()
		
		// Provide a helpful example command for the user to run on the server
		parts := strings.SplitN(destination, ":", 2)
		userHost := parts[0]
		remotePath := filepath.Join(parts[1], repoName) // Get the full remote path
		
		// Use "ssh -t" to force a pseudo-terminal allocation, allowing sudo to prompt for a password.
		fmt.Printf("ssh -t %s \"sudo chown %s:%s %s && sudo chmod 664 %s\"\n", userHost, remoteUser, remoteUser, remotePath, remotePath)
		fmt.Println("-----------------------------------------------------------------")
	},
}

// cloneCmd handles the 'teryx clone' command.
var cloneCmd = &cobra.Command{
	Use:   "clone <fossil-url>",
	Short: "Clones a remote repo into a structured local directory.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fossilURL := args[0]

		// BUG FIX: Strip the trailing '/home' from the URL if it exists, as this
		// is part of the web UI but not the actual clone URL.
		cleanURL := strings.TrimSuffix(fossilURL, "/home")

		fmt.Printf("🚀 Cloning from '%s'...\n", cleanURL)

		// Parse the URL
		parsedURL, err := url.Parse(cleanURL)
		if err != nil {
			log.Fatalf("❌ Invalid URL: %v", err)
		}

		// Get current user for home directory and username
		currentUser, err := user.Current()
		if err != nil {
			log.Fatalf("❌ Could not get current user: %v", err)
		}
		homeDir := currentUser.HomeDir
		username := currentUser.Username

		// Construct local target directory path: $HOME/fossils/<hostname>/<path>
		hostname := parsedURL.Hostname()
		urlPath := strings.TrimPrefix(parsedURL.Path, "/")
		targetDir := filepath.Join(homeDir, "fossils", hostname, filepath.Dir(urlPath))
		
		fmt.Printf("ℹ️  Local target directory will be: %s\n", targetDir)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Fatalf("❌ Failed to create target directory: %v", err)
		}

		// Construct new URL with username for authentication
		parsedURL.User = url.User(username)
		authURL := parsedURL.String()

		// Determine repository base name
		repoBaseName := strings.TrimSuffix(filepath.Base(urlPath), ".fossil")
		fossilFileName := repoBaseName + ".fossil"
		
		// Execute 'fossil clone' in the target directory
		if err := executeCommand(targetDir, "fossil", "clone", authURL, fossilFileName); err != nil {
			log.Fatalf("❌ Failed to clone repository: %v", err)
		}

		// Create and move into the checkout directory
		checkoutDir := filepath.Join(targetDir, repoBaseName)
		if err := os.MkdirAll(checkoutDir, 0755); err != nil {
			log.Fatalf("❌ Failed to create checkout directory: %v", err)
		}

		// Open the repository in the checkout directory
		repoFilePath := filepath.Join("..", fossilFileName)
		if err := executeCommand(checkoutDir, "fossil", "open", repoFilePath); err != nil {
			log.Fatalf("❌ Failed to open repository in checkout directory: %v", err)
		}

		fmt.Printf("✅ Success! Repo cloned and opened in: %s\n", checkoutDir)
	},
}


// --- Main Function ---

func main() {
	// --- Add flags to commands ---
	initCmd.Flags().StringP("password", "p", "", "Password for the new admin user (required)")
	initCmd.Flags().StringP("user", "u", "", "Admin username (defaults to current user)")
	
	transferCmd.Flags().StringP("destination", "d", "", "Remote destination in user@host:path format (required)")
	transferCmd.Flags().StringP("remote-user", "r", "www-data", "User/group for the web server on the remote host")

	// --- Add commands to root ---
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(cloneCmd)

	// --- Execute the root command ---
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

