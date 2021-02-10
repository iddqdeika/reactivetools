package reactivetools

import (
	"encoding/json"
	"fmt"
	adapter "github.com/iddqdeika/kafka-adapter"
	"github.com/iddqdeika/rrr/helpful"
	"strings"
	"time"
)

const (
	channelBuffer = 64
)

func NewChangesProvider(config helpful.Config, logger helpful.Logger,
	interceptors ...ChangesInterceptor) (ChangesProvider, error) {

	if config == nil {
		return nil, fmt.Errorf("must be not-nil config")
	}
	if logger == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}

	orderTopic, err := config.GetString("changes_topic_name")
	if err != nil {
		return nil, err
	}
	ten, err := config.GetString("target_event_name")
	if err != nil {
		return nil, err
	}
	tot, err := config.GetString("target_object_type")
	if err != nil {
		return nil, err
	}
	q, err := adapter.FromConfig(config, logger)
	if err != nil {
		return nil, err
	}

	err = q.EnsureTopic(orderTopic)
	if err != nil {
		return nil, err
	}
	q.ReaderRegister(orderTopic)

	p := &changesProvider{
		targetEventName:  ten,
		targetObjectType: tot,
		q:                q,
		l:                logger,
		orderTopicName:   orderTopic,
		ch:               make(chan ChangeEvent, channelBuffer),
	}
	for _, interceptor := range interceptors {
		if interceptor == nil {
			return nil, fmt.Errorf("must be not-nil interceptor")
		}
		p.interceptors = append(p.interceptors, interceptor)
	}
	go p.run()
	return p, nil
}

type changesProvider struct {
	targetObjectType string
	targetEventName  string

	q              *adapter.Queue
	l              helpful.Logger
	orderTopicName string
	ch             chan ChangeEvent
	interceptors   []ChangesInterceptor
}

func (p *changesProvider) run() {
	for {
		p.iteration()
	}
}

func (p *changesProvider) iteration() {
	// берем месседж
	msg, err := p.q.Get(p.orderTopicName)
	if err != nil {
		p.l.Errorf("cant get msg from topic %v, err: %v", p.orderTopicName, err)
		time.Sleep(intervalWhenCantGetMsg)
		return
	}
	cem := &ChangeEventMessage{}
	err = json.Unmarshal(msg.Data(), cem)
	if err != nil {
		p.l.Errorf("cant parse msg from topic %v, skipping, err: %v", p.orderTopicName, err)
		return
	}

	// проверяем, что тип объекта и ивент нужные
	if !(cem.ObjectType == p.targetObjectType && cem.EventName == p.targetEventName) {
		err := msg.Ack()
		if err != nil {
			p.l.Errorf("cant ack skipped msg, err: %v", err)
		}
		return
	}
	var event ChangeEvent
	event = &changeEvent{
		en:        cem.EventName,
		qm:        msg,
		change:    *cem,
		processed: make(chan struct{}),
	}
	for _, interceptor := range p.interceptors {
		ev, err := interceptor.Intercept(event)
		if err != nil {
			p.l.Infof("interceptor rejected event %v for entity(%v): %v with message: %v",
				event.EventName(), event.ObjectType(), event.ObjectIdentifier(), err)
			return
		}
		event = ev
	}

	p.ch <- event
}

func (p *changesProvider) ChangesChan() chan ChangeEvent {
	return p.ch
}

type ChangeEventMessage struct {
	ObjectType       string `json:"object_type"`
	ObjectIdentifier string `json:"object_identifier"`
	EventName        string `json:"event_name"`
	Data             string `json:"data"`
}

type changeEvent struct {
	en        string
	qm        *adapter.Message
	change    ChangeEventMessage
	processed chan struct{}
}

func (o *changeEvent) Processed() chan struct{} {
	return o.processed
}

func (o *changeEvent) EventName() string {
	return o.en
}

func (o *changeEvent) ObjectType() string {
	return o.change.ObjectType
}

func (o *changeEvent) ObjectIdentifier() string {
	return o.change.ObjectIdentifier
}

func (o *changeEvent) Data() string {
	return strings.Replace(o.change.Data, ",", ";", -1)
}

func (o *changeEvent) Ack() error {
	return o.qm.Ack()
}

func (o *changeEvent) Nack() error {
	return o.qm.Nack()
}
