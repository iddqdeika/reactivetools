package reactivetools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	"net/http"
	"strconv"
)

const (
	portConfigName   = "port"
	minPort          = 8000
	echoMethod       = "echo"
	statisticsMethod = "statistics"
)

// конструктор сервиса статистики
func NewStatisticService(config helpful.Config, sp StatisticProvider, l helpful.Logger) (Service, error) {
	if config == nil {
		return nil, fmt.Errorf("must be not-nil Config")
	}
	if sp == nil {
		return nil, fmt.Errorf("must be not-nil CheckOrderProvider")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil Logger")
	}

	port, err := config.GetInt(portConfigName)
	if err != nil {
		return nil, err
	}

	if port < minPort {
		return nil, fmt.Errorf("port must be above %v", minPort)
	}
	s := &statisticService{
		port: port,
		p:    sp,
		l:    l,
	}
	return s, nil
}

// предоставляет http метод для получения статистик и эхо метод
type statisticService struct {
	port int
	p    StatisticProvider
	l    helpful.Logger
}

func (s *statisticService) Run(ctx context.Context) error {
	sm := http.NewServeMux()
	sm.HandleFunc("/"+statisticsMethod, s.statisticHandler)
	s.l.Infof("%v registered in statisticservice", statisticsMethod, strconv.Itoa(s.port))
	sm.HandleFunc("/"+echoMethod, echo)
	s.l.Infof("%v registered in statisticservice", echoMethod, strconv.Itoa(s.port))
	ctx, cancel := context.WithCancel(ctx)

	var err error
	s.l.Infof("statisticservice started on port %v", strconv.Itoa(s.port))
	go func() {
		err = http.ListenAndServe(":"+strconv.Itoa(s.port), sm)
		cancel()
	}()
	<-ctx.Done()
	return err
}

func (s *statisticService) statisticHandler(w http.ResponseWriter, req *http.Request) {
	ss, err := s.p.Statistics()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Server side error: %v", err)))
		return
	}
	res := &statisticsDTO{Statistics: make([]statisticDTO, 0)}
	for _, s := range ss {
		res.Statistics = append(res.Statistics, statisticDTO{
			Name:        s.Name(),
			Value:       s.Value(),
			Description: s.Description(),
		})
	}
	data, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Server side error: %v", err)))
		return
	}

	_, err = w.Write(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.l.Errorf("err during response writing in statistic service: %v", err)
		return
	}
}

func echo(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(req.URL.RawQuery))
}

type statisticsDTO struct {
	Statistics []statisticDTO `json:"statistics"`
}

type statisticDTO struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}
