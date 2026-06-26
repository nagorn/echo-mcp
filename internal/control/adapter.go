package control

import "echo-mcp/internal/state"

// ToolAdapter is a local in-process adapter for tool-like control-plane calls.
type ToolAdapter struct {
	plane Plane
}

// ConfigureResponseRuleCommand is the provisional local payload for one MVP rule.
type ConfigureResponseRuleCommand struct {
	RuleID      string
	Method      string
	Path        string
	StatusCode  int
	ContentType string
	Body        string
}

// ConfigureResponseRuleResult is the provisional local result for one MVP rule.
type ConfigureResponseRuleResult struct {
	Configured bool
	RuleID     string
}

// NewToolAdapter creates a local adapter over the existing control-plane boundary.
func NewToolAdapter(plane Plane) *ToolAdapter {
	return &ToolAdapter{plane: plane}
}

// ConfigureResponseRule configures one in-memory HTTP response rule.
func (a *ToolAdapter) ConfigureResponseRule(command ConfigureResponseRuleCommand) (ConfigureResponseRuleResult, error) {
	rule := state.ResponseRule{
		ID:          command.RuleID,
		Method:      command.Method,
		Path:        command.Path,
		StatusCode:  command.StatusCode,
		ContentType: command.ContentType,
		Body:        command.Body,
	}

	if err := a.plane.ConfigureResponseRule(rule); err != nil {
		return ConfigureResponseRuleResult{}, err
	}

	return ConfigureResponseRuleResult{
		Configured: true,
		RuleID:     command.RuleID,
	}, nil
}
