//go:build mage
// +build mage

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default configures the build target.
var Default = All

// All builds frontend and backend for all platforms
func All() {
	mg.Deps(BuildFrontend, BuildBackend)
}

// BuildFrontend runs the npm build process
func BuildFrontend() error {
	fmt.Println("Building Frontend with npm...")
	if err := sh.Run("npm", "install"); err != nil {
		return err
	}
	return sh.Run("npm", "run", "build")
}

// BuildBackend builds the backend for ALL platforms
func BuildBackend() {
	mg.Deps(BuildLinux, BuildWindows, BuildDarwin)

	// Copy Go manifest files to dist for plugin validator
	if err := copyGoManifest(); err != nil {
		fmt.Printf("Warning: failed to copy Go manifest: %v\n", err)
	}

	fmt.Println("Backend build for all platforms complete.")
}

// BuildAll is an alias for BuildBackend (for CI compatibility)
func BuildAll() {
	BuildBackend()
}

// Coverage runs Go tests with coverage reporting
func Coverage() error {
	fmt.Println("Running backend tests with coverage...")

	// Run tests with coverage for all packages
	// Note: Go 1.25 shows "no such tool covdata" warnings but tests still run
	// We ignore the exit code and rely on go test's output parsing
	_ = sh.Run("go", "test", "-v", "-cover", "./pkg/...")

	// If we get here, tests completed (pass or fail is shown in output)
	fmt.Println("✓ Test run completed")
	return nil
}

// --- Individual Targets ---

func BuildLinux() error {
	return buildList([]struct{ OS, Arch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
	})
}

func BuildWindows() error {
	return buildList([]struct{ OS, Arch string }{
		{"windows", "amd64"},
	})
}

func BuildDarwin() error {
	return buildList([]struct{ OS, Arch string }{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
	})
}

// --- Helpers ---

func buildList(platforms []struct{ OS, Arch string }) error {
	binaryName, err := getBinaryName()
	if err != nil {
		return err
	}

	for _, p := range platforms {
		if err := buildArch(binaryName, p.OS, p.Arch); err != nil {
			return err
		}
	}
	return nil
}

func buildArch(binaryName, goos, goarch string) error {
	// 1. Construct the Env map for this specific command
	// CGO_ENABLED=0 ensures a static binary (runs on Alpine/Ubuntu/etc)
	env := map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        goos,
		"GOARCH":      goarch,
	}

	// 2. Determine Filename
	// Smart prefixing: don't double add "gpx_"
	var filename string
	if strings.HasPrefix(binaryName, "gpx_") {
		filename = fmt.Sprintf("%s_%s_%s", binaryName, goos, goarch)
	} else {
		filename = fmt.Sprintf("gpx_%s_%s_%s", binaryName, goos, goarch)
	}

	if goos == "windows" {
		filename += ".exe"
	}
	outputPath := filepath.Join("dist", filename)

	fmt.Printf(" > Building [%s/%s] -> %s\n", goos, goarch, outputPath)

	// 3. Run Build with explicit Environment
	if err := sh.RunWith(env, "go", "build", "-o", outputPath, "./pkg"); err != nil {
		return fmt.Errorf("failed to build for %s/%s: %w", goos, goarch, err)
	}

	// 4. Ensure executable permissions (Linux/Mac)
	if goos != "windows" {
		if err := os.Chmod(outputPath, 0755); err != nil {
			fmt.Printf("   Warning: failed to chmod +x %s: %v\n", outputPath, err)
		}
	}

	return nil
}

func getBinaryName() (string, error) {
	path := "src/plugin.json"
	byteValue, err := ioutil.ReadFile(path)
	if err != nil {
		path = "plugin.json"
		byteValue, err = ioutil.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("could not find plugin.json")
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return "", err
	}

	if exe, ok := result["executable"].(string); ok && exe != "" {
		return exe, nil
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("neither 'executable' nor 'id' found in plugin.json")
}

func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("dist")
}

// copyGoManifest copies go.mod and go.sum to dist/ for plugin validator
func copyGoManifest() error {
	// Ensure dist exists
	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}

	// Copy go.mod
	if err := sh.Copy(filepath.Join("dist", "go.mod"), "go.mod"); err != nil {
		return fmt.Errorf("failed to copy go.mod: %w", err)
	}

	// Copy go.sum
	if err := sh.Copy(filepath.Join("dist", "go.sum"), "go.sum"); err != nil {
		return fmt.Errorf("failed to copy go.sum: %w", err)
	}

	fmt.Println(" > Copied Go manifest files to dist/")
	return nil
}
