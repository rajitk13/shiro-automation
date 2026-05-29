package workflow

import (
	"testing"
)

func TestResolveEnvVarString(t *testing.T) {
	t.Setenv("SHIRO_TEST_HOST", "example.com")
	t.Setenv("SHIRO_TEST_TOKEN", "secret")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "full string match", input: "{{env.SHIRO_TEST_TOKEN}}", want: "secret"},
		{name: "embedded single", input: "https://{{env.SHIRO_TEST_HOST}}/api", want: "https://example.com/api"},
		{name: "multiple vars", input: "{{env.SHIRO_TEST_HOST}}:{{env.SHIRO_TEST_TOKEN}}", want: "example.com:secret"},
		{name: "unset left untouched", input: "Bearer {{env.SHIRO_TEST_MISSING}}", want: "Bearer {{env.SHIRO_TEST_MISSING}}"},
		{name: "no template", input: "plain-value", want: "plain-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveEnvVarString(tt.input); got != tt.want {
				t.Errorf("resolveEnvVarString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWorkflowValidation(t *testing.T) {
	tests := []struct {
		name    string
		wf      *Workflow
		wantErr bool
	}{
		{
			name: "valid workflow",
			wf: &Workflow{
				Name: "test-workflow",
				Steps: []Step{
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			wf: &Workflow{
				Steps: []Step{
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "no steps",
			wf: &Workflow{
				Name: "test-workflow",
			},
			wantErr: true,
		},
		{
			name: "duplicate step IDs",
			wf: &Workflow{
				Name: "test-workflow",
				Steps: []Step{
					{ID: "step1", Type: "test"},
					{ID: "step1", Type: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "circular dependency",
			wf: &Workflow{
				Name: "test-workflow",
				Steps: []Step{
					{ID: "step1", Type: "test", DependsOn: []string{"step2"}},
					{ID: "step2", Type: "test", DependsOn: []string{"step1"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wf.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadWorkflow(t *testing.T) {
	jsonData := `{
		"name": "test-workflow",
		"steps": [
			{
				"id": "step1",
				"type": "test"
			}
		]
	}`

	wf, err := LoadWorkflow([]byte(jsonData))
	if err != nil {
		t.Fatalf("LoadWorkflow() error = %v", err)
	}

	if wf.Name != "test-workflow" {
		t.Errorf("Expected name 'test-workflow', got '%s'", wf.Name)
	}

	if len(wf.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(wf.Steps))
	}
}

func TestGetStepByID(t *testing.T) {
	wf := &Workflow{
		Name: "test-workflow",
		Steps: []Step{
			{ID: "step1", Type: "test"},
			{ID: "step2", Type: "test"},
		},
	}

	step := wf.GetStepByID("step1")
	if step == nil {
		t.Fatal("GetStepByID() returned nil for existing step")
	}

	if step.ID != "step1" {
		t.Errorf("Expected step ID 'step1', got '%s'", step.ID)
	}

	step = wf.GetStepByID("nonexistent")
	if step != nil {
		t.Error("GetStepByID() returned non-nil for non-existent step")
	}
}
