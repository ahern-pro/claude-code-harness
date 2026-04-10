package api

type UsageSnapshot struct {
	inputTokens int
	outputTokens int
}

func (us *UsageSnapshot) TotalTokens() int {
	return us.inputTokens + us.outputTokens
}