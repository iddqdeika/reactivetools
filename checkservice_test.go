package reactivetools

import (
	"context"
	"github.com/iddqdeika/rrr/helpful"
	"testing"
	"time"
)

func TestNewCheckService(t *testing.T) {
	logger := helpful.DefaultLogger.WithLevel(helpful.LogNone)

	cfg, err := helpful.NewJsonCfg("config/check_service_cfg_test.json")
	if err != nil {
		t.Fatalf("cant create config for test: %v", err)
	}

	provider := NewStubOrderProvider(context.Background(), 5, time.Second)

	processor, err := NewStubCheckOrderProcessor()
	if err != nil {
		t.Fatalf("cant create stub check order processor: %v", err)
	}

	publisher := NewStubResultPublisher()

	checkService, err := NewCheckService(cfg, logger, provider, processor, publisher, nil)
	if err != nil {
		t.Fatalf("cant create checkService: %v", err)
	}

	err = checkService.Run(context.Background())
	if err != nil {
		t.Fatalf("checkService Run() method returned err: %v", err)
	}
}
