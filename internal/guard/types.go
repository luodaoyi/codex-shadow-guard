package guard

// HookInput is the stable subset of the JSON delivered by Codex lifecycle hooks.
// Unknown fields are intentionally ignored for forward compatibility.
type HookInput struct {
	SessionID     string         `json:"session_id"`
	TurnID        string         `json:"turn_id"`
	Transcript    string         `json:"transcript_path"`
	CWD           string         `json:"cwd"`
	Event         string         `json:"hook_event_name"`
	Permission    string         `json:"permission_mode"`
	ToolName      string         `json:"tool_name"`
	ToolInput     map[string]any `json:"tool_input"`
	ToolResponse  map[string]any `json:"tool_response"`
	LastAssistant string         `json:"last_assistant_message"`
}

// Decision is printed as hook JSON only when Codex must be stopped or blocked.
type Decision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// Finding is an auditable, deterministic result. The first version deliberately
// makes no model call and never uploads a repository diff.
type Finding struct {
	Level   string
	Message string
}

// Report is written to .codex-shadow-guard/latest-report.json inside the project.
type Report struct {
	Version  int       `json:"version"`
	Event    string    `json:"event"`
	Findings []Finding `json:"findings"`
}

const (
	AgentsBegin = "<!-- codex-shadow-guard:begin -->"
	AgentsEnd   = "<!-- codex-shadow-guard:end -->"
)

const AgentsBlock = AgentsBegin + `
<!-- Managed by codex-shadow-guard. Remove with: codex-shadow-guard uninstall -->
## Codex Shadow Guard

Before changing code, read the related implementation, tests, and project instructions.
Keep the change within the user's stated scope. Do not add speculative abstractions,
fallbacks, dependencies, or unrelated refactors.

Before claiming completion, inspect the diff and run the relevant verification, or state
clearly why verification was not run. Treat a Shadow Guard block as required feedback:
fix the concrete issue, narrow the change, or ask the user to accept the risk.
` + AgentsEnd + "\n"

const ManagedCommandMarker = "codex-shadow-guard hook"
const HookCommandArgument = "hook"
const ToolRecordFile = ".codex-shadow-guard/tool-events.jsonl"
const ReportFile = ".codex-shadow-guard/latest-report.json"
