package statistic

import (
	"fmt"
	"sync"
	"time"
)

func newCache(timeout time.Duration) (*statisticCache, error) {
	if timeout == 0 {
		return nil, fmt.Errorf("timeout must be greater than 0")
	}
	return &statisticCache{
		timeout: timeout,
		cache:   make(map[string]cachedStatistic),
	}, nil
}

type statisticCache struct {
	timeout time.Duration
	cache   map[string]cachedStatistic
	sync.Mutex
}

func (c *statisticCache) set(stat Statistic) {
	if stat == nil {
		return
	}
	c.Lock()
	defer c.Unlock()
	c.cache[stat.Name()] = cachedStatistic{
		s:        stat,
		deadLine: time.Now().Add(c.timeout),
	}
}

func (c *statisticCache) getAll() []Statistic {
	c.Lock()
	defer c.Unlock()
	n := time.Now()
	res := make([]Statistic, 0)
	for _, record := range c.cache {
		if record.deadLine.Before(n) {
			res = append(res, record.s)
		}
	}
	return res
}

type cachedStatistic struct {
	s        Statistic
	deadLine time.Time
}
