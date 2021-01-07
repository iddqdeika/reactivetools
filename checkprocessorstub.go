package reactivetools

import "context"

func NewStubCheckOrderProcessor() (CheckOrderProcessor, error) {
	return NewCheckOrderProcessor(&stubCheckProvider{})
}

type stubCheckProvider struct {
}

func (p *stubCheckProvider) PerformCheck(ctx context.Context, o CheckOrder) (msg string, success bool, err error) {
	return "stub_result_msg", true, nil
}
