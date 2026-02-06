package services

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// Use the UpdateInfo and InstallMethod types from the model package
// to avoid duplication

// UpdateService interface for version checking and updates
type UpdateService interface {
	// CheckForUpdates checks if a newer version is available
	CheckForUpdates(currentVersion string) (*model.UpdateInfo, error)

	// DetectInstallMethod attempts to detect how argonaut was installed
	DetectInstallMethod() model.InstallMethod

	// DownloadAndReplace downloads the latest version and replaces the current binary
	DownloadAndReplace(updateInfo *model.UpdateInfo) error

	// RestartApplication restarts the application after update
	RestartApplication() error
}

// UpdateServiceImpl provides concrete implementation of UpdateService
type UpdateServiceImpl struct {
	httpClient       *http.Client
	githubRepo       string
	checkIntervalMin int
}

// UpdateServiceConfig holds configuration for UpdateService
type UpdateServiceConfig struct {
	HTTPClient       *http.Client
	GitHubRepo       string // e.g., "darksworm/argonaut"
	CheckIntervalMin int    // Minutes between update checks
}

// NewUpdateService creates a new UpdateService implementation
func NewUpdateService(config UpdateServiceConfig) UpdateService {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	githubRepo := config.GitHubRepo
	if githubRepo == "" {
		githubRepo = "darksworm/argonaut"
	}

	checkInterval := config.CheckIntervalMin
	if checkInterval <= 0 {
		checkInterval = 60 // Default: check every hour
	}

	return &UpdateServiceImpl{
		httpClient:       httpClient,
		githubRepo:       githubRepo,
		checkIntervalMin: checkInterval,
	}
}

// CheckForUpdates implements UpdateService.CheckForUpdates
func (u *UpdateServiceImpl) CheckForUpdates(currentVersion string) (*model.UpdateInfo, error) {
	logger := cblog.With("component", "update")
	logger.Debug("Checking for updates", "current_version", currentVersion)

	// Fetch latest release from GitHub API
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.githubRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header as required by GitHub API
	req.Header.Set("User-Agent", "argonaut-update-checker")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	logger.Debug("Latest release", "version", release.TagName, "published", release.PublishedAt)

	// Compare versions (simple string comparison, assumes semantic versioning)
	updateAvailable := isVersionNewer(release.TagName, currentVersion)

	installMethod := u.DetectInstallMethod()

	updateInfo := &model.UpdateInfo{
		Available:        updateAvailable,
		CurrentVersion:   currentVersion,
		LatestVersion:    release.TagName,
		PublishedAt:      release.PublishedAt,
		InstallMethod:    installMethod,
		LastChecked:      time.Now(),
		CheckIntervalMin: u.checkIntervalMin,
	}

	// Find appropriate download URL when update is available
	// (now that we allow package manager users to proceed, they need URLs too)
	if updateAvailable {
		downloadURL := u.findDownloadURL(&release)
		updateInfo.DownloadURL = downloadURL
		if downloadURL != "" {
			logger.Debug("Found download URL", "url", downloadURL)
			checksumURL, checksumSHA256, checksumErr := u.fetchAssetChecksum(&release, downloadURL)
			if checksumErr != nil {
				logger.Warn("Failed to load release checksum", "err", checksumErr)
			} else if checksumSHA256 != "" {
				updateInfo.ChecksumURL = checksumURL
				updateInfo.ChecksumSHA256 = checksumSHA256
				logger.Debug("Loaded release checksum", "checksum_url", checksumURL)
			}
		} else {
			logger.Warn("No download URL found for platform", "os", runtime.GOOS, "arch", runtime.GOARCH)
		}
	}

	logger.Info("Update check completed",
		"available", updateAvailable,
		"current", currentVersion,
		"latest", release.TagName,
		"install_method", installMethod)

	return updateInfo, nil
}

// DetectInstallMethod implements UpdateService.DetectInstallMethod
func (u *UpdateServiceImpl) DetectInstallMethod() model.InstallMethod {
	logger := cblog.With("component", "update")

	// Get the path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		logger.Debug("Failed to get executable path", "err", err)
		return model.InstallMethodUnknown
	}

	logger.Debug("Detecting install method", "exec_path", execPath)

	// Check for Docker environment
	if _, err := os.Stat("/.dockerenv"); err == nil {
		logger.Debug("Detected Docker environment")
		return model.InstallMethodDocker
	}

	// Check for Homebrew installation (macOS/Linux)
	if strings.Contains(execPath, "/opt/homebrew/") ||
		strings.Contains(execPath, "/usr/local/Cellar/") ||
		strings.Contains(execPath, "/home/linuxbrew/") {
		logger.Debug("Detected Homebrew installation")
		return model.InstallMethodBrew
	}

	// Check for AUR installation (Arch Linux)
	if strings.Contains(execPath, "/usr/bin/") &&
		runtime.GOOS == "linux" {
		// Additional check for pacman database
		if _, err := os.Stat("/var/lib/pacman/local"); err == nil {
			// Check if argonaut is in pacman database
			if u.isInstalledViaPacman() {
				logger.Debug("Detected AUR/pacman installation")
				return model.InstallMethodAUR
			}
		}
	}

	// Default to manual installation
	logger.Debug("Detected manual installation")
	return model.InstallMethodManual
}

// DownloadAndReplace implements UpdateService.DownloadAndReplace
func (u *UpdateServiceImpl) DownloadAndReplace(updateInfo *model.UpdateInfo) error {
	if updateInfo == nil {
		return fmt.Errorf("update info is required")
	}
	if updateInfo.DownloadURL == "" {
		return fmt.Errorf("no download URL available")
	}
	if updateInfo.ChecksumSHA256 == "" {
		return fmt.Errorf("refusing update without SHA-256 checksum metadata")
	}

	logger := cblog.With("component", "update")
	logger.Info("Starting binary update",
		"from", updateInfo.CurrentVersion,
		"to", updateInfo.LatestVersion,
		"url", updateInfo.DownloadURL)

	// Download the new binary/archive
	resp, err := u.httpClient.Get(updateInfo.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	downloadData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read update payload: %w", err)
	}
	if err := verifySHA256(downloadData, updateInfo.ChecksumSHA256); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Determine if we're dealing with an archive or direct binary
	isArchive := strings.Contains(updateInfo.DownloadURL, ".tar.gz") || strings.Contains(updateInfo.DownloadURL, ".zip")

	if isArchive {
		// Handle archive extraction
		return u.downloadAndExtractArchive(bytes.NewReader(downloadData), execPath, updateInfo.DownloadURL)
	} else {
		// Handle direct binary download (legacy path)
		return u.downloadDirectBinary(bytes.NewReader(downloadData), execPath)
	}
}

// RestartApplication implements UpdateService.RestartApplication
func (u *UpdateServiceImpl) RestartApplication() error {
	logger := cblog.With("component", "update")
	logger.Info("Upgrade completed successfully. Please restart argonaut manually.")

	// Instead of automatically restarting (which can break the terminal),
	// just exit cleanly and let the user restart manually
	// This is safer and more predictable

	return nil // Don't actually restart - let the UI handle showing success message
}

// Helper methods

// isVersionNewer compares version strings and returns true if newVersion > currentVersion
func isVersionNewer(newVersion, currentVersion string) bool {
	// Remove 'v' prefix if present
	newVersion = strings.TrimPrefix(newVersion, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// Handle dev version
	if currentVersion == "dev" {
		return true // Always consider any release newer than dev
	}

	// Simple version comparison (assumes semantic versioning)
	newParts := strings.Split(newVersion, ".")
	currentParts := strings.Split(currentVersion, ".")

	// Pad to same length
	maxLen := len(newParts)
	if len(currentParts) > maxLen {
		maxLen = len(currentParts)
	}

	for len(newParts) < maxLen {
		newParts = append(newParts, "0")
	}
	for len(currentParts) < maxLen {
		currentParts = append(currentParts, "0")
	}

	// Compare each part
	for i := 0; i < maxLen; i++ {
		newNum, err1 := strconv.Atoi(newParts[i])
		currentNum, err2 := strconv.Atoi(currentParts[i])

		if err1 != nil || err2 != nil {
			// Fallback to string comparison
			if newParts[i] > currentParts[i] {
				return true
			} else if newParts[i] < currentParts[i] {
				return false
			}
			continue
		}

		if newNum > currentNum {
			return true
		} else if newNum < currentNum {
			return false
		}
	}

	return false // Versions are equal
}

// findDownloadURL finds the appropriate download URL for the current platform
func (u *UpdateServiceImpl) findDownloadURL(release *GitHubRelease) string {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	logger := cblog.With("component", "update")
	logger.Debug("Looking for download URL", "os", osName, "arch", archName)

	// For argonaut releases, we need to match the exact naming convention used
	// The releases use: argonaut-VERSION-OS-ARCH.tar.gz
	// Where OS is "darwin" or "linux" and ARCH is "amd64" or "arm64"

	// Look for exact platform match first (prefer tar.gz since that's what's available)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		logger.Debug("Checking asset", "name", asset.Name)

		// Check for exact OS and architecture match
		if strings.Contains(name, osName) && strings.Contains(name, archName) {
			logger.Debug("Found matching asset", "name", asset.Name, "url", asset.BrowserDownloadURL)
			return asset.BrowserDownloadURL
		}
	}

	// Fallback: try common architecture aliases
	archAliases := map[string][]string{
		"amd64": {"amd64", "x86_64", "x64"},
		"arm64": {"arm64", "aarch64", "arm_64"},
		"386":   {"386", "i386", "x86"},
	}

	if aliases, ok := archAliases[archName]; ok {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.Contains(name, osName) {
				for _, alias := range aliases {
					if strings.Contains(name, alias) {
						logger.Debug("Found matching asset with alias", "name", asset.Name, "alias", alias)
						return asset.BrowserDownloadURL
					}
				}
			}
		}
	}

	logger.Warn("No matching asset found", "os", osName, "arch", archName)
	return ""
}

// isInstalledViaPacman checks if argonaut is installed via pacman
func (u *UpdateServiceImpl) isInstalledViaPacman() bool {
	// Check if argonaut is in the pacman database
	_, err := os.Stat("/var/lib/pacman/local")
	if err != nil {
		return false
	}

	// Look for argonaut package directory
	pattern := "/var/lib/pacman/local/argonaut-*"
	matches, err := filepath.Glob(pattern)
	return err == nil && len(matches) > 0
}

// copyFile copies a file from src to dst
func (u *UpdateServiceImpl) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// moveFile moves a file from src to dst
func (u *UpdateServiceImpl) moveFile(src, dst string) error {
	// Try rename first (faster, atomic on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fallback to copy + remove
	if err := u.copyFile(src, dst); err != nil {
		return err
	}

	return os.Remove(src)
}

// downloadAndExtractArchive downloads and extracts a tar.gz archive, finding the binary inside
func (u *UpdateServiceImpl) downloadAndExtractArchive(reader io.Reader, execPath, downloadURL string) error {
	logger := cblog.With("component", "update")
	logger.Info("Downloading and extracting archive", "url", downloadURL)

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "argonaut-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract the archive
	if strings.Contains(downloadURL, ".tar.gz") {
		if err := u.extractTarGz(reader, tempDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported archive format: %s", downloadURL)
	}

	// Find the argonaut binary in the extracted files
	binaryPath, err := u.findBinaryInDir(tempDir, "argonaut")
	if err != nil {
		return fmt.Errorf("failed to find binary in archive: %w", err)
	}

	logger.Info("Found binary in archive", "path", binaryPath)

	// Replace the current binary
	return u.replaceBinary(binaryPath, execPath)
}

// downloadDirectBinary downloads a direct binary file
func (u *UpdateServiceImpl) downloadDirectBinary(reader io.Reader, execPath string) error {
	logger := cblog.With("component", "update")
	logger.Info("Downloading direct binary")

	// Create temporary file for new binary
	tempFile, err := os.CreateTemp(filepath.Dir(execPath), "argonaut-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Copy downloaded content to temp file
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write update to temp file: %w", err)
	}
	tempFile.Close()

	// Make temp file executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make temp file executable: %w", err)
	}

	// Replace the current binary
	return u.replaceBinary(tempFile.Name(), execPath)
}

// extractTarGz extracts a tar.gz archive to the specified directory
func (u *UpdateServiceImpl) extractTarGz(reader io.Reader, destDir string) error {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		destPath, err := secureExtractPath(destDir, header.Name)
		if err != nil {
			return err
		}

		// Ensure the destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			// Regular file
			file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", destPath, err)
			}

			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return fmt.Errorf("failed to write file %s: %w", destPath, err)
			}
		case tar.TypeDir:
			// Directory
			if err := os.MkdirAll(destPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		default:
			return fmt.Errorf("unsupported tar entry type %q for %s (size=%d)", header.Typeflag, header.Name, header.Size)
		}
	}

	return nil
}

func verifySHA256(data []byte, expected string) error {
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return fmt.Errorf("missing expected checksum")
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return fmt.Errorf("expected %s, got %s", expected, actual)
	}
	return nil
}

func secureExtractPath(destDir, entryName string) (string, error) {
	clean := filepath.Clean(entryName)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid archive entry: %q", entryName)
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("archive entry uses absolute path: %q", entryName)
	}

	destPath := filepath.Join(destDir, clean)
	rel, err := filepath.Rel(destDir, destPath)
	if err != nil {
		return "", fmt.Errorf("failed to validate archive path %q: %w", entryName, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes destination: %q", entryName)
	}
	return destPath, nil
}

func (u *UpdateServiceImpl) fetchAssetChecksum(release *GitHubRelease, downloadURL string) (string, string, error) {
	checksumURL := findChecksumAssetURL(release)
	if checksumURL == "" {
		return "", "", fmt.Errorf("no checksum asset found in release")
	}
	assetName, err := assetNameFromURL(downloadURL)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("GET", checksumURL, nil)
	if err != nil {
		return checksumURL, "", fmt.Errorf("failed to create checksum request: %w", err)
	}
	req.Header.Set("User-Agent", "argonaut-update-checker")
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return checksumURL, "", fmt.Errorf("failed to fetch checksum asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return checksumURL, "", fmt.Errorf("checksum asset download failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return checksumURL, "", fmt.Errorf("failed to read checksum asset: %w", err)
	}
	sum := parseChecksumForAsset(string(body), assetName)
	if sum == "" {
		return checksumURL, "", fmt.Errorf("checksum for asset %q not found", assetName)
	}
	return checksumURL, sum, nil
}

func findChecksumAssetURL(release *GitHubRelease) string {
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "sha256") || strings.Contains(name, "checksums") {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

func assetNameFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse download URL: %w", err)
	}
	name := filepath.Base(u.Path)
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("could not infer asset name from download URL")
	}
	return name, nil
}

func parseChecksumForAsset(contents, assetName string) string {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			candidate := strings.TrimPrefix(parts[1], "*")
			if candidate == assetName && isHexSHA256(parts[0]) {
				return strings.ToLower(parts[0])
			}
		}

		if strings.HasPrefix(line, "SHA256(") {
			// Format: SHA256(filename)= <hash>
			closeIdx := strings.Index(line, ")=")
			if closeIdx > len("SHA256(") {
				name := strings.TrimSpace(line[len("SHA256("):closeIdx])
				hash := strings.TrimSpace(line[closeIdx+2:])
				if name == assetName && isHexSHA256(hash) {
					return strings.ToLower(hash)
				}
			}
		}
	}
	return ""
}

func isHexSHA256(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// findBinaryInDir recursively searches for a binary file with the given name
func (u *UpdateServiceImpl) findBinaryInDir(dir, binaryName string) (string, error) {
	var binaryPath string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if this is the binary we're looking for
		if !info.IsDir() && strings.Contains(info.Name(), binaryName) {
			// Check if it's executable (or has no extension, indicating it might be a binary)
			if info.Mode()&0111 != 0 || !strings.Contains(info.Name(), ".") {
				binaryPath = path
				return filepath.SkipDir // Found it, stop searching
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if binaryPath == "" {
		return "", fmt.Errorf("binary '%s' not found in directory", binaryName)
	}

	return binaryPath, nil
}

// replaceBinary replaces the current binary with a new one, with backup
func (u *UpdateServiceImpl) replaceBinary(newBinaryPath, execPath string) error {
	logger := cblog.With("component", "update")

	// Check if we have write permissions to the directory and file
	if err := u.checkWritePermissions(execPath); err != nil {
		return fmt.Errorf("insufficient permissions to upgrade: %w\n\nTo fix this:\n• Run with sudo: sudo argonaut :upgrade\n• Or move argonaut to a user-writable location\n• Or upgrade manually from: https://github.com/darksworm/argonaut/releases/latest", err)
	}

	// Create backup of current binary
	backupPath := execPath + ".backup"
	if err := u.copyFile(execPath, backupPath); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied creating backup: %w\n\nTry running with elevated permissions: sudo argonaut :upgrade", err)
		}
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace current binary with new one
	if err := u.moveFile(newBinaryPath, execPath); err != nil {
		// Restore backup on failure
		u.moveFile(backupPath, execPath)

		if os.IsPermission(err) {
			return fmt.Errorf("permission denied replacing binary: %w\n\nTry running with elevated permissions: sudo argonaut :upgrade\nOr upgrade manually from: https://github.com/darksworm/argonaut/releases/latest", err)
		}
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove backup on success
	os.Remove(backupPath)

	logger.Info("Binary replacement completed successfully")
	return nil
}

// checkWritePermissions checks if we have write permissions to upgrade the binary
func (u *UpdateServiceImpl) checkWritePermissions(execPath string) error {
	// Check if the executable file is writable
	fileInfo, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("cannot access executable: %w", err)
	}

	// Check file permissions
	if fileInfo.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("executable is not writable")
	}

	// Check if the directory is writable (needed for creating backup and temp files)
	dir := filepath.Dir(execPath)
	if err := u.testDirectoryWrite(dir); err != nil {
		return fmt.Errorf("directory '%s' is not writable: %w", dir, err)
	}

	return nil
}

// testDirectoryWrite tests if we can write to a directory
func (u *UpdateServiceImpl) testDirectoryWrite(dir string) error {
	tempFile, err := os.CreateTemp(dir, "argonaut-permission-test-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	return nil
}
