package reactivetools

import (
	"context"
	"encoding/json"
	"fmt"
	adapter "github.com/iddqdeika/kafka-adapter"
	"github.com/iddqdeika/reactivetools/statistic"
	helpful "github.com/iddqdeika/rrr/helpful"
	"strconv"
	"time"
)

const (
	ConfigOrderTopicNameKey = "pim_check_orders_topic"
	ConfigObjectTypeKey     = "object_type"
	ConfigCheckNameKey      = "check_name"

	intervalWhenCantGetMsg  = time.Second
	checkOrderChannelBuffer = 64

	lagRetrievingTimeout = time.Second * 5
)

func NewKafkaOrderProvider(config helpful.Config, logger helpful.Logger) (CheckOrderProvider, error) {

	if config == nil {
		return nil, fmt.Errorf("must be not-nil config")
	}
	if logger == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}

	orderTopic, err := config.GetString(ConfigOrderTopicNameKey)
	if err != nil {
		return nil, err
	}
	objectType, err := config.GetString(ConfigObjectTypeKey)
	if err != nil {
		return nil, err
	}
	checkName, err := config.GetString(ConfigCheckNameKey)
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
	p := &checkOrderProvider{
		orderTopicName: orderTopic,
		objectType:     objectType,
		checkName:      checkName,
		q:              q,
		l:              logger,
		ch:             make(chan CheckOrder, checkOrderChannelBuffer),
	}
	go p.run()
	return p, nil
}

// читает нужный топик из данной Kafka и получает оттуда CheckOrder
// из них собирает все подходящие по названию проверки и типу объекта
// остальные - пропускает (подтверждая)
// выбранные заказы на проверку пхает в очередь, доступную по методу OrderChan()
type checkOrderProvider struct {
	orderTopicName string
	objectType     string
	checkName      string

	q  *adapter.Queue
	l  helpful.Logger
	ch chan CheckOrder
}

// дает статистики по провайдеру
// пока это только лаг очереди, которую он смотрит.
// не ну а шо, эт уже неплохо.
func (p *checkOrderProvider) Statistics() ([]statistic.Statistic, error) {
	ss := make([]statistic.Statistic, 0)
	lags, err := p.getKafkaLagStatistic()
	if err != nil {
		return nil, err
	}
	ss = append(ss, lags)
	return ss, nil
}

func (p *checkOrderProvider) getKafkaLagStatistic() (statistic.Statistic, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lagRetrievingTimeout)
	defer cancel()
	lag, err := p.q.GetConsumerLagForSinglePartition(ctx, p.orderTopicName)
	if err != nil {
		return nil, fmt.Errorf("cant get consumer lag for kafka: %v", err)
	}
	return &SimpleStatistic{
		N:    fmt.Sprintf("Consumer lag for check \"%v\" (object type: %v)", p.checkName, p.objectType),
		V:    strconv.Itoa(int(lag)),
		Desc: `Очередь на проверку. Разница между оффсетами последних обработанного и записанного сообщений.`,
	}, nil
}

type SimpleStatistic struct {
	N    string
	V    string
	Desc string
}

func (s SimpleStatistic) Error() error {
	return nil
}

func (s SimpleStatistic) Name() string {
	return s.N
}

func (s SimpleStatistic) Value() string {
	return s.V
}

func (s SimpleStatistic) Description() string {
	return s.Desc
}

func (p *checkOrderProvider) run() {
	for {
		p.iteration()
	}
}

func (p *checkOrderProvider) iteration() {
	msg, err := p.q.Get(p.orderTopicName)
	if err != nil {
		p.l.Errorf("cant get msg from topic %v, err: %v", p.orderTopicName, err)
		time.Sleep(intervalWhenCantGetMsg)
		return
	}
	od := &checkOrderData{}
	err = json.Unmarshal(msg.Data(), od)
	if err != nil {
		p.l.Errorf("cant parse msg from topic %v, skipping, err: %v", p.orderTopicName, err)
		return
	}

	//skip other msgs
	if !(od.ObjectType == p.objectType && od.CheckName == p.checkName) {
		err := msg.Ack()
		if err != nil {
			p.l.Errorf("cant ack skipped msg, err: %v", err)
		}
		return
	}
	order := newCheckOrder(od.CheckName, od.ObjectType, od.ObjectIdentifier, msg)
	p.ch <- order
}

func (p *checkOrderProvider) OrderChan() chan CheckOrder {
	return p.ch
}

type checkOrderData struct {
	ObjectType       string `json:"object_type"`
	CheckName        string `json:"check_name"`
	ObjectIdentifier string `json:"object_identifier"`
}
