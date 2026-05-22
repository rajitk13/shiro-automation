package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/config"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/pkg/data"
)

// SetCommand handles the data set command
func SetCommand(args []string) {
	flagSet := flag.NewFlagSet("set", flag.ExitOnError)
	stateStoreType := flagSet.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	namespace := flagSet.String("namespace", "", "Namespace for the key")
	ttl := flagSet.String("ttl", "", "Time-to-live (e.g., '24h', '1h30m')")
	showHelp := flagSet.Bool("help", false, "Show help information")

	flagSet.Parse(args)

	if *showHelp {
		printSetHelp()
		os.Exit(0)
	}

	// Get positional arguments
	positionalArgs := flagSet.Args()
	if len(positionalArgs) < 2 {
		fmt.Println("Error: key and value are required")
		printSetHelp()
		os.Exit(1)
	}

	key := positionalArgs[0]
	value := positionalArgs[1]

	// Load configuration
	cfg, err := config.LoadConfig(*shiroDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *stateStoreType != "" {
		cfg.StateStore = *stateStoreType
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	stateStore, err := stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// Create data module
	dataModule := data.NewDataModule(stateStore)

	// Store the data
	ctx := context.Background()
	if err := dataModule.SetData(ctx, key, value, *namespace, *ttl); err != nil {
		log.Fatalf("Failed to set data: %v", err)
	}

	fmt.Printf("Successfully set '%s' = '%s'\n", key, value)
	if *namespace != "" {
		fmt.Printf("Namespace: %s\n", *namespace)
	}
	if *ttl != "" {
		fmt.Printf("TTL: %s\n", *ttl)
	}
}

// GetCommand handles the data get command
func GetCommand(args []string) {
	flagSet := flag.NewFlagSet("get", flag.ExitOnError)
	stateStoreType := flagSet.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	namespace := flagSet.String("namespace", "", "Namespace for the key")
	defaultValue := flagSet.String("default", "", "Default value if key not found")
	showHelp := flagSet.Bool("help", false, "Show help information")

	flagSet.Parse(args)

	if *showHelp {
		printGetHelp()
		os.Exit(0)
	}

	// Get positional arguments
	positionalArgs := flagSet.Args()
	if len(positionalArgs) < 1 {
		fmt.Println("Error: key is required")
		printGetHelp()
		os.Exit(1)
	}

	key := positionalArgs[0]

	// Load configuration
	cfg, err := config.LoadConfig(*shiroDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *stateStoreType != "" {
		cfg.StateStore = *stateStoreType
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	stateStore, err := stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// Create data module
	dataModule := data.NewDataModule(stateStore)

	// Get the data
	ctx := context.Background()
	value, exists, err := dataModule.GetData(ctx, key, *namespace)
	if err != nil {
		if *defaultValue != "" {
			fmt.Println(*defaultValue)
			os.Exit(0)
		}
		log.Fatalf("Failed to get data: %v", err)
	}

	if !exists {
		if *defaultValue != "" {
			fmt.Println(*defaultValue)
			os.Exit(0)
		}
		log.Fatalf("Key '%s' not found", key)
	}

	// Output the value
	if str, ok := value.(string); ok {
		fmt.Println(str)
	} else {
		// For non-string values, print as JSON
		fmt.Printf("%v\n", value)
	}
}

// DeleteCommand handles the data delete command
func DeleteCommand(args []string) {
	flagSet := flag.NewFlagSet("delete", flag.ExitOnError)
	stateStoreType := flagSet.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	namespace := flagSet.String("namespace", "", "Namespace for the key")
	showHelp := flagSet.Bool("help", false, "Show help information")

	flagSet.Parse(args)

	if *showHelp {
		printDeleteHelp()
		os.Exit(0)
	}

	// Get positional arguments
	positionalArgs := flagSet.Args()
	if len(positionalArgs) < 1 {
		fmt.Println("Error: key is required")
		printDeleteHelp()
		os.Exit(1)
	}

	key := positionalArgs[0]

	// Load configuration
	cfg, err := config.LoadConfig(*shiroDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *stateStoreType != "" {
		cfg.StateStore = *stateStoreType
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	stateStore, err := stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// Create data module
	dataModule := data.NewDataModule(stateStore)

	// Delete the data
	ctx := context.Background()
	if err := dataModule.DeleteData(ctx, key, *namespace); err != nil {
		log.Fatalf("Failed to delete data: %v", err)
	}

	fmt.Printf("Successfully deleted '%s'\n", key)
	if *namespace != "" {
		fmt.Printf("Namespace: %s\n", *namespace)
	}
}

// ListCommand handles the data list command
func ListCommand(args []string) {
	flagSet := flag.NewFlagSet("list", flag.ExitOnError)
	stateStoreType := flagSet.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	prefix := flagSet.String("prefix", "", "Filter keys by prefix")
	showHelp := flagSet.Bool("help", false, "Show help information")

	flagSet.Parse(args)

	if *showHelp {
		printListHelp()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*shiroDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *stateStoreType != "" {
		cfg.StateStore = *stateStoreType
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	stateStore, err := stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// List data
	ctx := context.Background()
	dataModule := data.NewDataModule(stateStore)
	keys, err := dataModule.ListData(ctx, *prefix)
	if err != nil {
		log.Fatalf("Failed to list data: %v", err)
	}

	if len(keys) == 0 {
		fmt.Println("No data keys found")
		return
	}

	fmt.Println("Data keys:")
	for _, key := range keys {
		fmt.Printf("  - %s\n", key)
	}
}

func printSetHelp() {
	fmt.Print(`Usage: shiro set <key> <value> [options]

Store a value in the data store.

Arguments:
  key          The key to store the value under
  value        The value to store

Options:
  -state-store <type>    State store type (memory, filesystem, gitlab)
  -shiro-dir <path>     Path to .shiro directory
  -namespace <name>      Namespace for the key
  -ttl <duration>        Time-to-live (e.g., '24h', '1h30m')
  -help                  Show this help message

Examples:
  shiro set build_id "12345"
  shiro set api_key "secret" -namespace "production"
  shiro set temp_data "value" -ttl "1h"
`)
}

func printGetHelp() {
	fmt.Print(`Usage: shiro get <key> [options]

Retrieve a value from the data store.

Arguments:
  key          The key to retrieve

Options:
  -state-store <type>    State store type (memory, filesystem, gitlab)
  -shiro-dir <path>     Path to .shiro directory
  -namespace <name>      Namespace for the key
  -default <value>       Default value if key not found
  -help                  Show this help message

Examples:
  shiro get build_id
  shiro get api_key -namespace "production"
  shiro get missing_key -default "not found"
`)
}

func printDeleteHelp() {
	fmt.Print(`Usage: shiro delete <key> [options]

Delete a value from the data store.

Arguments:
  key          The key to delete

Options:
  -state-store <type>    State store type (memory, filesystem, gitlab)
  -shiro-dir <path>     Path to .shiro directory
  -namespace <name>      Namespace for the key
  -help                  Show this help message

Examples:
  shiro delete build_id
  shiro delete temp_key -namespace "cache"
`)
}

func printListHelp() {
	fmt.Print(`Usage: shiro list [options]

List all data keys in the store.

Options:
  -state-store <type>    State store type (memory, filesystem, gitlab)
  -shiro-dir <path>     Path to .shiro directory
  -prefix <string>       Filter keys by prefix
  -help                  Show this help message

Examples:
  shiro list
  shiro list -prefix "build_"
`)
}

// DataHelp prints help for data commands
func DataHelp() {
	fmt.Print(`Data Management Commands:

  set <key> <value>     Store a value
  get <key>             Retrieve a value
  delete <key>           Delete a value
  list                   List all keys

Use "shiro <command> -help" for more information about a command.
`)
}
