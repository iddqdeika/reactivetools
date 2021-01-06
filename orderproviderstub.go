package reactivetools

import (
	"context"
	"time"
)

func NewStubOrderProvider(ctx context.Context, count int, interval time.Duration) CheckOrderProvider {
	op := &stubOrderProvider{
		count:    count,
		interval: interval,
		ch:       make(chan CheckOrder, count),
	}
	go op.generate(ctx)
	return op
}

type stubOrderProvider struct {
	count    int
	interval time.Duration
	ch       chan CheckOrder
}

func (s *stubOrderProvider) generate(ctx context.Context) {
	for i := s.count; i > 0; i-- {
		select {
		case <-ctx.Done():
			return
		case s.ch <- newStubCheckOrder():
			time.Sleep(s.interval)
		}
	}
	close(s.ch)
}

func (s *stubOrderProvider) OrderChan() chan CheckOrder {
	return s.ch
}

func newStubCheckOrder() CheckOrder {
	return &stubCheckOrder{
		rch: make(chan CheckResult, 1),
		pub: make(chan struct{}),
	}
}

type stubCheckOrder struct {
	rch chan CheckResult
	pub chan struct{}
}

func (s *stubCheckOrder) ObjectType() string {
	return "stub_object_type"
}

func (s *stubCheckOrder) ObjectIdentifier() string {
	return "stub_object_identifier"
}

func (s *stubCheckOrder) CheckName() string {
	return "stub_check_name"
}

func (s *stubCheckOrder) Result() chan CheckResult {
	s.rch <- NewCheckResult("stub_type", s.ObjectIdentifier(), s.CheckName(), "stub_result", true)
	return s.rch
}

func (s *stubCheckOrder) Published() chan struct{} {
	return s.pub
}

func (s *stubCheckOrder) Ack() error {
	return nil
}

func (s *stubCheckOrder) Nack() error {
	return nil
}
