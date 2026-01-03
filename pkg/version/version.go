package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Version information (set via ldflags during build)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

const (
	githubRepo    = "sombochea/tungo"
	releasesURL   = "https://api.github.com/repos/" + githubRepo + "/releases"
	downloadURL   = "https://github.com/" + githubRepo + "/releases/download"
	clientTimeout = 30 * time.Second
)

// GetFullVersion returns the full version string
func GetFullVersion() string {
	return fmt.Sprintf("TunGo Client %s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}

// GetShortVersion returns just the version number
func GetShortVersion() string {
	return Version
}

// GetLatestRelease fetches the latest CLI release from GitHub (excluding SDK releases)
func GetLatestRelease() (*Release, error) {
	client := &http.Client{Timeout: clientTimeout}

	resp, err := client.Get(releasesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	// Filter out SDK releases (those starting with "sdk-")
	for _, release := range releases {
		if !strings.HasPrefix(release.TagName, "sdk-") {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("no CLI releases found")
}

// CheckForUpdates checks if a newer version is available
func CheckForUpdates() (bool, string, error) {
	if Version == "dev" {
		return false, "", nil // Skip update check for dev builds
	}

	latest, err := GetLatestRelease()
	if err != nil {
		return false, "", err
	}

	currentVersion := strings.TrimPrefix(Version, "v")
	latestVersion := strings.TrimPrefix(latest.TagName, "v")

	if latestVersion != currentVersion {
		return true, latest.TagName, nil
	}

	return false, "", nil
}

// DownloadAndInstall downloads and installs the latest version
func DownloadAndInstall() error {
	latest, err := GetLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Determine the asset name based on OS and architecture
	assetName := getAssetName()
	if assetName == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Find the matching asset
	var downloadURLStr string
	for _, asset := range latest.Assets {
		if asset.Name == assetName {
			downloadURLStr = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURLStr == "" {
		return fmt.Errorf("asset not found for %s", assetName)
	}

	log.Info().Str("version", latest.TagName).Msg("Downloading latest version...")

	// Download the binary
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURLStr)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Create a temporary file
	tmpPath := execPath + ".new"
	tmpFile, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Write the downloaded content
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write binary: %w", err)
	}
	tmpFile.Close()

	// Backup current binary
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Replace with new binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		os.Remove(tmpPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	log.Info().Str("version", latest.TagName).Msg("✓ Successfully upgraded!")
	log.Info().Msg("Please restart the tungo client to use the new version")

	return nil
}

// SelfUpgrade performs the upgrade and restarts the client
func SelfUpgrade(restartArgs []string) error {
	if err := DownloadAndInstall(); err != nil {
		return err
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Restart with same arguments
	log.Info().Msg("Restarting with new version...")
	cmd := exec.Command(execPath, restartArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart: %w", err)
	}

	os.Exit(0)
	return nil
}

// getAssetName returns the asset name for the current platform
func getAssetName() string {
	var osName, arch string

	switch runtime.GOOS {
	case "darwin":
		osName = "macos"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "windows"
	default:
		return ""
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return ""
	}

	if runtime.GOOS == "windows" {
		return fmt.Sprintf("tungo-%s-%s.exe", osName, arch)
	}

	return fmt.Sprintf("tungo-%s-%s", osName, arch)
}
