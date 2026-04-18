package sandbox

func IsDockerSandboxActive() bool {
	return false
}

// ValidateSandboxPath checks whether the resolved path is allowed
// within the sandbox relative to the working directory.
// Returns (allowed, reason).
func ValidateSandboxPath(resolved string, cwd string) (bool, string) {
	// TODO: implement full sandbox path validation
	return true, ""
}