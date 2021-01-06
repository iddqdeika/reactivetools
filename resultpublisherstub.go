package reactivetools

import (
	"fmt"
	"sync"
)

// инициализирует заглушку-публикатор результатов
func NewStubResultPublisher() CheckResultPublisher {
	return &stubPublisher{}
}

// заглушка-публикатор результатов.
// пишет данные о результате в консоль и усё.
type stubPublisher struct {
	m sync.Mutex
	i int
}

func (p *stubPublisher) PublishCheckResult(r CheckResult) error {
	p.m.Lock()
	defer p.m.Unlock()
	p.i++
	fmt.Printf("got %v result with status %v and msg %v\r\n", p.i, r.CheckSuccess(), r.ResultMessage())
	return nil
}
