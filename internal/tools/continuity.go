package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scopweb/mcp-go-context/internal/continuity"
)

type continuityServer interface {
	Continuity() *continuity.Service
}

type projectSaver interface {
	StoreForProject(project, key, content string, tags []string, decisionType, reason string, alternatives []string) error
}

func continuityOf(server interface{}) (*continuity.Service, error) {
	src, ok := server.(continuityServer)
	if !ok || src.Continuity() == nil {
		return nil, fmt.Errorf("continuity service is not available")
	}
	return src.Continuity(), nil
}

func storeMemory(server interface{}, mem MemoryInterface, path, project, key, content string, tags []string, decisionType, reason string, alternatives []string) error {
	projectID, err := resolveProject(server, path, project, true)
	if err != nil {
		return err
	}
	if projectID != "" {
		if saver, ok := mem.(projectSaver); ok {
			return saver.StoreForProject(projectID, key, content, tags, decisionType, reason, alternatives)
		}
	}
	if decisionType != "" || reason != "" || len(alternatives) > 0 {
		return mem.StoreWithType(key, content, tags, decisionType, reason, alternatives)
	}
	return mem.Store(key, content, tags)
}

func resolveProject(server interface{}, path, projectID string, create bool) (string, error) {
	if strings.TrimSpace(path) == "" && strings.TrimSpace(projectID) == "" {
		return "", nil
	}
	svc, err := continuityOf(server)
	if err != nil {
		return "", err
	}
	resolved, err := svc.Resolve(path, projectID, create)
	if err != nil {
		return "", err
	}
	return resolved.ID, nil
}

// SaveHandoffHandler stores a recoverable work summary for another client.
func SaveHandoffHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Path             string   `json:"path"`
		ProjectID        string   `json:"projectId"`
		HandoffID        string   `json:"handoffId"`
		ExpectedRevision int      `json:"expectedRevision"`
		SourceClient     string   `json:"sourceClient"`
		Objective        string   `json:"objective"`
		Completed        []string `json:"completed"`
		Verification     []string `json:"verification"`
		Pending          []string `json:"pending"`
		NextStep         string   `json:"nextStep"`
		References       []string `json:"references"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	svc, err := continuityOf(server)
	if err != nil {
		return createErrorResponse(err.Error())
	}
	result, err := svc.Save(continuity.Input{
		Path:             params.Path,
		ProjectID:        params.ProjectID,
		HandoffID:        params.HandoffID,
		ExpectedRevision: params.ExpectedRevision,
		SourceClient:     params.SourceClient,
		Objective:        params.Objective,
		Completed:        params.Completed,
		Verification:     params.Verification,
		Pending:          params.Pending,
		NextStep:         params.NextStep,
		References:       params.References,
	})
	if conflict, ok := continuity.IsConflict(err); ok {
		return textResponse(fmt.Sprintf(
			"CONFLICT: handoff `%s` is at revision %d and was kept. Incoming work was preserved as `%s`. Resume both and reconcile; do not overwrite either.",
			conflict.Current.HandoffID, conflict.Current.Revision, conflict.SavedAs,
		)), nil
	}
	if err != nil {
		return createErrorResponse(err.Error())
	}
	return textResponse(fmt.Sprintf(
		"Handoff saved.\n- projectId: %s\n- handoffId: %s\n- revision: %d\n- branch: %s\n- commit: %s",
		result.ProjectID, result.HandoffID, result.Revision, result.Branch, result.Commit,
	)), nil
}

// ResumeContextHandler returns the shared work state for a working path.
func ResumeContextHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Path      string `json:"path"`
		ProjectID string `json:"projectId"`
		HandoffID string `json:"handoffId"`
		Query     string `json:"query"`
		Depth     string `json:"depth"`
		MaxTokens int    `json:"maxTokens"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	svc, err := continuityOf(server)
	if err != nil {
		return createErrorResponse(err.Error())
	}
	view, err := svc.Resume(params.Path, params.ProjectID, params.HandoffID, params.Query, params.Depth, params.MaxTokens)
	if err != nil {
		return createErrorResponse(err.Error())
	}
	return textResponse(continuity.FormatResume(view, params.MaxTokens)), nil
}
