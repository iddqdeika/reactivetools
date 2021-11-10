package statistic

import (
	"context"
	"fmt"
	adapter "github.com/iddqdeika/kafka-adapter"
	"github.com/iddqdeika/rrr"
	"github.com/iddqdeika/rrr/helpful"
	"time"
)

// NewStatisticSender инициализирует отправщик статистики.
func NewStatisticSender(l helpful.Logger, cfg helpful.Config, provider StatisticProvider, kafkaAdapt *adapter.Queue) (rrr.Service, error) {
	if provider == nil {
		return nil, fmt.Errorf("must be not-nil StatisticProvider")
	}
	if kafkaAdapt == nil {
		return nil, fmt.Errorf("must be not-nil kafkaAdapter.Queue")
	}
	topic, err := cfg.GetString("topic")
	if err != nil {
		return nil, err
	}
	kafkaAdapt.WriterRegister(topic)

	interval, err := cfg.GetInt("interval_in_secs")
	if err != nil {
		return nil, err
	}

	return &kafkaStatisticSender{
		l:        l,
		p:        provider,
		adapter:  kafkaAdapt,
		topic:    topic,
		interval: time.Duration(interval) * time.Second,
	}, nil
}

type kafkaStatisticSender struct {
	l        helpful.Logger
	p        StatisticProvider
	adapter  *adapter.Queue
	topic    string
	interval time.Duration
}

func (k *kafkaStatisticSender) Run(ctx context.Context) error {
	k.l.Infof("kafka statistic sender started")
	defer k.l.Infof("kafka statistic sender finishing")

	t := time.NewTicker(k.interval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			err := k.iterate(ctx)
			if err != nil {
				k.l.Errorf("err during statistics send: %v")
			}
			t.Reset(k.interval)
		}
	}
}

func (k *kafkaStatisticSender) iterate(ctx context.Context) error {
	stats, err := k.p.Statistics()
	if err != nil {
		return fmt.Errorf("cant get statistics to send")
	}
	return k.send(ctx, stats)
}

func (k *kafkaStatisticSender) send(ctx context.Context, stats []Statistic) error {
	for _, stat := range stats {
		err := k.sendSingleStatistic(ctx, stat)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *kafkaStatisticSender) sendSingleStatistic(ctx context.Context, stat Statistic) error {
	dto, err := convert(stat)
	if err != nil {
		return fmt.Errorf("cant convert statistic: %v", err)
	}
	return k.adapter.Put(k.topic, dto.marshal())
}
