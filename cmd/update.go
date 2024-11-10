package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/juancwu/mi/common"
	"github.com/spf13/cobra"
)

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Auto update mi CLI. Use flag -latest to get the latest version.",
		Long:  "Auto update mi CLI. Use flag -latest to get the latest version. Updating to the latest version may have breaking changes to available commands and default behaviours.",
		RunE: func(cmd *cobra.Command, args []string) error {
			latest, err := cmd.Flags().GetBool("latest")
			if err != nil {
				return fmt.Errorf("failed to get latest flag: %w", err)
			}
			return runUpdate(latest)
		},
	}
	cmd.Flags().Bool("latest", false, "Allow updating to latest version.")
	return cmd
}

func runUpdate(allowLatest bool) error {
	// get current version
	currentVersion, err := semver.NewVersion(common.Version)
	if err != nil {
		return fmt.Errorf("invalid current version: %w", err)
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	// fetch releases from github
	releases, err := fetchReleases()
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}

	// find latest compatible version
	latestVersion, asset, err := findLatestCompatibleVersion(releases, currentVersion, allowLatest)
	if err != nil {
		return err
	}

	if latestVersion.LessThanEqual(currentVersion) {
		fmt.Println("Already at the latest version!")
		return nil
	}

	fmt.Printf("New version available: %s\n", latestVersion)
	fmt.Println("Downloading update...")

	// download and replace binary
	if err := downloadAndUpdate(asset.BrowserDownloadURL); err != nil {
		return fmt.Errorf("failed to update binary: %w", err)
	}

	fmt.Printf("Successfully updated to version %s\n", latestVersion.String())
	return nil
}

func fetchReleases() ([]Release, error) {
	resp, err := http.Get("https://api.github.com/repos/juancwu/mi/releases")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch releases: status %d: %s", resp.StatusCode, string(body))
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func findLatestCompatibleVersion(releases []Release, currentVersion *semver.Version, allowLatest bool) (*semver.Version, *Asset, error) {
	var latestVersion *semver.Version
	var selectedAsset *Asset

	assetName := fmt.Sprintf("mi_%s_%s", runtime.GOOS, runtime.GOARCH)
	assetName = strings.ToLower(assetName)

	for _, release := range releases {
		v, err := semver.NewVersion(strings.TrimPrefix(release.TagName, "v"))
		if err != nil {
			continue
		}

		// check if this is a compatible update
		if !allowLatest && v.Major() != currentVersion.Major() {
			continue
		}

		if latestVersion == nil || v.GreaterThan(latestVersion) {
			// find matching asset for current platform/arch
			for _, asset := range release.Assets {
				if strings.Contains(strings.ToLower(asset.Name), assetName) {
					latestVersion = v
					selectedAsset = &asset
					break
				}
			}
		}
	}

	if latestVersion == nil {
		return nil, nil, fmt.Errorf("no compatible version found")
	}

	if selectedAsset == nil {
		return nil, nil, fmt.Errorf("no compatible binary found for %s_%s", runtime.GOOS, runtime.GOARCH)
	}

	return latestVersion, selectedAsset, nil
}

func downloadAndUpdate(downloadURL string) error {
	// create temporary directory
	tmpDir, err := os.MkdirTemp("", "mi-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// download archive
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download archive: status %d: %s", resp.StatusCode, string(body))
	}

	// create gzip reader
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// create tar reader
	tarReader := tar.NewReader(gzReader)

	// find and extract the "mi" executable
	var foundExecutable bool
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if filepath.Base(header.Name) == "mi" {
			foundExecutable = true
			// create temporary file for the executable
			tmpFile := filepath.Join(tmpDir, "mi")
			file, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return err
			}
			file.Close()

			// get current executable path
			execPath, err := os.Executable()
			if err != nil {
				return err
			}
			execPath, err = filepath.EvalSymlinks(execPath)
			if err != nil {
				return err
			}

			// replace current binary
			if runtime.GOOS == "windows" {
				oldPath := execPath + ".old"
				if err := os.Rename(execPath, oldPath); err != nil {
					return err
				}
				if err := os.Rename(tmpFile, execPath); err != nil {
					// try to restore old binary if update fails
					os.Rename(oldPath, execPath)
					return err
				}
				os.Remove(oldPath)
			} else {
				if err := os.Rename(tmpFile, execPath); err != nil {
					return err
				}
			}
			break
		}
	}

	if !foundExecutable {
		return fmt.Errorf("no 'mi' executable found in the archive")
	}

	return nil
}
