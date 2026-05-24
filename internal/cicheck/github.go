package cicheck

import (
	"fmt"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
	"gopkg.in/yaml.v3"
)

// GitHubChecker validates a GitHub Actions workflow file against a Shiro workflow
type GitHubChecker struct{}

func (g *GitHubChecker) Platform() string { return "GitHub Actions" }

type githubJob struct {
	Name        string
	Steps       []githubStep
	Environment string
	Needs       []string
}

type githubStep struct {
	Name string
	Run  string
	Uses string
}

func (g *GitHubChecker) Check(wf *workflow.Workflow, ciData []byte) ([]Finding, error) {
	jobs, err := parseGitHubActions(ciData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub Actions YAML: %w", err)
	}

	var findings []Finding

	findings = append(findings, g.checkPauseSteps(wf, jobs)...)
	findings = append(findings, g.checkStateStore(wf, jobs)...)

	return findings, nil
}

// checkPauseSteps warns when workflow has pause:true — GitHub Actions has no native mid-pipeline manual gate
func (g *GitHubChecker) checkPauseSteps(wf *workflow.Workflow, jobs []githubJob) []Finding { //nolint:unparam
	var findings []Finding

	pauseSteps := pauseStepIDs(wf)
	if len(pauseSteps) == 0 {
		return nil
	}

	// Check if any job has environment protection (closest GitHub equivalent)
	hasEnvironmentProtection := false
	for _, job := range jobs {
		if job.Environment != "" {
			hasEnvironmentProtection = true
			break
		}
	}

	if !hasEnvironmentProtection {
		findings = append(findings, Finding{
			Severity: SeverityWarning,
			Rule:     "pause-needs-environment-protection",
			Message:  fmt.Sprintf("Workflow has pause:true on step(s) %v — GitHub Actions has no native mid-pipeline manual gate", pauseSteps),
			Hint:     "Add `environment: production` (or similar) with required reviewers to the job running shiro, OR split into two jobs with the second requiring manual approval via environment protection rules",
		})
	} else {
		findings = append(findings, Finding{
			Severity: SeverityInfo,
			Rule:     "pause-environment-protection-found",
			Message:  "Workflow has pause:true — environment protection rule found in CI",
			Hint:     "Ensure the environment has required reviewers configured in GitHub repository settings",
		})
	}

	return findings
}

// checkStateStore warns if state-store gitlab is used in a GitHub Actions CI file
func (g *GitHubChecker) checkStateStore(_ *workflow.Workflow, jobs []githubJob) []Finding {
	var findings []Finding

	for _, job := range jobs {
		for _, step := range job.Steps {
			if step.Run == "" {
				continue
			}
			if strings.Contains(step.Run, "shiro run") &&
				(strings.Contains(step.Run, "-state-store gitlab") || strings.Contains(step.Run, "--state-store gitlab")) {
				findings = append(findings, Finding{
					Severity: SeverityError,
					Rule:     "state-store-gitlab-wrong-platform",
					Message:  fmt.Sprintf("Job %q uses `-state-store gitlab` but this is a GitHub Actions CI file", job.Name),
					Hint:     "Use `-state-store filesystem` instead and add the state directory to `actions/upload-artifact` / `actions/download-artifact`",
				})
			}
		}
	}

	return findings
}

func parseGitHubActions(data []byte) ([]githubJob, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	jobsRaw, ok := raw["jobs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no jobs found in GitHub Actions file")
	}

	var jobs []githubJob
	for name, jobVal := range jobsRaw {
		jobMap, ok := jobVal.(map[string]interface{})
		if !ok {
			continue
		}

		job := githubJob{Name: name}

		// environment
		if env, ok := jobMap["environment"].(string); ok {
			job.Environment = env
		} else if envMap, ok := jobMap["environment"].(map[string]interface{}); ok {
			if envName, ok := envMap["name"].(string); ok {
				job.Environment = envName
			}
		}

		// needs
		if n, ok := jobMap["needs"]; ok {
			job.Needs = toStringSlice(n)
		}

		// steps
		if stepsRaw, ok := jobMap["steps"].([]interface{}); ok {
			for _, stepVal := range stepsRaw {
				stepMap, ok := stepVal.(map[string]interface{})
				if !ok {
					continue
				}
				step := githubStep{}
				if n, ok := stepMap["name"].(string); ok {
					step.Name = n
				}
				if r, ok := stepMap["run"].(string); ok {
					step.Run = r
				}
				if u, ok := stepMap["uses"].(string); ok {
					step.Uses = u
				}
				job.Steps = append(job.Steps, step)
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}
