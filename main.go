package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const INSTALLPATH = "~/bin/ollama"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestOllamaVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/ollama/ollama/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return release.TagName, nil
}

func getDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/ollama/ollama/releases/download/%s/ollama-linux-amd64.tgz",
		version)
}

func installOllama(url string) error {
	// Get home directory for temporary files
	homeDir, _ := os.UserHomeDir()
	tempFile := filepath.Join(homeDir, "ollama.tgz")

	// Download the file to temp path
	fmt.Printf("Downloading Ollama from %s...\n", url)
	curlCommand := exec.Command("curl", "-L", "-#", url, "-o", tempFile)
	curlCommand.Stdout = os.Stdout
	curlCommand.Stderr = os.Stderr
	if err := curlCommand.Run(); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Create the bin directory if it doesn't exist
	binDir := filepath.Join(homeDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Create temporary extraction directory
	tempDir := filepath.Join(homeDir, "ollama-extract")
	os.RemoveAll(tempDir)
	os.MkdirAll(tempDir, 0755)

	// Extract the binary from the tgz file
	fmt.Printf("Extracting Ollama binary...\n")
	extractCommand := exec.Command("tar", "-xzf", tempFile, "-C", tempDir)
	extractCommand.Stdout = os.Stdout
	extractCommand.Stderr = os.Stderr
	if err := extractCommand.Run(); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	// Move the extracted binary to the final location
	finalPath := filepath.Join(binDir, "ollama")
	moveCommand := exec.Command("mv", filepath.Join(tempDir, "bin", "ollama"), finalPath)
	if err := moveCommand.Run(); err != nil {
		return fmt.Errorf("failed to move binary: %w", err)
	}

	// Make it executable
	chmodCommand := exec.Command("chmod", "+x", finalPath)
	if err := chmodCommand.Run(); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Clean up temporary files
	os.Remove(tempFile)
	os.RemoveAll(tempDir)

	// Update PATH in .bashrc
	if err := updatePath(homeDir); err != nil {
		fmt.Printf("Warning: Failed to update PATH: %v\n", err)
	}

	fmt.Printf("Ollama installed successfully to %s\n", finalPath)
	fmt.Printf("Please restart your terminal OR log out and log back in to use the new version\n")
	return nil
}

func updatePath(homeDir string) error {
	pathExport := `export PATH="$HOME/bin:$PATH"`

	// List of shell configuration files to update (in order of preference)
	configFiles := []string{".zshrc", ".bash_profile", ".bashrc", ".profile"}

	for _, configFile := range configFiles {
		configPath := filepath.Join(homeDir, configFile)

		// Check if PATH export already exists in this file
		if file, err := os.Open(configPath); err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), pathExport) {
					file.Close()
					return nil // Already exists in this file
				}
			}
			file.Close()
		}
	}

	// Try to update .zshrc first (for zsh users)
	zshrcPath := filepath.Join(homeDir, ".zshrc")
	if fileExists(zshrcPath) {
		if err := appendToFile(zshrcPath, pathExport); err == nil {
			fmt.Printf("Updated .zshrc with PATH export\n")
			return nil
		}
	}

	// Try to update .bash_profile (for bash login shells)
	bashProfilePath := filepath.Join(homeDir, ".bash_profile")
	if err := appendToFile(bashProfilePath, pathExport); err == nil {
		fmt.Printf("Updated .bash_profile with PATH export\n")
		return nil
	}

	// Fallback to .bashrc
	bashrcPath := filepath.Join(homeDir, ".bashrc")
	if err := appendToFile(bashrcPath, pathExport); err == nil {
		fmt.Printf("Updated .bashrc with PATH export\n")
		return nil
	}

	// Final fallback to .profile
	profilePath := filepath.Join(homeDir, ".profile")
	if err := appendToFile(profilePath, pathExport); err == nil {
		fmt.Printf("Updated .profile with PATH export\n")
		return nil
	}

	return fmt.Errorf("failed to update any shell configuration file")
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func appendToFile(filePath, content string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer file.Close()

	if _, err := file.WriteString("\n" + content + "\n"); err != nil {
		return fmt.Errorf("failed to write to %s: %w", filePath, err)
	}

	return nil
}

func main() {
	version, err := getLatestOllamaVersion()
	if err != nil {
		fmt.Printf("Error getting latest version: %v\n", err)
		return
	}

	url := getDownloadURL(version)
	fmt.Printf("Latest Ollama version: %s\n", version)

	if err := installOllama(url); err != nil {
		fmt.Printf("Installation failed: %v\n", err)
		return
	}
}
