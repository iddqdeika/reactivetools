package reactivetools

import (
	"strconv"
	"time"
)

func NewChangesProviderStub(count int, interval time.Duration) (ChangesProvider, error) {
	p := &stubChangesProvider{ch: make(chan ChangeEvent, count)}
	go p.run(count, interval)
	return p, nil
}

type stubChangesProvider struct {
	ch chan ChangeEvent
}

func (s *stubChangesProvider) ChangesChan() chan ChangeEvent {
	return s.ch
}

func (s *stubChangesProvider) run(count int, interval time.Duration) {
	for i := 0; i < count; i++ {
		s.ch <- newStubChangeEvent("stubObject", strconv.Itoa(i), "stubEvent", "stubData")
	}
}

func newStubChangeEvent(objectType, objectIdentifier, eventName, data string) ChangeEvent {
	return &stubChangeEvent{
		ot:   objectType,
		oi:   objectIdentifier,
		en:   eventName,
		data: data,
		ch:   make(chan struct{}, 0),
	}
}

type stubChangeEvent struct {
	ot   string
	oi   string
	en   string
	data string
	ch   chan struct{}
}

func (s *stubChangeEvent) ObjectType() string {
	return s.ot
}

func (s *stubChangeEvent) ObjectIdentifier() string {
	return s.oi
}

func (s *stubChangeEvent) EventName() string {
	return s.en
}

func (s *stubChangeEvent) Data() string {
	return s.data
}

func (s *stubChangeEvent) Ack() error {
	return nil
}

func (s *stubChangeEvent) Nack() error {
	return nil
}

func (s *stubChangeEvent) Processed() chan struct{} {
	return s.ch
}
