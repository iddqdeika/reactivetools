package reactivetools

import (
	"encoding/json"
	"fmt"
	adapter "github.com/iddqdeika/kafka-adapter"
	"github.com/iddqdeika/rrr/helpful"
)

const (
	ConfigResultsTopicNameKey = "pim_check_results_topic"
)

// инстанциирует публикатор результатов
func NewKafkaResultPublisher(config helpful.Config, logger helpful.Logger) (CheckResultPublisher, error) {
	if config == nil {
		return nil, fmt.Errorf("must be not-nil config")
	}
	if logger == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}

	resultTopic, err := config.GetString(ConfigResultsTopicNameKey)
	if err != nil {
		return nil, err
	}

	q, err := adapter.FromConfig(config, logger)
	if err != nil {
		return nil, err
	}
	err = q.EnsureTopic(resultTopic)
	if err != nil {
		return nil, err
	}
	q.WriterRegister(resultTopic)
	return &publisher{
		q:               q,
		l:               logger,
		resultTopicName: resultTopic,
	}, nil
}

// публикатор результатов.
// публикует результаты проверок в кафка.
type publisher struct {
	q               *adapter.Queue
	l               helpful.Logger
	resultTopicName string
}

func (p *publisher) PublishCheckResult(r CheckResult) error {
	res := ResultDTO{
		ObjectType:   r.ObjectType(),
		Identifier:   r.ObjectIdentifier(),
		CheckName:    r.CheckName(),
		CheckStatus:  r.CheckSuccess(),
		CheckMessage: r.ResultMessage(),
	}
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return p.q.Put(p.resultTopicName, data)
}

type ResultDTO struct {
	ObjectType   string `json:"object_type"`
	Identifier   string `json:"identifier"`
	CheckName    string `json:"check_name"`
	CheckStatus  bool   `json:"check_status"`
	CheckMessage string `json:"check_message"`
}
