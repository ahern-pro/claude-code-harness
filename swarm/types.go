package swarm

type BackendType string

const (
	BackendTypeSubprocess BackendType = "subprocess"
	BackendTypeTmux       BackendType = "tmux"
	BackendTypeInProcess  BackendType = "in_process"
	BackendTypeIterm2     BackendType = "iterm2"
)

type TeammateExecutor interface {
	IsAvailable() bool
	Spawn(*TeammateSpawnConfig) *SpawnResult
	SendMessage(agentId string, message *TeammateMessage)
	Shutdown(agentId string, force bool) bool
}

type SpawnResult struct {
	TaskID      string
	AgentID     string
	BackendType BackendType
	Success     bool
	Error       string
}

type TeammateSpawnConfig struct {
	Name             string
	Team             string
	Prompt           string
	ParentSessionID  string
	Color            string
	PlanModeRequired bool
}

type TeammateMessage struct {
	Text      string
	FromAgent string
	Color     string
	Timestamp string
}

type BackendDetectionResult struct {
	Backend    BackendType
	IsNative   bool
	NeedsSetup bool
}
