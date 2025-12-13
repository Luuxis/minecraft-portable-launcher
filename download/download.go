package download

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ProductVersion represents version info
type ProductVersion struct {
	Name     string `json:"name"`
	Released string `json:"released"`
}

// ManifestFile represents download info for a file
type ManifestFile struct {
	SHA1 string `json:"sha1"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

// ManifestDownloads represents the downloads object
type ManifestDownloads struct {
	Raw ManifestFile `json:"raw"`
}

// ManifestEntry represents a file or directory in the manifest
type ManifestEntry struct {
	Type       string                 `json:"type"`
	Downloads  ManifestDownloads      `json:"downloads"`
	Executable bool                   `json:"executable"`
	Files      map[string]interface{} `json:"files"`
}

// Product represents a product (bootstrap, core, etc.)
type Product struct {
	Manifest ManifestFile   `json:"manifest"`
	Version  ProductVersion `json:"version"`
}

// LauncherMetadata represents the launcher metadata from Mojang
type LauncherMetadata struct {
	Bootstrap Product `json:"bootstrap"`
	Core      Product `json:"core,omitempty"`
}

// Global variable to store the executable file path found during download
var executablePath string

func DownloadLauncher(launcherURL string) error {
	metadata, err := fetchLauncherMetadata(launcherURL)
	if err != nil {
		return fmt.Errorf("failed to fetch launcher metadata: %w", err)
	}

	// Get the launcher base path
	launcherDir, err := getLauncherPath()
	if err != nil {
		return fmt.Errorf("failed to get launcher path: %w", err)
	}

	// Download the bootstrap manifest structure
	if err := downloadLauncherFiles(launcherDir, metadata.Bootstrap); err != nil {
		return fmt.Errorf("failed to download launcher files: %w", err)
	}

	fmt.Println("Launcher downloaded successfully!")
	return nil
}

func fetchLauncherMetadata(url string) (*LauncherMetadata, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var metadata LauncherMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func getLauncherPath() (string, error) {
	launcherDir := filepath.Join("Minecraft", "Minecraft Launcher")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(launcherDir, 0755); err != nil {
		return "", err
	}

	return launcherDir, nil
}

func downloadLauncherFiles(launcherDir string, product Product) error {
	resp, err := http.Get(product.Manifest.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch launcher manifest structure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var manifestStructure map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&manifestStructure); err != nil {
		return fmt.Errorf("failed to parse manifest structure: %w", err)
	}

	// Get the "files" entry from the manifest structure
	filesData, ok := manifestStructure["files"]
	if !ok {
		return fmt.Errorf("no 'files' entry in manifest structure")
	}

	filesMap, ok := filesData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("'files' entry is not a map")
	}

	// Determine the destination path based on OS
	var destinationPath string
	if runtime.GOOS == "darwin" {
		destinationPath = filepath.Join(launcherDir, "launcher.app")
		executablePath = filepath.Join(launcherDir, "launcher.app")
	} else {
		destinationPath = launcherDir
	}

	if err := os.MkdirAll(destinationPath, 0755); err != nil {
		return fmt.Errorf("failed to create launcher directory: %w", err)
	}

	// Download all files from the manifest structure
	return downloadFilesRecursive(destinationPath, "", filesMap)
}

func downloadFilesRecursive(baseDir string, currentPath string, files map[string]interface{}) error {
	for fileName, fileData := range files {
		fullPath := filepath.Join(currentPath, fileName)
		fullFilePath := filepath.Join(baseDir, fullPath)

		// Try to parse as ManifestEntry first
		var entry ManifestEntry
		data, err := json.Marshal(fileData)
		if err == nil {
			err = json.Unmarshal(data, &entry)
		}

		// Check if we have a valid file/directory entry
		if err == nil && (entry.Type == "file" || entry.Type == "directory") {
			if entry.Type == "directory" {
				// Create directory
				if err := os.MkdirAll(fullFilePath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
				}

				// Recursively download files in this directory
				if len(entry.Files) > 0 {
					if err := downloadFilesRecursive(baseDir, fullPath, entry.Files); err != nil {
						return err
					}
				}
			} else if entry.Type == "file" && entry.Downloads.Raw.URL != "" {
				// Download file
				if err := downloadFileWithHash(
					entry.Downloads.Raw.URL,
					fullFilePath,
					entry.Downloads.Raw.SHA1,
				); err != nil {
					return fmt.Errorf("failed to download file %s: %w", fullPath, err)
				}

				// Set executable permission if needed
				if entry.Executable {
					if err := os.Chmod(fullFilePath, 0755); err != nil {
						return fmt.Errorf("failed to set executable permission for %s: %w", fullPath, err)
					}
					if runtime.GOOS != "darwin" {
						executablePath = fullFilePath
					}
				}
			}
		} else {
			if nestedMap, ok := fileData.(map[string]interface{}); ok {
				if typeVal, ok := nestedMap["type"]; ok {
					if typeStr, ok := typeVal.(string); ok {
						if typeStr == "directory" {
							if err := os.MkdirAll(fullFilePath, 0755); err != nil {
								return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
							}
						}
					}
				}

				// Remove known keys and recursively process remaining items
				nestedMap2 := make(map[string]interface{})
				for k, v := range nestedMap {
					if k != "type" && k != "downloads" && k != "executable" && k != "size" && k != "files" {
						nestedMap2[k] = v
					}
				}

				if len(nestedMap2) > 0 {
					if err := downloadFilesRecursive(baseDir, fullPath, nestedMap2); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func downloadFileWithHash(url string, filePath string, expectedHash string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	// Download the file with retry
	var resp *http.Response
	var err error

	for attempts := 0; attempts < 3; attempts++ {
		resp, err = http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempts < 2 {
			fmt.Printf("Retry attempt %d...\n", attempts+1)
			time.Sleep(time.Second * 2)
		}
	}

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Create the file and calculate hash while writing
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	hash := sha1.New()
	writer := io.MultiWriter(out, hash)

	// Copy the content
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return err
	}

	// Verify the hash
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if calculatedHash != expectedHash {
		os.Remove(filePath)
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, calculatedHash)
	}

	return nil
}

// GetExecutablePath returns the path to the executable file found during download
func GetExecutablePath() string {
	return executablePath
}
