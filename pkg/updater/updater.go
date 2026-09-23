package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"lingo-translate/pkg/version"
)

const (
	repoOwner = "ProjectErotic"
	repoName  = "Lingo-Translate"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// CheckUpdate checks GitHub Releases to see if a newer version is available.
func CheckUpdate() (hasUpdate bool, currentVer string, latestVer string, err error) {
	currentVer = version.Version
	latest, err := fetchLatestRelease()
	if err != nil {
		return false, currentVer, "", err
	}

	latestVer = strings.TrimPrefix(latest.TagName, "v")
	cleanCurrent := strings.TrimPrefix(currentVer, "v")

	if isNewer(latestVer, cleanCurrent) {
		return true, currentVer, latestVer, nil
	}
	return false, currentVer, latestVer, nil
}

// Update downloads and replaces the current binary with the latest release.
func Update(force bool) error {
	currentVer := version.Version
	fmt.Println("🔍 Checking for updates...")

	latest, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch release information: %w", err)
	}

	latestVer := strings.TrimPrefix(latest.TagName, "v")
	cleanCurrent := strings.TrimPrefix(currentVer, "v")

	if !force && !isNewer(latestVer, cleanCurrent) {
		fmt.Printf("✅ Lingo CLI is already up to date (current: v%s, latest: v%s)\n", cleanCurrent, latestVer)
		return nil
	}

	fmt.Printf("📦 Found version v%s (current: v%s)\n", latestVer, cleanCurrent)

	// Determine matching asset for current OS and Architecture
	asset := findMatchingAsset(latest.Assets, runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		return fmt.Errorf("no release asset found matching OS '%s' and ARCH '%s'", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("⬇️  Downloading %s (%.2f MB)...\n", asset.Name, float64(asset.Size)/(1024*1024))

	tmpDir, err := os.MkdirTemp("", "lingo-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset.Name)
	if err := downloadFile(asset.DownloadURL, archivePath); err != nil {
		return fmt.Errorf("failed to download release: %w", err)
	}

	// Extract binary from downloaded archive
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	if strings.HasSuffix(asset.Name, ".zip") {
		if err := extractZip(archivePath, extractDir); err != nil {
			return fmt.Errorf("failed to extract zip archive: %w", err)
		}
	} else if strings.HasSuffix(asset.Name, ".tar.gz") {
		if err := extractTarGz(archivePath, extractDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz archive: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported archive format: %s", asset.Name)
	}

	// Locate the lingo binary inside extractDir
	targetBinName := "lingo"
	if runtime.GOOS == "windows" {
		targetBinName = "lingo.exe"
	}

	var newBinPath string
	err = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (info.Name() == targetBinName || info.Name() == "nst" || info.Name() == "nst.exe") {
			newBinPath = path
			return io.EOF // Stop search
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return fmt.Errorf("error finding extracted binary: %w", err)
	}
	if newBinPath == "" {
		return fmt.Errorf("binary '%s' not found inside release archive", targetBinName)
	}

	// Get current executable path
	currentExec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine current executable path: %w", err)
	}
	currentExec, err = filepath.EvalSymlinks(currentExec)
	if err != nil {
		return fmt.Errorf("failed to resolve executable symlink: %w", err)
	}

	fmt.Printf("🔄 Replacing %s with v%s...\n", currentExec, latestVer)

	if err := replaceExecutable(newBinPath, currentExec); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("🎉 Successfully updated Lingo CLI to v%s!\n", latestVer)
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Lingo-CLI-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func findMatchingAsset(assets []githubAsset, goos, goarch string) *githubAsset {
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) {
			if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
				return &a
			}
		}
	}
	return nil
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 3 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Lingo-CLI-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func replaceExecutable(newBinPath, currentExec string) error {
	newBytes, err := os.ReadFile(newBinPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(currentExec)
	tmpFile := filepath.Join(dir, fmt.Sprintf(".lingo-update-%d.tmp", time.Now().UnixNano()))

	if err := os.WriteFile(tmpFile, newBytes, 0755); err != nil {
		// If permission denied in target dir (e.g. /usr/local/bin without sudo)
		return fmt.Errorf("permission denied writing to %s (run with sudo if installed globally): %w", dir, err)
	}

	if runtime.GOOS == "windows" {
		oldBackup := currentExec + ".old"
		_ = os.Remove(oldBackup)
		if err := os.Rename(currentExec, oldBackup); err != nil {
			_ = os.Remove(tmpFile)
			return err
		}
		if err := os.Rename(tmpFile, currentExec); err != nil {
			_ = os.Rename(oldBackup, currentExec)
			return err
		}
		_ = os.Remove(oldBackup)
	} else {
		if err := os.Rename(tmpFile, currentExec); err != nil {
			_ = os.Remove(tmpFile)
			return err
		}
		_ = os.Chmod(currentExec, 0755)
	}

	return nil
}

func isNewer(latest, current string) bool {
	lParts := strings.Split(latest, ".")
	cParts := strings.Split(current, ".")

	for i := 0; i < len(lParts) && i < len(cParts); i++ {
		var lNum, cNum int
		fmt.Sscanf(lParts[i], "%d", &lNum)
		fmt.Sscanf(cParts[i], "%d", &cNum)
		if lNum > cNum {
			return true
		} else if lNum < cNum {
			return false
		}
	}
	return len(lParts) > len(cParts)
}
