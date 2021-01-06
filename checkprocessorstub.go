package reactivetools

import "context"

func NewStubCheckOrderProcessor() (CheckOrderProcessor, error) {
	return NewCheckOrderProcessor(stubCheckOrderProcessorFunc)
}

func stubCheckOrderProcessorFunc(ctx context.Context, o CheckOrder) (msg string, success bool, err error) {
	return "stub_result_msg", true, nil
}
