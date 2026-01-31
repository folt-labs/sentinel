package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHubRepo is the repository for releases
	GitHubRepo = "folt-labs/sentinel"
	// GitHubAPIURL is the base URL for GitHub API
	GitHubAPIURL = "https://api.github.com"
)

// Config holds updater configuration
type Config struct {
	Enabled       bool          `mapstructure:"enabled"`
	CheckInterval time.Duration `mapstructure:"check_interval"`
	Channel       string        `mapstructure:"channel"` // stable, beta
}

// DefaultConfig returns the default updater configuration
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		CheckInterval: 24 * time.Hour,
		Channel:       "stable",
	}
}

// Release represents a GitHub release
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
	Body       string  `json:"body"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	DownloadURL    string
	ChecksumURL    string
	ExpectedHash   string
	ReleaseNotes   string
}

// Updater handles agent self-updates
type Updater struct {
	cfg            Config
	currentVersion string
	httpClient     *http.Client
	restartChan    chan struct{} // Signal to main to trigger graceful restart
}

// New creates a new Updater instance
// restartChan is optional - if provided, signals when a restart is needed after update
func New(cfg Config, currentVersion string, restartChan chan struct{}) *Updater {
	return &Updater{
		cfg:            cfg,
		currentVersion: currentVersion,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		restartChan: restartChan,
	}
}

// CheckForUpdate checks if a newer version is available
func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	release, err := u.getLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	// Skip prereleases unless on beta channel
	if release.Prerelease && u.cfg.Channel != "beta" {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: u.currentVersion,
			LatestVersion:  release.TagName,
		}, nil
	}

	// Compare versions
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(u.currentVersion, "v")

	// If current is "dev", always consider update available
	needsUpdate := currentVersion == "dev" || compareVersions(latestVersion, currentVersion) > 0

	info := &UpdateInfo{
		Available:      needsUpdate,
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
	}

	if needsUpdate {
		// Find the correct asset for this architecture
		assetName := u.getAssetName()
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				info.DownloadURL = asset.BrowserDownloadURL
			}
			if asset.Name == "checksums.txt" {
				info.ChecksumURL = asset.BrowserDownloadURL
			}
		}

		if info.DownloadURL == "" {
			return nil, fmt.Errorf("no binary available for %s/%s", runtime.GOOS, runtime.GOARCH)
		}

		// Fetch expected checksum if available
		if info.ChecksumURL != "" {
			if hash, err := u.fetchExpectedChecksum(ctx, info.ChecksumURL, assetName); err == nil {
				info.ExpectedHash = hash
			} else {
				log.Printf("Warning: could not fetch checksums: %v", err)
			}
		}
	}

	return info, nil
}

// Update downloads and applies the update
func (u *Updater) Update(ctx context.Context, info *UpdateInfo) error {
	if !info.Available || info.DownloadURL == "" {
		return fmt.Errorf("no update available")
	}

	log.Printf("Downloading update %s...", info.LatestVersion)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "sentinel-agent-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up on failure

	// Download the new binary
	req, err := http.NewRequestWithContext(ctx, "GET", info.DownloadURL, nil)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Copy to temp file
	hash := sha256.New()
	writer := io.MultiWriter(tmpFile, hash)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update: %w", err)
	}
	tmpFile.Close()

	// Verify checksum
	checksum := hex.EncodeToString(hash.Sum(nil))
	log.Printf("Downloaded update, SHA256: %s", checksum)

	if info.ExpectedHash != "" {
		if checksum != info.ExpectedHash {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", info.ExpectedHash, checksum)
		}
		log.Printf("Checksum verified successfully")
	} else {
		log.Printf("Warning: no checksum available for verification")
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Verify the new binary runs
	cmd := exec.CommandContext(ctx, tmpPath, "--version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("new binary verification failed: %w, output: %s", err, output)
	}

	// Backup current binary
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	log.Printf("Update installed successfully. Restarting...")

	// Signal systemd to restart
	return u.restart()
}

// Run starts the background update checker
func (u *Updater) Run(ctx context.Context) {
	if !u.cfg.Enabled {
		log.Println("Auto-update disabled")
		return
	}

	log.Printf("Auto-update enabled, checking every %s", u.cfg.CheckInterval)

	// Add random jitter (0-60 minutes) to spread updates across fleet
	jitter := time.Duration(rand.Int63n(int64(60 * time.Minute)))
	initialDelay := 5*time.Minute + jitter

	log.Printf("First update check in %s", initialDelay.Round(time.Second))

	// Wait for initial delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(initialDelay):
	}

	// Run first check
	u.checkAndUpdate(ctx)

	// Start periodic checks
	ticker := time.NewTicker(u.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.checkAndUpdate(ctx)
		}
	}
}

func (u *Updater) checkAndUpdate(ctx context.Context) {
	log.Println("Checking for updates...")

	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		log.Printf("Update check failed: %v", err)
		return
	}

	if !info.Available {
		log.Printf("Already running latest version (%s)", info.CurrentVersion)
		return
	}

	log.Printf("Update available: %s -> %s", info.CurrentVersion, info.LatestVersion)

	if err := u.Update(ctx, info); err != nil {
		log.Printf("Update failed: %v", err)
	}
}

func (u *Updater) getLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPIURL, GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "sentinel-agent/"+u.currentVersion)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	return &release, nil
}

func (u *Updater) getAssetName() string {
	arch := runtime.GOARCH
	if arch == "arm64" || arch == "aarch64" {
		arch = "arm64"
	} else if arch == "amd64" || arch == "x86_64" {
		arch = "amd64"
	}
	return fmt.Sprintf("sentinel-agent-linux-%s", arch)
}

func (u *Updater) fetchExpectedChecksum(ctx context.Context, checksumURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch checksums: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse checksums.txt format: "hash  filename"
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == assetName {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("checksum not found for %s", assetName)
}

func (u *Updater) restart() error {
	// The agent runs as non-root user, so we can't use systemctl restart.
	// Instead, we signal the main loop to shutdown gracefully. Since the
	// systemd service has Restart=always, systemd will restart us with the new binary.
	log.Println("Signaling graceful restart for update...")
	if u.restartChan != nil {
		close(u.restartChan)
	} else {
		// Fallback if no channel provided (e.g., manual update command)
		os.Exit(0)
	}
	return nil
}

// compareVersions compares two semantic versions
// Returns: 1 if a > b, -1 if a < b, 0 if equal
func compareVersions(a, b string) int {
	aParts := parseVersion(a)
	bParts := parseVersion(b)

	for i := 0; i < 3; i++ {
		if aParts[i] > bParts[i] {
			return 1
		}
		if aParts[i] < bParts[i] {
			return -1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	var parts [3]int
	v = strings.TrimPrefix(v, "v")

	// Handle versions like "0.1.3-beta"
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}

	segments := strings.Split(v, ".")
	for i := 0; i < len(segments) && i < 3; i++ {
		fmt.Sscanf(segments[i], "%d", &parts[i])
	}
	return parts
}
