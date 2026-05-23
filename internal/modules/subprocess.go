package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SubprocessRequest is the JSON sent to subprocess modules via stdin
type SubprocessRequest struct {
	Action  string                 `json:"action"`
	Config  map[string]interface{} `json:"config"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// SubprocessResponse is the JSON read from subprocess modules via stdout
type SubprocessResponse struct {
	Output map[string]interface{} `json:"output"`
	Error  string                 `json:"error,omitempty"`
}

// SubprocessModule implements Module by running an external binary
type SubprocessModule struct {
	Name       string
	BinaryPath string
	metadata   *ModuleMetadata
}

// NewSubprocessModule creates a new subprocess module
func NewSubprocessModule(name, binaryPath string) *SubprocessModule {
	return &SubprocessModule{
		Name:       name,
		BinaryPath: binaryPath,
	}
}

// GoRunModule implements Module by running via 'go run' on a GitHub repo
type GoRunModule struct {
	Name     string
	Repo     string // e.g., github.com/user/module
	metadata *ModuleMetadata
}

// NewGoRunModule creates a new go-run module
func NewGoRunModule(name, repo string) *GoRunModule {
	return &GoRunModule{
		Name: name,
		Repo: repo,
	}
}

// Run executes the module by spawning the binary and communicating via stdin/stdout JSON
func (m *SubprocessModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	return runSubprocess(ctx, stepCtx, step, []string{m.BinaryPath})
}

// Run executes the module by running 'go run' on the GitHub repo
func (m *GoRunModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	return runSubprocess(ctx, stepCtx, step, []string{"go", "run", m.Repo + "/cmd/main.go"})
}

// runSubprocess is the shared implementation for both binary and go-run modules
func runSubprocess(ctx context.Context, stepCtx interface{}, step interface{}, cmdArgs []string) (map[string]interface{}, error) {
	// Extract action and config from step
	type stepLike interface {
		GetType() string
		GetConfig() map[string]interface{}
	}

	var action string
	var config map[string]interface{}

	// Use type assertion to extract step fields
	if s, ok := step.(interface {
		GetType() string
		GetConfig() map[string]interface{}
	}); ok {
		action = s.GetType()
		config = s.GetConfig()
	} else {
		// Fallback: marshal/unmarshal to extract fields
		data, err := json.Marshal(step)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal step: %w", err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step: %w", err)
		}
		if t, ok := raw["type"].(string); ok {
			action = t
		}
		if c, ok := raw["config"].(map[string]interface{}); ok {
			config = c
		}
	}

	// Strip module prefix from action (e.g. "jira.create_issue" → "create_issue")
	if idx := strings.Index(action, "."); idx >= 0 {
		action = action[idx+1:]
	}

	// Build context map from stepCtx
	var contextMap map[string]interface{}
	if stepCtx != nil {
		data, _ := json.Marshal(stepCtx)
		json.Unmarshal(data, &contextMap)
	}

	req := SubprocessRequest{
		Action:  action,
		Config:  config,
		Context: contextMap,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Spawn subprocess
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = bytes.NewReader(reqBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("subprocess error: %w\nstderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("subprocess error: %w", err)
	}

	// Parse response
	var resp SubprocessResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse subprocess response: %w\nstdout: %s", err, stdout.String())
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return resp.Output, nil
}

// Metadata returns module metadata, fetching from subprocess if needed
func (m *SubprocessModule) Metadata() ModuleMetadata {
	if m.metadata != nil {
		return *m.metadata
	}
	meta := fetchMetadata(m.BinaryPath)
	m.metadata = &meta
	return meta
}

// Metadata returns module metadata, fetching from subprocess if needed
func (m *GoRunModule) Metadata() ModuleMetadata {
	if m.metadata != nil {
		return *m.metadata
	}
	meta := fetchMetadata("go", "run", m.Repo+"/cmd/main.go")
	m.metadata = &meta
	return meta
}

// fetchMetadata runs the subprocess to get metadata
func fetchMetadata(cmdArgs ...string) ModuleMetadata {
	req := SubprocessRequest{Action: "__metadata__"}
	reqBytes, _ := json.Marshal(req)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = bytes.NewReader(reqBytes)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err == nil {
		var meta ModuleMetadata
		if err := json.Unmarshal(stdout.Bytes(), &meta); err == nil {
			return meta
		}
	}

	return ModuleMetadata{
		Name:        "subprocess",
		Description: "Subprocess module",
	}
}

// DiscoverSubprocessModules scans for subprocess plugin binaries and registers them
// It looks in: .shiro/plugins/, PATH for binaries named shiro-<module>
func DiscoverSubprocessModules(registry *Registry, shiroDir string) {
	pluginsDir := filepath.Join(shiroDir, "plugins")

	// Scan .shiro/plugins/ directory
	if entries, err := os.ReadDir(pluginsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			registerSubprocessFromEntry(registry, entry, pluginsDir)
		}
	}

	// Also scan PATH for shiro-* binaries
	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range pathDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, "shiro-") && name != "shiro" {
				moduleName := strings.TrimPrefix(name, "shiro-")
				// Strip OS extension if present
				moduleName = strings.TrimSuffix(moduleName, ".exe")
				binaryPath := filepath.Join(dir, name)
				if isExecutable(entry) {
					if err := registry.Register(moduleName, NewSubprocessModule(moduleName, binaryPath)); err != nil {
						// Already registered (e.g. from plugins dir), skip
						log.Printf("Subprocess module %s already registered, skipping PATH entry", moduleName)
					} else {
						log.Printf("Registered subprocess module: %s (%s)", moduleName, binaryPath)
					}
				}
			}
		}
	}
}

func registerSubprocessFromEntry(registry *Registry, entry fs.DirEntry, dir string) {
	name := entry.Name()
	// Strip OS extension
	moduleName := strings.TrimSuffix(name, ".exe")
	// Strip shiro- prefix if present
	moduleName = strings.TrimPrefix(moduleName, "shiro-")

	if !isExecutable(entry) {
		return
	}

	binaryPath := filepath.Join(dir, name)
	if err := registry.Register(moduleName, NewSubprocessModule(moduleName, binaryPath)); err != nil {
		log.Printf("Subprocess module %s already registered, skipping plugins dir entry", moduleName)
	} else {
		log.Printf("Registered subprocess module: %s (%s)", moduleName, binaryPath)
	}
}

func isExecutable(entry fs.DirEntry) bool {
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(entry.Name(), ".exe")
	}
	info, err := entry.Info()
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// DownloadSubprocessModule downloads a pre-built plugin binary from artifact_url
func DownloadSubprocessModule(name, artifactURL, pluginsDir string, insecureTLS bool) (string, error) {
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create plugins dir: %w", err)
	}

	binaryName := fmt.Sprintf("shiro-%s", name)
	destPath := filepath.Join(pluginsDir, binaryName)

	fmt.Printf("Downloading %s plugin from %s...\n", name, artifactURL)

	// Build curl command (reuse SHIRO_INSECURE_TLS pattern)
	args := []string{"-fsSL", "-o", destPath}
	if insecureTLS {
		args = append(args, "-k")
	}
	args = append(args, artifactURL)

	cmd := exec.Command("curl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download plugin: %w", err)
	}

	// Make executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make plugin executable: %w", err)
	}

	fmt.Printf("✓ Downloaded %s to %s\n", name, destPath)
	return destPath, nil
}

// PlatformSuffix returns the OS/arch suffix for binary names (e.g. "linux-arm64")
func PlatformSuffix() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	// Map Go arch names to common binary naming conventions
	switch goarch {
	case "amd64":
		goarch = "amd64"
	case "arm64":
		goarch = "arm64"
	}
	return fmt.Sprintf("%s-%s", goos, goarch)
}
