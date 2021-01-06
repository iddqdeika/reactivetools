package reactivetools

import (
	"context"
	"fmt"
)

// функция для обрабтки заказов на проверку
// важно, чтобы она нормально работала с контекстом и завершалась при его закрытии.
type CheckOrderProcessorFunc func(ctx context.Context, o CheckOrder) (msg string, success bool, err error)

// инстанциирует новый процессор по данной функции для обработки заказов на проверку.
func NewCheckOrderProcessor(f CheckOrderProcessorFunc) (CheckOrderProcessor, error) {
	if f == nil {
		return nil, fmt.Errorf("must be not-nil CheckOrderProcessorFunc")
	}
	return &checkOrderProcessor{
		f: f,
	}, nil
}

// процессов заказов на проверку.
// выполняет саму проверку.
type checkOrderProcessor struct {
	f CheckOrderProcessorFunc
}

func (c *checkOrderProcessor) Process(ctx context.Context, o CheckOrder) error {
	return c.process(ctx, o)
}

func (c *checkOrderProcessor) process(ctx context.Context, o CheckOrder) error {
	msg, success, err := c.f(ctx, o)
	if err != nil {
		return err
	}
	setResult(o, msg, success)
	return nil
}

func setResult(o CheckOrder, msg string, success bool) {
	o.Result() <- NewCheckResult(o.ObjectType(), o.ObjectIdentifier(), o.CheckName(), msg, success)
}
