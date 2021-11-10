package statistic

import (
	"context"
	"fmt"
	kafkaadapt "github.com/iddqdeika/kafka-adapter"
	"github.com/iddqdeika/rrr/helpful"
	"time"
)

func NewStatisticAggregator(l helpful.Logger, cfg helpful.Config, adapt *kafkaadapt.Queue) (StatisticProvider, error) {
	if l == nil {
		return nil, fmt.Errorf("must be not-nil Logger")
	}
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil Config")
	}
	if adapt == nil {
		return nil, fmt.Errorf("must be not-nil Queue")
	}
	timeout, err := cfg.GetInt("cache_timeout_in_secs")
	if err != nil {
		return nil, err
	}
	topic, err := cfg.GetString("topic")
	if err != nil {
		return nil, err
	}
	adapt.ReaderRegister(topic)
	cache, err := newCache(time.Duration(timeout) * time.Second)
	if err != nil {
		return nil, err
	}
	res := &kafkaStatisticAggregator{
		l:     l,
		adapt: adapt,
		topic: topic,
		c:     cache,
	}
	go res.run(context.Background())
	return res, nil
}

type kafkaStatisticAggregator struct {
	l     helpful.Logger
	adapt *kafkaadapt.Queue
	topic string
	c     *statisticCache
}

func (k *kafkaStatisticAggregator) run(ctx context.Context) {
	k.l.Infof("kafka statistic aggregator daemon for topic %v started", k.topic)
	defer k.l.Infof("kafka statistic aggregator daemon for topic %v finished", k.topic)
	for {
		ok := k.iteration(ctx)
		if !ok {
			return
		}
	}
}

func (k *kafkaStatisticAggregator) iteration(ctx context.Context) bool {
	msg, err := k.adapt.GetWithCtx(ctx, k.topic)
	if err != nil {
		if err == context.Canceled {
			return false
		}
		k.l.Errorf("cant get stats from kafka topic %v, err: %v", k.topic, err)
		return true
	}
	defer msg.Ack()
	stat, err := unmarshal(msg.Data())
	if err != nil {
		k.l.Errorf("cant parse stat: %v", err)
		return true
	}
	k.c.set(stat.convert())
	return true
}

func (k *kafkaStatisticAggregator) Statistics() ([]Statistic, error) {
	return k.c.getAll(), nil
}
