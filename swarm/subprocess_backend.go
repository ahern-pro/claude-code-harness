package swarm

type SubprocessBackend struct {
	bType BackendType
	agentTasks map[string]string
}

func NewSubprocessBackend() TeammateExecutor {
	return &SubprocessBackend{
		bType:  BackendTypeSubprocess,
		agentTasks: make(map[string]string),
	}
}

func (sb *SubprocessBackend)IsAvailable() bool {
	return true
}

func (sb *SubprocessBackend)Spawn(config *TeammateSpawnConfig) *SpawnResult {
	return  nil
}

func (sb *SubprocessBackend)SendMessage(agentId string, message *TeammateMessage) {

}

func (sb *SubprocessBackend)Shutdown(agentId string, force bool) bool {
	return  true
}