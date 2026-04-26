package swarm

import (
	"fmt"
	"os"
	"os/exec"
)

func detectTmux() bool {
	if os.Getenv("TMUX") == "" {
		return false
	}

	_, err := exec.LookPath("tmux")
	return err == nil
}

var backendRegistry *BackendRegistry
var detectTmuxFunc = detectTmux

func GetBackendRegistry() *BackendRegistry {
	if backendRegistry != nil {
		return backendRegistry
	}

	backendRegistry = &BackendRegistry{}
	backendRegistry.registerDefaults()
	return backendRegistry
}

type BackendRegistry struct {
	backends                map[BackendType]TeammateExecutor
	detected                BackendType
	detectionResult         *BackendDetectionResult
	inProcessFallbackActive bool
}

func (br *BackendRegistry) registerDefaults() {
	if br.backends == nil {
		br.backends = make(map[BackendType]TeammateExecutor)
	}
	br.backends[BackendTypeSubprocess] = NewSubprocessBackend()
	br.backends[BackendTypeInProcess] = NewInProcessBackend()
	// todo implementation others backend
}

func (br *BackendRegistry) GetExecutor(backend BackendType) (TeammateExecutor, error) {
	if br.backends == nil {
		br.registerDefaults()
	}

	if backend == "" {
		backend = br.detectBackend()
	}

	executor, ok := br.backends[backend]
	if !ok {
		return nil, fmt.Errorf("Backend %s is not registered.", backend)
	}

	return executor, nil
}

/*
按照 backend 优先级探测可用较高级 backend，并缓存结果
*/
func (br *BackendRegistry) detectBackend() BackendType {
	if br.backends == nil {
		br.registerDefaults()
	}

	if br.detected != "" {
		return br.detected
	}

	if br.inProcessFallbackActive {
		br.detected = BackendTypeInProcess
		br.detectionResult = &BackendDetectionResult{
			Backend:  BackendTypeInProcess,
			IsNative: true,
		}
		return br.detected
	}

	if detectTmuxFunc() {
		if _, ok := br.backends[BackendTypeTmux]; ok {
			br.detected = BackendTypeTmux
			br.detectionResult = &BackendDetectionResult{
				Backend:  BackendTypeTmux,
				IsNative: true,
			}
			return br.detected
		}
	}

	br.detected = BackendTypeSubprocess
	br.detectionResult = &BackendDetectionResult{
		Backend:  BackendTypeSubprocess,
		IsNative: false,
	}
	return br.detected
}
