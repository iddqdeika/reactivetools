package reactivetools

import "context"

type CheckOrder interface {
	ObjectType() string
	ObjectIdentifier() string
	CheckName() string
	Result() chan CheckResult
	Published() chan struct{}
	Ack() error
	Nack() error
}

type CheckResult interface {
	ObjectType() string
	ObjectIdentifier() string
	CheckName() string
	ResultMessage() string
	CheckSuccess() bool
}

type CheckOrderProvider interface {
	OrderChan() chan CheckOrder
}

type CheckOrderProcessor interface {
	Process(ctx context.Context, o CheckOrder) error
}

type CheckResultPublisher interface {
	PublishCheckResult(r CheckResult) error
}

type StatisticProvider interface {
	Statistics() ([]Statistic, error)
}

type Statistic interface {
	Name() string
	Value() string
	Description() string
}
