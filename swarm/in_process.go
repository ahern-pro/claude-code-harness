package swarm

import (
	"fmt"
	"sync"
	"time"
)

type TeammateEntry struct {
	AgentID      string
	TaskID       string
	Config       TeammateSpawnConfig
	Messages     []*TeammateMessage
	StartedAt    time.Time
	ForceStopped bool
}

type InProcessBackend struct {
	mu         sync.RWMutex
	bType      BackendType
	active     map[string]*TeammateEntry
	nextTaskID uint64
}

func NewInProcessBackend() TeammateExecutor {
	return &InProcessBackend{
		bType:  BackendTypeInProcess,
		active: make(map[string]*TeammateEntry),
	}
}

func (pb *InProcessBackend) IsAvailable() bool {
	return true
}

func (pb *InProcessBackend) Spawn(config *TeammateSpawnConfig) *SpawnResult {
	if config == nil {
		return &SpawnResult{
			BackendType: pb.bType,
			Success:     false,
			Error:       "spawn config is required",
		}
	}
	if config.Name == "" || config.Team == "" {
		return &SpawnResult{
			BackendType: pb.bType,
			Success:     false,
			Error:       "spawn config requires both name and team",
		}
	}

	agentID := fmt.Sprintf("%s@%s", config.Name, config.Team)

	pb.mu.Lock()
	defer pb.mu.Unlock()

	if _, exists := pb.active[agentID]; exists {
		return &SpawnResult{
			AgentID:     agentID,
			BackendType: pb.bType,
			Success:     false,
			Error:       fmt.Sprintf("agent %q is already running", agentID),
		}
	}

	pb.nextTaskID++
	taskID := fmt.Sprintf("in_process_%06d", pb.nextTaskID)
	entry := &TeammateEntry{
		AgentID:   agentID,
		TaskID:    taskID,
		Config:    *config,
		Messages:  make([]*TeammateMessage, 0),
		StartedAt: time.Now(),
	}
	pb.active[agentID] = entry

	return &SpawnResult{
		TaskID:      taskID,
		AgentID:     agentID,
		BackendType: pb.bType,
		Success:     true,
	}
}

func (pb *InProcessBackend) SendMessage(agentId string, message *TeammateMessage) {
	if message == nil {
		return
	}

	pb.mu.Lock()
	defer pb.mu.Unlock()

	entry, ok := pb.active[agentId]
	if !ok {
		return
	}

	copied := *message
	entry.Messages = append(entry.Messages, &copied)
}

func (pb *InProcessBackend) Shutdown(agentId string, force bool) bool {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	entry, ok := pb.active[agentId]
	if !ok {
		return false
	}

	entry.ForceStopped = force
	delete(pb.active, agentId)
	return true
}

func (pb *InProcessBackend) IsActive(agentId string) bool {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	_, ok := pb.active[agentId]
	return ok
}
