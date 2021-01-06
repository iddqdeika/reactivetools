package reactivetools

import adapter "github.com/iddqdeika/kafka-adapter"

func newCheckOrder(checkName, objectType, objectIdentifier string, msg *adapter.Message) CheckOrder {
	return &checkOrder{
		cn:        checkName,
		qm:        msg,
		ot:        objectType,
		oid:       objectIdentifier,
		result:    make(chan CheckResult),
		published: make(chan struct{}),
	}
}

type checkOrder struct {
	cn        string
	qm        *adapter.Message
	ot        string
	oid       string
	result    chan CheckResult
	published chan struct{}
}

func (o *checkOrder) ObjectType() string {
	return o.ot
}

func (o *checkOrder) Published() chan struct{} {
	return o.published
}

func (o *checkOrder) CheckName() string {
	return o.cn
}

func (o *checkOrder) ObjectIdentifier() string {
	return o.oid
}

func (o *checkOrder) Result() chan CheckResult {
	return o.result
}

func (o *checkOrder) Ack() error {
	return o.qm.Ack()
}

func (o *checkOrder) Nack() error {
	return o.qm.Nack()
}
