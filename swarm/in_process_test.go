package swarm

import "testing"

func TestInProcessBackendSpawnTracksActiveAgent(t *testing.T) {
	backend := NewInProcessBackend().(*InProcessBackend)

	result := backend.Spawn(&TeammateSpawnConfig{
		Name: "researcher",
		Team: "alpha",
	})
	if result == nil {
		t.Fatal("Spawn() returned nil")
	}
	if !result.Success {
		t.Fatalf("Spawn() success = false, error = %q", result.Error)
	}
	if result.AgentID != "researcher@alpha" {
		t.Fatalf("Spawn() agent_id = %q, want %q", result.AgentID, "researcher@alpha")
	}
	if result.BackendType != BackendTypeInProcess {
		t.Fatalf("Spawn() backend_type = %q, want %q", result.BackendType, BackendTypeInProcess)
	}

	entry, ok := backend.active[result.AgentID]
	if !ok {
		t.Fatalf("active entry for %q was not created", result.AgentID)
	}
	if entry.TaskID != result.TaskID {
		t.Fatalf("entry task_id = %q, want %q", entry.TaskID, result.TaskID)
	}
	if !backend.IsActive(result.AgentID) {
		t.Fatalf("IsActive(%q) = false, want true", result.AgentID)
	}
}

func TestInProcessBackendRejectsDuplicateSpawn(t *testing.T) {
	backend := NewInProcessBackend().(*InProcessBackend)
	config := &TeammateSpawnConfig{
		Name: "researcher",
		Team: "alpha",
	}

	first := backend.Spawn(config)
	second := backend.Spawn(config)

	if first == nil || !first.Success {
		t.Fatalf("first Spawn() = %#v, want success", first)
	}
	if second == nil {
		t.Fatal("duplicate Spawn() returned nil")
	}
	if second.Success {
		t.Fatalf("duplicate Spawn() success = true, want false")
	}
	if len(backend.active) != 1 {
		t.Fatalf("len(active) = %d, want 1", len(backend.active))
	}
}

func TestInProcessBackendSendMessageQueuesMessage(t *testing.T) {
	backend := NewInProcessBackend().(*InProcessBackend)
	result := backend.Spawn(&TeammateSpawnConfig{
		Name: "researcher",
		Team: "alpha",
	})

	message := &TeammateMessage{
		Text:      "hello from leader",
		FromAgent: "leader",
		Color:     "blue",
		Timestamp: "1710000000",
	}
	backend.SendMessage(result.AgentID, message)

	entry := backend.active[result.AgentID]
	if len(entry.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(entry.Messages))
	}
	if got := entry.Messages[0]; got.Text != message.Text || got.FromAgent != message.FromAgent {
		t.Fatalf("queued message = %#v, want %#v", got, message)
	}
}

func TestInProcessBackendShutdownRemovesAgent(t *testing.T) {
	backend := NewInProcessBackend().(*InProcessBackend)
	result := backend.Spawn(&TeammateSpawnConfig{
		Name: "researcher",
		Team: "alpha",
	})

	if ok := backend.Shutdown(result.AgentID, false); !ok {
		t.Fatalf("Shutdown(%q) = false, want true", result.AgentID)
	}
	if backend.IsActive(result.AgentID) {
		t.Fatalf("IsActive(%q) = true, want false", result.AgentID)
	}
	if ok := backend.Shutdown(result.AgentID, false); ok {
		t.Fatalf("second Shutdown(%q) = true, want false", result.AgentID)
	}
}
