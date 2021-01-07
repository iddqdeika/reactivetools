package reactivetools

import (
	"context"
	"errors"
	"fmt"
	"github.com/iddqdeika/rrr"
	"github.com/iddqdeika/rrr/helpful"
	"strings"
)

const (
	checkServiceJsonConfigFootprintFileName = "config_footprint.json"
	checkServiceJsonConfigFileName          = "config_template.json"
)

// собирает корень композиции с данной функцией-обработчиком заказа на проверку.
// корень потом достаточно выполнить через rrr паттерн, чтобы получить готовое приложение.
func NewKafkaCheckServiceRoot(f CheckFunc) (rrr.Root, error) {
	if f == nil {
		return nil, fmt.Errorf("must be not-nil CheckFunc")
	}
	return &kafkaCheckServiceRoot{
		f: f,
	}, nil
}

type kafkaCheckServiceRoot struct {
	l helpful.Logger
	f CheckFunc
	s CheckService
}

// регистрация компонент.
// выбираем реализации и инстанциируем здесь.
func (r *kafkaCheckServiceRoot) Register() []error {
	// логгер
	r.l = helpful.DefaultLogger.WithLevel(helpful.LogInfo)
	r.l.Infof("logger initialized")
	defer r.l.Infof("register finished")

	// конфиг
	cfg, err := helpful.NewJsonCfg(checkServiceJsonConfigFileName)
	// если ошибка конфига, то дальше нет смысла идти
	if err != nil {
		return []error{err}
	}

	// сбор ошибок
	var errs []error
	e := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	// конструктор сервиса
	r.s, err = NewKafkaCheckService(cfg, r.l, r.f)
	e(err)

	return errs
}

// исполнение логики приложения с помощью компонент, определенных на этапе Register
func (r *kafkaCheckServiceRoot) Resolve(ctx context.Context) error {
	r.l.Infof("root resolve started")
	defer r.l.Infof("root resolve finished")

	//исполняем
	return r.s.Run(ctx)
}

// высвобождаем ресурсы перед завершением работы, если надо
func (r *kafkaCheckServiceRoot) Release() error {
	return nil
}

//собрать список ошибок в одну.
func composeErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	errStrings := make([]string, len(errs))
	for i, e := range errs {
		errStrings[i] = e.Error()
	}
	return errors.New(strings.Join(errStrings, "; \r\n"))
}
