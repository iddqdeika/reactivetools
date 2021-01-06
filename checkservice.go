package reactivetools

import (
	"context"
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	"time"
)

const (
	processRetryInterval = time.Second * 5
)

func NewCheckService(cfg helpful.Config, l helpful.Logger,
	prov CheckOrderProvider, proc CheckOrderProcessor,
	pub CheckResultPublisher) (*checkService, error) {

	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil config")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}
	if prov == nil {
		return nil, fmt.Errorf("must be not-nil provider")
	}
	if proc == nil {
		return nil, fmt.Errorf("must be not-nil processor")
	}
	if pub == nil {
		return nil, fmt.Errorf("must be not-nil publisher")
	}

	parallelism, err := cfg.GetInt("parallelism")
	if err != nil {
		return nil, err
	}

	cs := &checkService{
		l:             l,
		provider:      prov,
		processor:     proc,
		publisher:     pub,
		balancer:      make(chan struct{}, parallelism),
		processing:    make(chan CheckOrder, parallelism),
		publishing:    make(chan CheckOrder, parallelism),
		acknowledging: make(chan CheckOrder, parallelism),
	}
	return cs, nil
}

type checkService struct {
	l helpful.Logger

	provider  CheckOrderProvider
	processor CheckOrderProcessor
	publisher CheckResultPublisher

	balancer      chan struct{}
	processing    chan CheckOrder
	publishing    chan CheckOrder
	acknowledging chan CheckOrder
}

func (c *checkService) Run(ctx context.Context) error {
	go c.handleProcessing(ctx)
	go c.handlePublishing(ctx)
	go c.handleAcknowledging(ctx)
	c.l.Infof("service started")
	for {
		select {
		case <-ctx.Done():
			return nil
		case o, opened := <-c.provider.OrderChan():
			if !opened {
				return fmt.Errorf("check order channel was closed")
			}
			c.l.Infof("got order %v for item %v", o.CheckName(), o.ObjectIdentifier())
			c.dispatch(ctx, o)
		}
	}
}

//берем из процессинга, ждем Result кладем в publishing публикуем результаты и закрываем Published
func (c *checkService) handleProcessing(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case o := <-c.processing:
			c.publishing <- o
			go func() {
				res := <-o.Result()
				c.publish(res)
				c.l.Infof("order %v for item %v published", o.CheckName(), o.ObjectIdentifier())
				close(o.Published())
			}()
		}
	}
}

//берем из publishing, ждём закрытия Published и кладём в acknowledging
func (c *checkService) handlePublishing(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case o := <-c.publishing:
			<-o.Published()
			c.acknowledging <- o
		}
	}
}

func (c *checkService) handleAcknowledging(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case o := <-c.acknowledging:
			o = lastOrDefault(o, c.acknowledging)
			for {
				err := o.Ack() //удалить когда adapter сможет в паралеллизм
				if err != nil {
					c.l.Errorf("cant ack published order, waiting 100ms, err: %v", err)
					time.Sleep(time.Millisecond * 100)
				} else {
					break
				}
			}
		}
	}
}

func lastOrDefault(o CheckOrder, ch chan CheckOrder) CheckOrder {
	for {
		select {
		case o = <-ch:
			continue
		default:
			return o
		}
	}
}

func (c *checkService) publish(res CheckResult) {
	for {
		err := c.publisher.PublishCheckResult(res)
		if err == nil {
			return
		}
		c.l.Errorf("cant publish check result: %v", err)
	}
}

//отправляем в очередь процессинга и запускаем процесс.
func (c *checkService) dispatch(ctx context.Context, o CheckOrder) {
	select {
	case c.balancer <- struct{}{}:
		c.processing <- o
		go func() {
			c.l.Infof("order %v for item %v dispatched", o.CheckName(), o.ObjectIdentifier())
			c.process(o)
			<-c.balancer
		}()
	case <-ctx.Done():
		return
	}
}

func (c *checkService) process(o CheckOrder) {
	for {
		err := c.processor.Process(nil, o)
		if err == nil {
			return
		}
		c.l.Errorf("err during check order processing: %v", err)
		time.Sleep(processRetryInterval)
	}
}
