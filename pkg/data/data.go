package data

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// StateStore defines the interface for state storage
type StateStore interface {
	Save(ctx context.Context, key string, state interface{}) error
	Load(ctx context.Context, key string, target interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// DataModule implements data storage operations
type DataModule struct {
	store StateStore
}

// NewDataModule creates a new data module with the given state store
func NewDataModule(store StateStore) *DataModule {
	return &DataModule{store: store}
}

// Run executes data store or load operations
func (m *DataModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	// Determine operation type
	operation := wfStep.Type

	switch operation {
	case "data.store":
		return m.storeData(ctx, stepCtx, wfStep)
	case "data.load":
		return m.loadData(ctx, stepCtx, wfStep)
	case "data.delete":
		return m.deleteData(ctx, stepCtx, wfStep)
	case "data.exists":
		return m.existsData(ctx, stepCtx, wfStep)
	default:
		return nil, fmt.Errorf("unknown data operation: %s", operation)
	}
}

func (m *DataModule) storeData(ctx context.Context, stepCtx interface{}, wfStep workflow.Step) (map[string]interface{}, error) {
	// Extract configuration
	key, ok := wfStep.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required for data.store")
	}

	value := wfStep.Config["value"]
	if value == nil {
		return nil, fmt.Errorf("value is required for data.store")
	}

	ttlStr, _ := wfStep.Config["ttl"].(string)
	namespace, _ := wfStep.Config["namespace"].(string)

	// Resolve variables if we have execution context
	if execCtx, ok := stepCtx.(*workflow.ExecutionContext); ok {
		resolver := workflow.NewVariableResolver(execCtx)

		resolvedKey, err := resolver.Resolve(key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve key: %w", err)
		}
		if str, ok := resolvedKey.(string); ok {
			key = str
		}

		resolvedValue, err := resolver.Resolve(value)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve value: %w", err)
		}
		value = resolvedValue
	}

	// Build full key with namespace and pipeline ID
	fullKey := m.buildKey(key, namespace)

	// Add TTL metadata if specified
	var dataToStore interface{} = value
	if ttlStr != "" {
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TTL format: %w", err)
		}
		dataToStore = map[string]interface{}{
			"_data":       value,
			"_ttl":        ttlStr,
			"_stored_at":  time.Now().UTC().Format(time.RFC3339),
			"_expires_at": time.Now().Add(ttl).UTC().Format(time.RFC3339),
		}
	}

	// Store the data
	if err := m.store.Save(ctx, fullKey, dataToStore); err != nil {
		return nil, fmt.Errorf("failed to store data: %w", err)
	}

	return map[string]interface{}{
		"success":   true,
		"key":       key,
		"namespace": namespace,
		"stored":    true,
	}, nil
}

func (m *DataModule) loadData(ctx context.Context, stepCtx interface{}, wfStep workflow.Step) (map[string]interface{}, error) {
	// Extract configuration
	key, ok := wfStep.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required for data.load")
	}

	namespace, _ := wfStep.Config["namespace"].(string)
	fallback := wfStep.Config["fallback"]

	// Resolve variables if we have execution context
	if execCtx, ok := stepCtx.(*workflow.ExecutionContext); ok {
		resolver := workflow.NewVariableResolver(execCtx)

		resolvedKey, err := resolver.Resolve(key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve key: %w", err)
		}
		if str, ok := resolvedKey.(string); ok {
			key = str
		}

		if fallback != nil {
			resolvedFallback, err := resolver.Resolve(fallback)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve fallback: %w", err)
			}
			fallback = resolvedFallback
		}
	}

	// Build full key
	fullKey := m.buildKey(key, namespace)

	// Load the data
	var storedData interface{}
	if err := m.store.Load(ctx, fullKey, &storedData); err != nil {
		// Return fallback if provided
		if fallback != nil {
			return map[string]interface{}{
				"success":   true,
				"key":       key,
				"namespace": namespace,
				"value":     fallback,
				"fallback":  true,
				"exists":    false,
			}, nil
		}
		return nil, fmt.Errorf("data not found for key '%s': %w", key, err)
	}

	// Check for TTL metadata
	var value interface{} = storedData
	if dataMap, ok := storedData.(map[string]interface{}); ok {
		if _, hasTTL := dataMap["_ttl"]; hasTTL {
			// This is TTL-wrapped data
			if expiresAt, ok := dataMap["_expires_at"].(string); ok {
				expiry, err := time.Parse(time.RFC3339, expiresAt)
				if err == nil && time.Now().After(expiry) {
					// Data has expired
					if fallback != nil {
						return map[string]interface{}{
							"success":   true,
							"key":       key,
							"namespace": namespace,
							"value":     fallback,
							"fallback":  true,
							"exists":    false,
							"expired":   true,
						}, nil
					}
					return nil, fmt.Errorf("data for key '%s' has expired", key)
				}
			}
			value = dataMap["_data"]
		}
	}

	return map[string]interface{}{
		"success":   true,
		"key":       key,
		"namespace": namespace,
		"value":     value,
		"exists":    true,
		"fallback":  false,
	}, nil
}

func (m *DataModule) deleteData(ctx context.Context, stepCtx interface{}, wfStep workflow.Step) (map[string]interface{}, error) {
	key, ok := wfStep.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required for data.delete")
	}

	namespace, _ := wfStep.Config["namespace"].(string)

	// Resolve variables if we have execution context
	if execCtx, ok := stepCtx.(*workflow.ExecutionContext); ok {
		resolver := workflow.NewVariableResolver(execCtx)

		resolvedKey, err := resolver.Resolve(key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve key: %w", err)
		}
		if str, ok := resolvedKey.(string); ok {
			key = str
		}
	}

	fullKey := m.buildKey(key, namespace)

	if err := m.store.Delete(ctx, fullKey); err != nil {
		return map[string]interface{}{
			"success":   false,
			"key":       key,
			"namespace": namespace,
			"deleted":   false,
			"error":     err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"success":   true,
		"key":       key,
		"namespace": namespace,
		"deleted":   true,
	}, nil
}

func (m *DataModule) existsData(ctx context.Context, stepCtx interface{}, wfStep workflow.Step) (map[string]interface{}, error) {
	key, ok := wfStep.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required for data.exists")
	}

	namespace, _ := wfStep.Config["namespace"].(string)

	// Resolve variables if we have execution context
	if execCtx, ok := stepCtx.(*workflow.ExecutionContext); ok {
		resolver := workflow.NewVariableResolver(execCtx)

		resolvedKey, err := resolver.Resolve(key)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve key: %w", err)
		}
		if str, ok := resolvedKey.(string); ok {
			key = str
		}
	}

	fullKey := m.buildKey(key, namespace)

	exists, err := m.store.Exists(ctx, fullKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check existence: %w", err)
	}

	return map[string]interface{}{
		"success":   true,
		"key":       key,
		"namespace": namespace,
		"exists":    exists,
	}, nil
}

// buildKey constructs the full storage key
func (m *DataModule) buildKey(key, namespace string) string {
	// Always include pipeline ID for isolation
	pipelineID := os.Getenv("CI_PIPELINE_ID")
	if pipelineID == "" {
		pipelineID = "default"
	}

	if namespace != "" {
		return fmt.Sprintf("data-%s-%s-%s", namespace, pipelineID, key)
	}
	return fmt.Sprintf("data-%s-%s", pipelineID, key)
}

// SetData stores data directly (for CLI usage)
func (m *DataModule) SetData(ctx context.Context, key string, value interface{}, namespace string, ttl string) error {
	fullKey := m.buildKey(key, namespace)

	var dataToStore interface{} = value
	if ttl != "" {
		ttlDuration, err := time.ParseDuration(ttl)
		if err != nil {
			return fmt.Errorf("invalid TTL format: %w", err)
		}
		dataToStore = map[string]interface{}{
			"_data":       value,
			"_ttl":        ttl,
			"_stored_at":  time.Now().UTC().Format(time.RFC3339),
			"_expires_at": time.Now().Add(ttlDuration).UTC().Format(time.RFC3339),
		}
	}

	return m.store.Save(ctx, fullKey, dataToStore)
}

// GetData retrieves data directly (for CLI usage)
func (m *DataModule) GetData(ctx context.Context, key string, namespace string) (interface{}, bool, error) {
	fullKey := m.buildKey(key, namespace)

	var storedData interface{}
	if err := m.store.Load(ctx, fullKey, &storedData); err != nil {
		return nil, false, err
	}

	// Check for TTL metadata
	var value interface{} = storedData
	if dataMap, ok := storedData.(map[string]interface{}); ok {
		if _, hasTTL := dataMap["_ttl"]; hasTTL {
			if expiresAt, ok := dataMap["_expires_at"].(string); ok {
				expiry, err := time.Parse(time.RFC3339, expiresAt)
				if err == nil && time.Now().After(expiry) {
					return nil, false, fmt.Errorf("data has expired")
				}
			}
			value = dataMap["_data"]
		}
	}

	return value, true, nil
}

// DeleteData deletes data directly (for CLI usage)
func (m *DataModule) DeleteData(ctx context.Context, key string, namespace string) error {
	fullKey := m.buildKey(key, namespace)
	return m.store.Delete(ctx, fullKey)
}

// ListData lists all data keys with optional prefix (for CLI usage)
func (m *DataModule) ListData(ctx context.Context, prefix string) ([]string, error) {
	// This is a simplified implementation - in a real system you'd want to
	// scan the state store for matching keys
	return nil, fmt.Errorf("list operation not yet implemented")
}

// Metadata returns module metadata
func (m *DataModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "data.store",
		Description: "Store and retrieve persistent data with custom keys and optional TTL",
		InputSchema: map[string]modules.SchemaField{
			"key": {
				Type:        "string",
				Description: "Unique key for storing/retrieving data",
				Required:    true,
			},
			"value": {
				Type:        "any",
				Description: "Value to store (required for store operations)",
				Required:    false,
			},
			"namespace": {
				Type:        "string",
				Description: "Optional namespace for key isolation",
				Required:    false,
			},
			"ttl": {
				Type:        "string",
				Description: "Time-to-live duration (e.g., '24h', '1h30m')",
				Required:    false,
			},
			"fallback": {
				Type:        "any",
				Description: "Default value if key not found (for load operations)",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"success": {
				Type:        "boolean",
				Description: "Whether operation succeeded",
				Required:    true,
			},
			"key": {
				Type:        "string",
				Description: "Key that was accessed",
				Required:    true,
			},
			"namespace": {
				Type:        "string",
				Description: "Namespace used",
				Required:    false,
			},
			"value": {
				Type:        "any",
				Description: "Retrieved value (for load operations)",
				Required:    false,
			},
			"exists": {
				Type:        "boolean",
				Description: "Whether key exists (for load/exists operations)",
				Required:    false,
			},
			"stored": {
				Type:        "boolean",
				Description: "Whether data was stored (for store operations)",
				Required:    false,
			},
			"deleted": {
				Type:        "boolean",
				Description: "Whether data was deleted (for delete operations)",
				Required:    false,
			},
			"fallback": {
				Type:        "boolean",
				Description: "Whether fallback value was used",
				Required:    false,
			},
			"expired": {
				Type:        "boolean",
				Description: "Whether data has expired",
				Required:    false,
			},
		},
	}
}

// StoreFactory creates DataModules with state stores
type StoreFactory struct {
	stores map[string]StateStore
}

// NewStoreFactory creates a new store factory
func NewStoreFactory() *StoreFactory {
	return &StoreFactory{
		stores: make(map[string]StateStore),
	}
}

// RegisterStore registers a state store
func (f *StoreFactory) RegisterStore(name string, store StateStore) {
	f.stores[name] = store
}

// GetStore retrieves a registered state store
func (f *StoreFactory) GetStore(name string) (StateStore, bool) {
	store, ok := f.stores[name]
	return store, ok
}
