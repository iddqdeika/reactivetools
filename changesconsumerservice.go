package reactivetools

import (
	"context"
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	"time"
)

func NewChangesConsumerService(cfg helpful.Config, l helpful.Logger, p ChangesProvider, s ChangesProcessor) (ChangesConsumerService, error) {

	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil config")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}
	if p == nil {
		return nil, fmt.Errorf("must be not-nil provider")
	}
	if s == nil {
		return nil, fmt.Errorf("must be not-nil saver")
	}

	parallelism, err := cfg.GetInt("parallelism")
	if err != nil {
		return nil, err
	}

	c := &consumer{
		l:             l,
		prov:          p,
		proc:          s,
		balancer:      make(chan struct{}, parallelism),
		processing:    make(chan ChangeEvent, parallelism),
		acknowledging: make(chan ChangeEvent, parallelism),
	}

	return c, nil
}

type consumer struct {
	l    helpful.Logger
	prov ChangesProvider
	proc ChangesProcessor

	balancer      chan struct{}
	processing    chan ChangeEvent
	acknowledging chan ChangeEvent
}

func (c *consumer) Run(ctx context.Context) error {
	go c.handleProcessing(ctx)
	go c.handleAcknowledging(ctx)
	c.l.Infof("service started")
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-c.prov.ChangesChan():
			c.l.Infof("got event %v for %v(%v)", e.EventName(), e.ObjectType(), e.ObjectIdentifier())
			c.dispatch(ctx, e)
		}
	}
}

func (c *consumer) handleProcessing(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-c.processing:
			<-e.Processed()
			c.acknowledging <- e
		}
	}
}

func (c *consumer) handleAcknowledging(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-c.acknowledging:
			e = lastOrDefaultChange(e, c.acknowledging)
			for {
				err := e.Ack() //удалить когда adapter сможет в паралеллизм
				if err != nil {
					c.l.Errorf("cant ack published order, waiting 100ms, err: %v", err)
					time.Sleep(time.Millisecond * 100)
				} else {
					c.l.Infof("event %v for %v(%v) acked", e.EventName(), e.ObjectType(), e.ObjectIdentifier())
					break
				}
			}
		}
	}
}

func (c *consumer) dispatch(ctx context.Context, e ChangeEvent) {
	select {
	case c.balancer <- struct{}{}:
		c.processing <- e
		go func() {
			c.l.Infof("event %v for %v(%v) dispatched", e.EventName(), e.ObjectType(), e.ObjectIdentifier())
			c.process(e)
			close(e.Processed())
			<-c.balancer
		}()
	case <-ctx.Done():
		return
	}
}

func (c *consumer) process(e ChangeEvent) {
	for {
		err := c.proc.Process(e)
		if err == nil {
			return
		}
		c.l.Errorf("err during check order processing: %v", err)
		time.Sleep(processRetryInterval)
	}
}

func lastOrDefaultChange(o ChangeEvent, ch chan ChangeEvent) ChangeEvent {
	for {
		select {
		case o = <-ch:
			continue
		default:
			return o
		}
	}
}
