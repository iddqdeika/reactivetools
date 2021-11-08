package reactivetools

import (
	"context"
	"fmt"
)

var ErrNeedSkipResult = fmt.Errorf("need skip result")

// инстанциирует новый процессор по данной функции для обработки заказов на проверку.
func NewCheckOrderProcessor(p CheckProvider) (CheckOrderProcessor, error) {
	if p == nil {
		return nil, fmt.Errorf("must be not-nil CheckProvider")
	}
	return &checkOrderProcessor{
		p: p,
	}, nil
}

// процессов заказов на проверку.
// выполняет саму проверку.
type checkOrderProcessor struct {
	p CheckProvider
}

func (c *checkOrderProcessor) Process(ctx context.Context, o CheckOrder) error {
	return c.process(ctx, o)
}

func (c *checkOrderProcessor) process(ctx context.Context, o CheckOrder) error {
	msg, success, err := c.p.PerformCheck(ctx, o)
	if err != nil && err != ErrNeedSkipResult {
		return err
	}
	setResult(o, msg, success)
	return nil
}

func setResult(o CheckOrder, msg string, success bool) {
	o.Result() <- NewCheckResult(o.ObjectType(), o.ObjectIdentifier(), o.CheckName(), msg, success)
}
