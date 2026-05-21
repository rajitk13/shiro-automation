package cli

import (
	"strings"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/errors"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

func TestValidateAIWorkflowConfigAllowsSingleProviderDefaultModel(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Steps: []workflow.Step{
			{ID: "review", Type: "ai.generate", Config: map[string]interface{}{"prompt": "review this"}},
		},
	}
	modelConfig := map[string]map[string]interface{}{
		"openai-custom": {"type": "openai", "model": "gpt-4.1"},
	}

	if validationErrors := validateAIWorkflowConfig(wf, modelConfig); len(validationErrors) != 0 {
		t.Fatalf("validateAIWorkflowConfig() = %v, want no errors", validationErrors)
	}
}

func TestValidateAIWorkflowConfigReportsProviderModelAndPromptErrors(t *testing.T) {
	wf := &workflow.Workflow{
		Name: "test",
		Steps: []workflow.Step{
			{ID: "missing_provider", Type: "ai.generate", Config: map[string]interface{}{"provider": "missing", "prompt": "review this"}},
			{ID: "missing_model", Type: "ai.generate", Config: map[string]interface{}{"provider": "openai-custom", "prompt": "review this"}},
			{ID: "missing_prompt", Type: "ai.generate", Config: map[string]interface{}{"provider": "openai-custom", "model": "gpt-4.1"}},
		},
	}
	modelConfig := map[string]map[string]interface{}{
		"openai-custom": {"type": "openai"},
	}

	validationErrors := validateAIWorkflowConfig(wf, modelConfig)
	if len(validationErrors) != 3 {
		t.Fatalf("len(validationErrors) = %d, want 3: %v", len(validationErrors), validationErrors)
	}
	assertValidationErrorContains(t, validationErrors, "steps[missing_provider].config.provider", "provider \"missing\" not found")
	assertValidationErrorContains(t, validationErrors, "steps[missing_model].config.model", "model is required")
	assertValidationErrorContains(t, validationErrors, "steps[missing_prompt].config.prompt", "prompt is required")
}

func assertValidationErrorContains(t *testing.T, validationErrors errors.ValidationErrors, field string, message string) {
	t.Helper()
	for _, validationError := range validationErrors {
		if validationError.Field == field && strings.Contains(validationError.Message, message) {
			return
		}
	}
	t.Fatalf("validation error %s containing %q not found in %v", field, message, validationErrors)
}
