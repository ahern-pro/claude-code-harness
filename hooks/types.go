package hooks

type AggregatedHookResult struct {
}

func (ahr *AggregatedHookResult) Blocked() bool {
	return false
}

func (ahr *AggregatedHookResult) Reason() string {
	return ""
}
