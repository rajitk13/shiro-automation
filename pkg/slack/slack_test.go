package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

func TestSlackNotifyIncludesGitLabReviewButton(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	module := NewSlackModule(false)
	_, err := module.Run(context.Background(), nil, workflow.Step{
		ID:   "approval",
		Type: "slack.notify",
		Config: map[string]interface{}{
			"webhook_url":         server.URL,
			"message":             "Review decision",
			"gitlab_pipeline_url": "https://gitlab.example.com/project/-/pipelines/1",
			"button_text":         "Review in GitLab",
			"username":            "Shiro",
			"icon_emoji":          ":robot_face:",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if payload["username"] != "Shiro" {
		t.Fatalf("username = %v, want Shiro", payload["username"])
	}
	if payload["icon_emoji"] != ":robot_face:" {
		t.Fatalf("icon_emoji = %v, want :robot_face:", payload["icon_emoji"])
	}
	if _, ok := payload["blocks"]; !ok {
		t.Fatal("payload missing blocks for GitLab review button")
	}
}

func TestSlackNotifyRejectsInvalidURLs(t *testing.T) {
	module := NewSlackModule(false)
	_, err := module.Run(context.Background(), nil, workflow.Step{
		ID:   "approval",
		Type: "slack.notify",
		Config: map[string]interface{}{
			"webhook_url": "not-a-url",
			"message":     "Review decision",
		},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want invalid URL error")
	}
}
