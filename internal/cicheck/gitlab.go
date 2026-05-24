package cicheck

import (
	"fmt"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
	"gopkg.in/yaml.v3"
)

// GitLabChecker validates a .gitlab-ci.yml against a workflow
type GitLabChecker struct{}

func (g *GitLabChecker) Platform() string { return "GitLab CI" }

// gitlabJob represents a parsed GitLab CI job
type gitlabJob struct {
	Name        string
	Script      []string
	When        string
	Needs       []string
	Artifacts   gitlabArtifacts
	Stage       string
	Environment string
}

type gitlabArtifacts struct {
	Paths []string
}

func (g *GitLabChecker) Check(wf *workflow.Workflow, ciData []byte) ([]Finding, error) {
	jobs, err := parseGitLabCI(ciData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitLab CI YAML: %w", err)
	}

	var findings []Finding

	findings = append(findings, g.checkPauseSteps(wf, jobs)...)
	findings = append(findings, g.checkStateStoreArtifacts(wf, jobs)...)

	return findings, nil
}

// checkPauseSteps ensures every workflow step with pause:true has:
// 1. An initial job that runs shiro run (non-manual)
// 2. A resume job with when:manual + needs:[initial job]
func (g *GitLabChecker) checkPauseSteps(wf *workflow.Workflow, jobs []gitlabJob) []Finding { //nolint:unparam
	var findings []Finding

	pauseSteps := pauseStepIDs(wf)
	if len(pauseSteps) == 0 {
		return nil
	}

	// Find jobs that run shiro (initial vs resume)
	var shiroJobs []gitlabJob
	for _, job := range jobs {
		if jobRunsShiro(job) {
			shiroJobs = append(shiroJobs, job)
		}
	}

	if len(shiroJobs) == 0 {
		findings = append(findings, Finding{
			Severity: SeverityError,
			Rule:     "pause-needs-manual-resume",
			Message:  fmt.Sprintf("Workflow has %d step(s) with pause:true but no jobs running shiro found in CI file", len(pauseSteps)),
			Hint:     "Add a job that runs `shiro run` for the initial stage and a second job with `when: manual` for the resume stage",
		})
		return findings
	}

	// Find manual resume jobs
	var manualJobs []gitlabJob
	var normalJobs []gitlabJob
	for _, job := range shiroJobs {
		if job.When == "manual" {
			manualJobs = append(manualJobs, job)
		} else {
			normalJobs = append(normalJobs, job)
		}
	}

	if len(manualJobs) == 0 {
		findings = append(findings, Finding{
			Severity: SeverityError,
			Rule:     "pause-needs-manual-resume",
			Message:  fmt.Sprintf("Workflow has pause:true on step(s) %v but no manual resume job found in CI", pauseSteps),
			Hint:     "Add a second job with `when: manual` that runs `shiro run` (without -fresh flag) to resume after pause",
		})
		return findings
	}

	// Check that each manual job has needs: pointing to an initial job
	for _, manualJob := range manualJobs {
		if len(manualJob.Needs) == 0 {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				Rule:     "pause-resume-needs-dependency",
				Message:  fmt.Sprintf("Manual resume job %q has no `needs:` — it won't receive state from the initial job", manualJob.Name),
				Hint:     fmt.Sprintf("Add `needs: [<initial-job-name>]` to job %q so it receives the .shiro/ artifact", manualJob.Name),
			})
		}
	}

	// Check that initial jobs use -fresh and resume jobs don't
	for _, normalJob := range normalJobs {
		scriptStr := strings.Join(normalJob.Script, " ")
		if strings.Contains(scriptStr, "shiro run") && !strings.Contains(scriptStr, "-fresh") && !strings.Contains(scriptStr, "--fresh") {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				Rule:     "pause-initial-needs-fresh",
				Message:  fmt.Sprintf("Initial job %q runs `shiro run` without `-fresh` flag", normalJob.Name),
				Hint:     "Add `-fresh` flag to the initial job so it starts a fresh workflow execution, not a resume",
			})
		}
	}

	return findings
}

// checkStateStoreArtifacts ensures jobs using -state-store gitlab expose .shiro/ as an artifact
func (g *GitLabChecker) checkStateStoreArtifacts(_ *workflow.Workflow, jobs []gitlabJob) []Finding {
	var findings []Finding

	for _, job := range jobs {
		scriptStr := strings.Join(job.Script, " ")
		if !strings.Contains(scriptStr, "shiro run") {
			continue
		}
		if !strings.Contains(scriptStr, "-state-store gitlab") && !strings.Contains(scriptStr, "--state-store gitlab") {
			continue
		}

		hasShiroArtifact := false
		for _, path := range job.Artifacts.Paths {
			if strings.Contains(path, ".shiro") {
				hasShiroArtifact = true
				break
			}
		}

		if !hasShiroArtifact {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Rule:     "state-store-gitlab-needs-artifact",
				Message:  fmt.Sprintf("Job %q uses `-state-store gitlab` but does not expose `.shiro/` as an artifact", job.Name),
				Hint:     "Add `artifacts: { paths: [.shiro/] }` to this job so state is available to downstream jobs",
			})
		}
	}

	return findings
}

// parseGitLabCI parses a .gitlab-ci.yml and returns the list of jobs
func parseGitLabCI(data []byte) ([]gitlabJob, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	reservedKeys := map[string]bool{
		"stages": true, "variables": true, "image": true, "services": true,
		"before_script": true, "after_script": true, "cache": true, "include": true,
		"workflow": true, "default": true,
	}

	var jobs []gitlabJob
	for key, val := range raw {
		if reservedKeys[key] {
			continue
		}
		jobMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}

		job := gitlabJob{Name: key}

		// script
		if s, ok := jobMap["script"]; ok {
			job.Script = toStringSlice(s)
		}

		// when
		if w, ok := jobMap["when"].(string); ok {
			job.When = w
		}

		// stage
		if st, ok := jobMap["stage"].(string); ok {
			job.Stage = st
		}

		// needs
		if n, ok := jobMap["needs"]; ok {
			job.Needs = parseNeeds(n)
		}

		// artifacts.paths
		if a, ok := jobMap["artifacts"].(map[string]interface{}); ok {
			if paths, ok := a["paths"]; ok {
				job.Artifacts.Paths = toStringSlice(paths)
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func jobRunsShiro(job gitlabJob) bool {
	for _, line := range job.Script {
		if strings.Contains(line, "shiro run") || strings.Contains(line, "shiro validate") {
			return true
		}
	}
	return false
}

func pauseStepIDs(wf *workflow.Workflow) []string {
	var ids []string
	for _, step := range wf.Steps {
		if step.Pause {
			ids = append(ids, step.ID)
		}
	}
	return ids
}

func toStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case []interface{}:
		var result []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		return []string{t}
	}
	return nil
}

func parseNeeds(v interface{}) []string {
	switch t := v.(type) {
	case []interface{}:
		var result []string
		for _, item := range t {
			switch n := item.(type) {
			case string:
				result = append(result, n)
			case map[string]interface{}:
				if job, ok := n["job"].(string); ok {
					result = append(result, job)
				}
			}
		}
		return result
	}
	return nil
}
