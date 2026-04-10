package hooks

type HooksExecutor struct {
}

func (he *HooksExecutor) Execute(event string, payload map[string]any) *AggregatedHookResult {
	return nil
}
