package swarm

import "testing"

type stubExecutor struct{}

func (stubExecutor) IsAvailable() bool { return true }

func (stubExecutor) Spawn(*TeammateSpawnConfig) *SpawnResult { return nil }

func (stubExecutor) SendMessage(string, *TeammateMessage) {}

func (stubExecutor) Shutdown(string, bool) bool { return true }

func TestDetectBackendUsesCachedValue(t *testing.T) {
	registry := &BackendRegistry{
		detected: BackendTypeTmux,
	}

	detectTmuxFunc = func() bool {
		t.Fatal("tmux detection should not run when backend is cached")
		return false
	}
	defer func() {
		detectTmuxFunc = detectTmux
	}()

	if got := registry.detectBackend(); got != BackendTypeTmux {
		t.Fatalf("detectBackend() = %q, want %q", got, BackendTypeTmux)
	}
}

func TestDetectBackendPrefersInProcessFallback(t *testing.T) {
	registry := &BackendRegistry{
		inProcessFallbackActive: true,
	}

	detectTmuxFunc = func() bool {
		t.Fatal("tmux detection should not run when in-process fallback is active")
		return false
	}
	defer func() {
		detectTmuxFunc = detectTmux
	}()

	if got := registry.detectBackend(); got != BackendTypeInProcess {
		t.Fatalf("detectBackend() = %q, want %q", got, BackendTypeInProcess)
	}

	if registry.detectionResult == nil || registry.detectionResult.Backend != BackendTypeInProcess {
		t.Fatalf("detectionResult = %#v, want backend %q", registry.detectionResult, BackendTypeInProcess)
	}
}

func TestDetectBackendPrefersTmuxWhenDetectedAndRegistered(t *testing.T) {
	originalDetectTmux := detectTmuxFunc
	detectTmuxFunc = func() bool { return true }
	defer func() {
		detectTmuxFunc = originalDetectTmux
	}()

	registry := &BackendRegistry{
		backends: map[BackendType]TeammateExecutor{
			BackendTypeTmux: stubExecutor{},
		},
	}

	if got := registry.detectBackend(); got != BackendTypeTmux {
		t.Fatalf("detectBackend() = %q, want %q", got, BackendTypeTmux)
	}

	if registry.detectionResult == nil || registry.detectionResult.Backend != BackendTypeTmux {
		t.Fatalf("detectionResult = %#v, want backend %q", registry.detectionResult, BackendTypeTmux)
	}
}

func TestDetectBackendFallsBackToSubprocess(t *testing.T) {
	originalDetectTmux := detectTmuxFunc
	detectTmuxFunc = func() bool { return false }
	defer func() {
		detectTmuxFunc = originalDetectTmux
	}()

	registry := &BackendRegistry{}

	if got := registry.detectBackend(); got != BackendTypeSubprocess {
		t.Fatalf("detectBackend() = %q, want %q", got, BackendTypeSubprocess)
	}

	if registry.detectionResult == nil || registry.detectionResult.Backend != BackendTypeSubprocess {
		t.Fatalf("detectionResult = %#v, want backend %q", registry.detectionResult, BackendTypeSubprocess)
	}
}

func TestDetectBackendFallsBackToSubprocessWhenTmuxIsNotRegistered(t *testing.T) {
	originalDetectTmux := detectTmuxFunc
	detectTmuxFunc = func() bool { return true }
	defer func() {
		detectTmuxFunc = originalDetectTmux
	}()

	registry := &BackendRegistry{
		backends: map[BackendType]TeammateExecutor{},
	}

	if got := registry.detectBackend(); got != BackendTypeSubprocess {
		t.Fatalf("detectBackend() = %q, want %q", got, BackendTypeSubprocess)
	}
}

func TestGetBackendRegistryRegistersDefaults(t *testing.T) {
	originalRegistry := backendRegistry
	backendRegistry = nil
	defer func() {
		backendRegistry = originalRegistry
	}()

	registry := GetBackendRegistry()
	if registry == nil {
		t.Fatal("GetBackendRegistry() returned nil")
	}
	if _, ok := registry.backends[BackendTypeSubprocess]; !ok {
		t.Fatalf("subprocess backend was not registered")
	}
	if _, ok := registry.backends[BackendTypeInProcess]; !ok {
		t.Fatalf("in-process backend was not registered")
	}
}
