package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/errors"
	"gopkg.in/yaml.v3"
)

// Config represents the Shiro configuration
type Config struct {
	WorkflowFile string
	ConfigFile   string
	ShiroDir     string
	StateStore   string
}

// ModelConfig represents AI model configuration
type ModelConfig struct {
	Models map[string]map[string]interface{} `json:"models" yaml:"models"`
}

// LoadConfig loads configuration with auto-detection
func LoadConfig(shiroDir string) (*Config, error) {
	cfg := &Config{
		ShiroDir:   shiroDir,
		StateStore: "gitlab", // Default state store
	}

	// Auto-detect workflow file
	cfg.WorkflowFile = detectWorkflowFile(shiroDir)

	// Auto-detect config file
	cfg.ConfigFile = detectConfigFile(shiroDir)

	return cfg, nil
}

// detectWorkflowFile finds the workflow file with priority order
func detectWorkflowFile(shiroDir string) string {
	// Priority: .shiro/workflow.json > .shiro/workflow.json (with custom dir) > workflow.json
	paths := []string{
		filepath.Join(shiroDir, "workflow.json"),
		".shiro/workflow.json",
		"workflow.json",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// detectConfigFile finds the config file with priority order
func detectConfigFile(shiroDir string) string {
	// Priority: .shiro/config.yaml > .shiro/config.yaml (with custom dir) > configs/models.yaml
	paths := []string{
		filepath.Join(shiroDir, "config.yaml"),
		".shiro/config.yaml",
		"configs/models.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// LoadModelConfig loads AI model configuration from a file
func LoadModelConfig(configFile string) (map[string]map[string]interface{}, error) {
	if configFile == "" {
		return make(map[string]map[string]interface{}), nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.NewConfigError(configFile, "failed to read config file", err)
	}

	var config ModelConfig

	// Detect file format by extension
	ext := strings.ToLower(filepath.Ext(configFile))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, errors.NewConfigError(configFile, "failed to parse YAML config file", err)
		}
	} else {
		return nil, errors.NewConfigError(configFile, fmt.Sprintf("unsupported config file format: %s", ext), nil)
	}

	// Resolve environment variables in config
	resolveEnvVars(config.Models)

	return config.Models, nil
}

// resolveEnvVars resolves {{env.VARIABLE}} templates in config values
func resolveEnvVars(config map[string]map[string]interface{}) {
	for _, modelDef := range config {
		for key, value := range modelDef {
			if strValue, ok := value.(string); ok {
				resolved := resolveEnvVarString(strValue)
				if resolved != strValue {
					// Log that environment variable was resolved
					fmt.Printf("[Config] Resolved env var in config: %s\n", key)
				}
				modelDef[key] = resolved
			}
		}
	}
}

// resolveEnvVarString resolves a single {{env.VARIABLE}} template
func resolveEnvVarString(input string) string {
	if strings.HasPrefix(input, "{{env.") && strings.HasSuffix(input, "}}") {
		envVar := strings.TrimPrefix(input, "{{env.")
		envVar = strings.TrimSuffix(envVar, "}}")
		envValue := os.Getenv(envVar)
		if envValue != "" {
			fmt.Printf("[Config] Resolved %s = %s\n", envVar, "***")
			return envValue
		}
		fmt.Printf("[Config] Environment variable %s not found\n", envVar)
	}
	return input
}

// GetRegistryPath returns the module registry path
func GetRegistryPath(shiroDir string) string {
	registryPath := filepath.Join(shiroDir, "modules", "registry.yaml")
	if _, err := os.Stat(registryPath); err != nil {
		return "modules/registry.yaml" // Fallback to default
	}
	return registryPath
}
