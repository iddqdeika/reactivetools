package misc

import (
	"context"
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	"net/http"
	"strconv"
)

func NewEchoService(cfg helpful.Config, l helpful.Logger) (*echoService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil Config")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil Logger")
	}

	echoMethod, err := cfg.GetString("echo_method_name")
	if err != nil {
		return nil, err
	}
	echoPort, err := cfg.GetInt("echo_port")
	if err != nil {
		return nil, err
	}
	return &echoService{
		method: echoMethod,
		port:   echoPort,
	}, nil
}

type echoService struct {
	method string
	port   int
}

func (s *echoService) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.method, echo)
	ch := make(chan error)
	go func() {
		ch <- http.ListenAndServe(":"+strconv.Itoa(s.port), mux)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-ch:
		return err
	}
}

func echo(rw http.ResponseWriter, req *http.Request) {
	msgs := req.URL.Query()["msg"]
	if len(msgs) >= 1 {
		rw.Write([]byte(msgs[0]))
	}
	return
}
