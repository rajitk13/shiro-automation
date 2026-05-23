package cli

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"gopkg.in/yaml.v3"
)

// ModuleCommand handles module-related commands
func ModuleCommand(args []string) {
	if len(args) < 1 {
		printModuleHelp()
		os.Exit(1)
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list":
		listModules()
	case "add":
		addModule(subArgs)
	case "remove":
		removeModule(subArgs)
	case "search":
		searchModules(subArgs)
	case "install":
		installModule(subArgs)
	case "info":
		moduleInfo(subArgs)
	case "docs":
		moduleDocs(subArgs)
	case "help":
		printModuleHelp()
	default:
		fmt.Printf("Unknown module command: %s\n", subcommand)
		printModuleHelp()
		os.Exit(1)
	}
}

// listModules lists all available modules
func listModules() {
	registryPath := ".shiro/modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	modules := discoverer.ListModules()
	fmt.Println("Available modules:")
	for _, name := range modules {
		config, _ := discoverer.GetModuleConfig(name)
		fmt.Printf("  - %s (%s): %s\n", name, config.Type, config.Description)
	}
}

// addModule adds a module to the registry
func addModule(args []string) {
	// Skip optional "module" keyword (allows `shiro add module <repo>`)
	if len(args) > 0 && args[0] == "module" {
		args = args[1:]
	}

	// Check if it's a GitHub URL or a module name
	var moduleName, gitURL string
	var isGitURL bool

	if len(args) > 0 {
		input := args[0]
		if strings.Contains(input, "github.com") {
			isGitURL = true
			gitURL = input
			// Extract module name from URL
			parts := strings.Split(input, "/")
			if len(parts) > 0 {
				moduleName = parts[len(parts)-1]
			}
		} else {
			moduleName = input
			// Auto-discover from official repo
			gitURL = fmt.Sprintf("github.com/rkuthiala/%s-module", moduleName)
			fmt.Printf("Auto-discovering module '%s' from official repository...\n", moduleName)
		}
	} else {
		log.Fatal("Module name or GitHub URL is required")
	}

	// For auto-discovery, search GitHub first
	if !isGitURL {
		token := os.Getenv("GITHUB_TOKEN")
		githubClient := modules.NewGitHubClient(token)
		results, err := githubClient.SearchModules(moduleName)
		if err != nil {
			fmt.Printf("Warning: Failed to search GitHub: %v\n", err)
		} else if len(results) > 0 {
			// Use the first result
			gitURL = results[0].FullName
			fmt.Printf("Found module: %s\n", results[0].Name)
			fmt.Printf("Repository: %s\n", gitURL)
			fmt.Printf("Stars: %d\n", results[0].Stargazers)
			fmt.Printf("Description: %s\n", results[0].Description)
		} else {
			fmt.Printf("Module '%s' not found in official repository.\n", moduleName)
			fmt.Printf("Please provide a GitHub URL: shiro add module github.com/user/%s-module\n", moduleName)
			os.Exit(1)
		}
	}

	// Parse GitHub repository
	repoPath, err := modules.ParseGitHubRepo(gitURL)
	if err != nil {
		log.Fatalf("Invalid GitHub repository format: %v", err)
	}

	// Get module metadata from GitHub
	token := os.Getenv("GITHUB_TOKEN")
	githubClient := modules.NewGitHubClient(token)
	metadata, err := githubClient.GetModuleMetadata(repoPath)
	if err != nil {
		log.Fatalf("Failed to get module metadata: %v", err)
	}

	// Determine module name
	if moduleName == "" {
		parts := strings.Split(repoPath, "/")
		moduleName = parts[len(parts)-1]
	}

	// Create module config
	registryPath := ".shiro/modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)
	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	// Fetch module.yaml for additional metadata (type, package, factory, etc.)
	moduleYAML, err := fetchModuleYAML(githubClient, repoPath)
	if err != nil {
		fmt.Printf("Warning: Could not fetch module.yaml: %v\n", err)
		fmt.Println("Falling back to HTTP module type.")
		moduleYAML = nil
	}

	var config modules.ModuleConfig

	if moduleYAML != nil && moduleYAML.Type == "subprocess" {
		// Subprocess plugin module - try binary download, fall back to go run
		insecureTLS := os.Getenv("SHIRO_INSECURE_TLS") == "1" || os.Getenv("SHIRO_INSECURE_TLS") == "true"
		pluginsDir := ".shiro/plugins"

		// Try to download binary if artifact_url is specified
		artifactURL := moduleYAML.ArtifactURL
		if artifactURL == "" && moduleYAML.Binary != "" {
			// Build default artifact URL from repo releases
			artifactURL = fmt.Sprintf("https://github.com/%s/releases/latest/download/%s-%s",
				repoPath, moduleYAML.Binary, modules.PlatformSuffix())
		}

		var useGoRun bool
		if artifactURL != "" {
			_, err := modules.DownloadSubprocessModule(moduleName, artifactURL, pluginsDir, insecureTLS)
			if err != nil {
				fmt.Printf("Warning: Failed to download binary: %v\n", err)
				fmt.Println("Falling back to 'go run' mode (requires Go toolchain)")
				useGoRun = true
			}
		} else {
			fmt.Println("No artifact_url specified, using 'go run' mode (requires Go toolchain)")
			useGoRun = true
		}

		config = modules.ModuleConfig{
			Name:        moduleName,
			Type:        "subprocess",
			Description: moduleYAML.Description,
			Version:     moduleYAML.Version,
			Source:      metadata.Repository,
		}
		if useGoRun {
			config.GoRunRepo = repoPath
		}
		if err := discoverer.AddModule(moduleName, config); err != nil {
			log.Fatalf("Failed to add module to registry: %v", err)
		}

		if useGoRun {
			fmt.Printf("✓ Module '%s' added successfully (go run mode)!\n", moduleName)
			fmt.Printf("Repo: %s\n", repoPath)
			fmt.Println("Run 'shiro run' to use the new module — requires Go toolchain.")
		} else {
			fmt.Printf("✓ Module '%s' added successfully!\n", moduleName)
			fmt.Println("Run 'shiro run' to use the new module — no rebuild needed.")
		}

		if len(moduleYAML.Credentials) > 0 {
			fmt.Println("\nRequired credentials:")
			for _, cred := range moduleYAML.Credentials {
				if cred.Required {
					fmt.Printf("  - %s: %s\n", cred.Name, cred.Description)
				}
			}
		}
	} else {
		// HTTP-based module (original behavior)
		config = modules.ModuleConfig{
			Name:        metadata.Name,
			Type:        "http",
			Description: metadata.Description,
			Version:     "1.0.0",
			Source:      metadata.Repository,
			Docs:        fmt.Sprintf("%s/blob/main/README.md", metadata.Repository),
		}

		if err := discoverer.AddModule(moduleName, config); err != nil {
			log.Fatalf("Failed to add module: %v", err)
		}

		fmt.Printf("Module '%s' added successfully!\n", moduleName)
		fmt.Printf("Source: %s\n", metadata.Repository)
		fmt.Printf("To use this module, configure its endpoints in .shiro/modules/registry.yaml\n")
	}
}

// removeModule removes a module from the registry
func removeModule(args []string) {
	if len(args) < 1 {
		log.Fatal("Module name is required")
	}

	name := args[0]
	registryPath := ".shiro/modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	if err := discoverer.RemoveModule(name); err != nil {
		log.Fatalf("Failed to remove module: %v", err)
	}

	fmt.Printf("Module %s removed successfully\n", name)
}

// searchModules searches for modules on GitHub
func searchModules(args []string) {
	if len(args) < 1 {
		log.Fatal("Search query is required")
	}

	query := args[0]
	token := os.Getenv("GITHUB_TOKEN") // Optional GitHub token for higher rate limits

	githubClient := modules.NewGitHubClient(token)
	results, err := githubClient.SearchModules(query)
	if err != nil {
		log.Fatalf("Failed to search modules: %v", err)
	}

	fmt.Printf("Found %d modules for '%s':\n", len(results), query)
	for _, result := range results {
		fmt.Printf("  - %s (%s)\n", result.Name, result.FullName)
		fmt.Printf("    Description: %s\n", result.Description)
		fmt.Printf("    Stars: %d, Language: %s\n", result.Stargazers, result.Language)
		fmt.Printf("    Repository: %s\n\n", result.HTMLURL)
	}
}

// installModule installs a module from a GitHub repository
func installModule(args []string) {
	if len(args) < 1 {
		log.Fatal("GitHub repository is required")
	}

	repo := args[0]
	token := os.Getenv("GITHUB_TOKEN")

	// Parse GitHub repository format
	repoPath, err := modules.ParseGitHubRepo(repo)
	if err != nil {
		log.Fatalf("Invalid GitHub repository format: %v", err)
	}

	githubClient := modules.NewGitHubClient(token)
	metadata, err := githubClient.GetModuleMetadata(repoPath)
	if err != nil {
		log.Fatalf("Failed to get module metadata: %v", err)
	}

	// Create module configuration
	moduleConfig := modules.ModuleConfig{
		Name:        metadata.Name,
		Type:        "http",
		Version:     "1.0.0",
		Description: metadata.Description,
		Source:      metadata.Repository,
		Docs:        fmt.Sprintf("%s/blob/main/README.md", metadata.Repository),
	}

	// Add to registry
	registryPath := ".shiro/modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	if err := discoverer.AddModule(metadata.Name, moduleConfig); err != nil {
		log.Fatalf("Failed to add module: %v", err)
	}

	fmt.Printf("Module %s installed successfully\n", metadata.Name)
	fmt.Printf("Source: %s\n", metadata.Repository)
	fmt.Printf("Please configure the module endpoints in .shiro/modules/registry.yaml\n")
}

// moduleInfo displays information about a module
func moduleInfo(args []string) {
	if len(args) < 1 {
		log.Fatal("Module name is required")
	}

	name := args[0]
	registryPath := ".shiro/modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	config, err := discoverer.GetModuleConfig(name)
	if err != nil {
		log.Fatalf("Failed to get module info: %v", err)
	}

	fmt.Printf("Module: %s\n", config.Name)
	fmt.Printf("Type: %s\n", config.Type)
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Description: %s\n", config.Description)

	if config.Source != "" {
		fmt.Printf("Source: %s\n", config.Source)
	}

	if config.Docs != "" {
		fmt.Printf("Documentation: %s\n", config.Docs)
	}

	if len(config.Endpoints) > 0 {
		fmt.Printf("Endpoints: %v\n", config.Endpoints)
	} else if config.Endpoint != "" {
		fmt.Printf("Endpoint: %s\n", config.Endpoint)
	}
}

// moduleDocs opens module documentation
func moduleDocs(args []string) {
	if len(args) < 1 {
		log.Fatal("Module name is required")
	}

	name := args[0]
	registryPath := ".shiro/modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	config, err := discoverer.GetModuleConfig(name)
	if err != nil {
		log.Fatalf("Failed to get module info: %v", err)
	}

	if config.Docs != "" {
		fmt.Printf("Opening documentation: %s\n", config.Docs)
		// In a real implementation, this would open the URL in a browser
		// For now, just print the URL
	} else if config.Source != "" {
		docsURL := fmt.Sprintf("%s/blob/main/README.md", config.Source)
		fmt.Printf("Documentation: %s\n", docsURL)
	} else {
		fmt.Printf("No documentation available for module %s\n", name)
	}
}

// printModuleHelp prints help for module commands
func printModuleHelp() {
	fmt.Println("Usage: shiro module <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list              List all available modules")
	fmt.Println("  add <module>      Add a module (auto-discovers from GitHub)")
	fmt.Println("  remove <module>   Remove a module")
	fmt.Println("  search <query>    Search for modules on GitHub")
	fmt.Println("  install <repo>    Install a module from a GitHub repository")
	fmt.Println("  info <module>     Display information about a module")
	fmt.Println("  docs <module>     Open module documentation")
	fmt.Println("  help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shiro module list")
	fmt.Println("  shiro module add jira")
	fmt.Println("  shiro module add github.com/user/custom-module")
	fmt.Println("  shiro module search slack")
	fmt.Println("  shiro module info jira")
}

// ModuleYAML represents the module.yaml file structure
type ModuleYAML struct {
	Name        string             `yaml:"name"`
	Type        string             `yaml:"type"`
	Package     string             `yaml:"package"`
	Factory     string             `yaml:"factory"`
	Description string             `yaml:"description"`
	Version     string             `yaml:"version"`
	Binary      string             `yaml:"binary"`
	ArtifactURL string             `yaml:"artifact_url"`
	Credentials []ModuleCredential `yaml:"credentials"`
}

// ModuleCredential represents a required credential
type ModuleCredential struct {
	Name        string `yaml:"name"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
	Secret      bool   `yaml:"secret"`
}

// getHTTPClient returns an HTTP client, optionally with TLS verification disabled
func getHTTPClient() *http.Client {
	if os.Getenv("SHIRO_INSECURE_TLS") == "1" || os.Getenv("SHIRO_INSECURE_TLS") == "true" {
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return &http.Client{}
}

// fetchModuleYAML fetches and parses module.yaml from a GitHub repository
func fetchModuleYAML(client *modules.GitHubClient, repo string) (*ModuleYAML, error) {
	httpClient := getHTTPClient()

	// Fetch module.yaml from GitHub
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/module.yaml", repo)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch module.yaml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try 'master' branch if 'main' fails
		url = fmt.Sprintf("https://raw.githubusercontent.com/%s/master/module.yaml", repo)
		resp, err = httpClient.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch module.yaml from master: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("module.yaml not found in repository")
		}
	}

	var moduleYAML ModuleYAML
	if err := yaml.NewDecoder(resp.Body).Decode(&moduleYAML); err != nil {
		return nil, fmt.Errorf("failed to parse module.yaml: %w", err)
	}

	return &moduleYAML, nil
}
