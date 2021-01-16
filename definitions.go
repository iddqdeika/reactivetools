package reactivetools

import (
	"context"
	"io"
)

// сервис проверки.
// запускается с контекстом, завершается без ошибки если контекст закрыт или закончились данные для обработки
type CheckService interface {
	Run(ctx context.Context) error
}

// заказ на проверку.
// когда надо проверить нечто - то надо знать его тип, идентификатор, имя необходимой проверки.
// также позволяет дожидаться результата проверки через канал Result()
// осторожно, в стандартной реализации Result возвращает результат лишь однократно!
// после окончания работы с заказом, когда необходимые действия выполнены - необходимо его подтвердить или отклонить.
// отклонение использовать нежелательно, т.к. изначально концепция ориентирована на кафка.
type CheckOrder interface {
	ObjectType() string
	ObjectIdentifier() string
	CheckName() string
	Result() chan CheckResult
	Published() chan struct{}
	Ack() error
	Nack() error
}

// результат проверки. содержит всю информацию о проверке и результате исполнения
// обычно собирается из результатов исполнения функции-процессора (CheckProvider)
// CheckSuccess должен возвращать ложь(false) только в том случае, если сама проверка выполнена, но не пройдена.
// если во время проверки возникла ошибка, то результат нужно публиковать только в случае,
// если на этом обработку закака можно завершить. как правило необходимо повторять её заново до тех пор,
// пока она не будет завершена корректно.
// например: если внешний ресурс, необходимый для проверки, недоступен -
// то после таймаута не стоит возвращать результат. вместо этого функция-процессор должна возвращать ошибку(err)
type CheckResult interface {
	ObjectType() string
	ObjectIdentifier() string
	CheckName() string
	ResultMessage() string
	CheckSuccess() bool
}

// функция для обрабтки заказов на проверку
// важно, чтобы она нормально работала с контекстом и завершалась при его закрытии.
// функция должна быть конкурентно-безопасна.
// success должен принимать значение ложь(false) только в том случае, если сама проверка выполнена, но не пройдена.
// если во время проверки возникла ошибка, то результат нужно публиковать только в случае,
// если на этом обработку заказа можно завершить. иногда необходимо повторять её заново до тех пор,
// пока она не будет завершена корректно.
// например: если внешний ресурс, необходимый для проверки, недоступен -
// то после таймаута не стоит возвращать результат. вместо этого функция-процессор должна возвращать ошибку(err)
type CheckProvider interface {
	PerformCheck(ctx context.Context, o CheckOrder) (msg string, success bool, err error)
}

// провайдер заказов на проверку.
// предоставляет канал, из которого можно забирать поступающие заказы на проверку
type CheckOrderProvider interface {
	OrderChan() chan CheckOrder
	StatisticProvider
}

// процессор проверок
// собственно и содержит бизнес-логику проверки.
// по результатам выполненной проверки кладет результат в канал Result самого заказа
// на случай долгих поцессов во время проверки должен следить за закрытием контекста.
// в случае невозможности завершить проверку (недоступность внешних ресурсов, например) - возвращает ошибку
type CheckOrderProcessor interface {
	Process(ctx context.Context, o CheckOrder) error
}

// публикатор результатов проверки
// передает резултаты проверки куда надо (например, публикует в соотв. очередь)
type CheckResultPublisher interface {
	PublishCheckResult(r CheckResult) error
}

// провайдер статистик
// интерфейс, который может вернуть разнообразные статистики.
// например, стандартная реализация CheckOrderProvider предоставляет данные об очереди заказов на проверку
type StatisticProvider interface {
	Statistics() ([]Statistic, error)
}

// статистика
// прредставляет единицу данных, отображающих статус/метрику некоего процесса/объекта
type Statistic interface {
	Name() string
	Value() string
	Description() string
}

// сервис для получения и обработки изменений
type ChangesConsumerService interface {
	Run(ctx context.Context) error
}

// предоставляет канал изменений, начитывая его, например, из кафка
type ChangesProvider interface {
	ChangesChan() chan ChangeEvent
}

// отвечает за обработку изменений
type ChangesProcessor interface {
	Process(event ChangeEvent) error
}

// аггрегатор изменений
// собирает в себе изменения значений и хранит их в виде ключ-значение
// любую логику завершения работы и gracefull shutdown нужно реализовывать в методе Close
type ChangesAggregator interface {
	ChangesProcessor
	KeyValStorage
	io.Closer
}

// аггрегатор данных ключ-значение
// любая реализация должна быть конкурентно-безопасной
type KeyValStorage interface {
	Set(key string, val string) error
	Get(key string) (string, error)
}

// объект, описывающий изменение
type ChangeEvent interface {
	ObjectType() string
	ObjectIdentifier() string
	EventName() string
	Data() string
	Ack() error
	Nack() error
	Processed() chan struct{}
}
