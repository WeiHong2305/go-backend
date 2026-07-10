package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

type Config struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	MaxRetries int
}

func (c Config) Delay(attempt int) time.Duration {
	exp := c.BaseDelay << attempt
	if exp > c.MaxDelay || exp <= 0 {
		exp = c.MaxDelay
	}
	return time.Duration(rand.Int64N(int64(exp)))
}

func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = time.Second
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = 30 * time.Second
	}

	var err error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		err = fn(ctx)
		if err == nil {
			return nil
		}
		if attempt == cfg.MaxRetries {
			break
		}
		delay := cfg.Delay(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}
