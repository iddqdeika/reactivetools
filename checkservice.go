package reactivetools

import (
	"context"
	"fmt"
	"github.com/iddqdeika/reactivetools/statistic"
	"github.com/iddqdeika/rrr"
	"github.com/iddqdeika/rrr/helpful"
	"time"
)

const (
	processRetryInterval = time.Second * 5

	CheckOrderProviderConfigKey   = "check_order_provider"
	CheckResultPublisherConfigKey = "check_result_publisher"
	StatisticServiceConfigKey     = "statistics"
)

// инстанциирует сервис проверки, инициализируя провайдер и паблишер из конфига
// стоит использовать, когда надо сделать стандартный сервис.
// в конфиге должны быть соответствующие компонентам дети (Child): provider и publisher
// также необходимо передать реализацию CheckProvider (сама логика проверки) и Logger
func NewKafkaCheckService(cfg helpful.Config, l helpful.Logger, p CheckProvider) (CheckService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil Config")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil Logger")
	}
	if p == nil {
		return nil, fmt.Errorf("must be not-nil CheckProvider")
	}

	// соберем провайдера
	prov, err := NewKafkaOrderProvider(cfg.Child(CheckOrderProviderConfigKey), l)
	if err != nil {
		return nil, err
	}

	// статистик сервис
	stats, err := statistic.NewStatisticService(cfg.Child(StatisticServiceConfigKey), prov, l)
	if err != nil {
		return nil, err
	}

	// соберем процессор с данной функцией-обработчиком
	proc, err := NewCheckOrderProcessor(p)
	if err != nil {
		return nil, err
	}

	// соберем паблишер
	pub, err := NewKafkaResultPublisher(cfg.Child(CheckResultPublisherConfigKey), l)
	if err != nil {
		return nil, err
	}

	// собираем сам сервис
	return NewCheckService(cfg, l, prov, proc, pub, stats)
}

// инстанциирует сервис проверки с данными компонентами.
func NewCheckService(cfg helpful.Config, l helpful.Logger,
	prov CheckOrderProvider, proc CheckOrderProcessor,
	pub CheckResultPublisher, stats Service) (CheckService, error) {

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
		stats:         stats,
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

	stats Service

	balancer      chan struct{}
	processing    chan CheckOrder
	publishing    chan CheckOrder
	acknowledging chan CheckOrder
}

func (c *checkService) Run(ctx context.Context) error {
	// соберем сервисы для запуска (помимо самого сервиса проверок надо запустить, например, статистику, если она задана)
	var services []rrr.Service
	services = append(services, &serviceSurrogate{callback: c.run})
	if c.stats != nil {
		services = append(services, c.stats)
	}
	errs := rrr.RunServices(ctx, services...)
	return rrr.ComposeErrors("CheckService", errs...)
}

type serviceSurrogate struct {
	callback func(ctx context.Context) error
}

func (s *serviceSurrogate) Run(ctx context.Context) error {
	return s.callback(ctx)
}

func (c *checkService) run(ctx context.Context) error {
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
				c.l.Infof("provider's order chan was closed, finishing")
				return nil
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
				defer close(o.Published())
				res := <-o.Result()
				c.publish(res)
				c.l.Infof("order %v for item %v published", o.CheckName(), o.ObjectIdentifier())
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
			o = lastOrDefaultOrder(o, c.acknowledging)
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

func lastOrDefaultOrder(o CheckOrder, ch chan CheckOrder) CheckOrder {
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
