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
	// название файла конфига, который будет читаться рутом при регистрации.
	checkServiceJsonConfigFileName = "config.json"
)

// собирает корень композиции с данной функцией-обработчиком заказа на проверку.
// корень потом достаточно выполнить через rrr паттерн, чтобы получить готовое приложение.
func NewKafkaCheckServiceRoot(p CheckProviderFabric) (rrr.Root, error) {
	if p == nil {
		return nil, fmt.Errorf("must be not-nil CheckProvider")
	}
	return &kafkaCheckServiceRoot{
		p: p,
	}, nil
}

type kafkaCheckServiceRoot struct {
	l helpful.Logger
	p CheckProviderFabric
	s CheckService
}

// регистрация компонент.
// выбираем реализации и инстанциируем здесь.
func (r *kafkaCheckServiceRoot) Register() []error {

	// конфиг
	cfg, err := helpful.NewJsonCfg(checkServiceJsonConfigFileName)
	// если ошибка конфига, то дальше нет смысла идти
	if err != nil {
		return []error{err}
	}
	// логгер
	r.l = helpful.DefaultLogger.WithLevel(helpful.LogInfo)
	if cfg.Contains("log") {
		lvl, _ := cfg.Child("log").GetString("level")
		switch lvl {
		case "error":
			r.l = helpful.DefaultLogger.WithLevel(helpful.LogError)
		case "info":
		default:
			r.l = helpful.DefaultLogger.WithLevel(helpful.LogInfo)
		}
	}
	r.l.Infof("logger initialized")
	defer r.l.Infof("register finished")

	// сбор ошибок
	var errs []error
	e := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}

	// дергаем фабрику провайдера
	p, err := r.p.New(cfg.Child("check_logic_provider"), r.l)
	e(err)

	// конструктор сервиса
	r.s, err = NewKafkaCheckService(cfg, r.l, p)
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
