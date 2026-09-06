package ingest

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

type callbackTiming struct {
	count   atomic.Int64
	total   atomic.Int64
	max     atomic.Int64
	started atomic.Int64
}

func (t *callbackTiming) record(start time.Time) {
	d := time.Since(start).Nanoseconds()
	t.count.Add(1)
	t.total.Add(d)
	for old := t.max.Load(); d > old; old = t.max.Load() {
		if t.max.CompareAndSwap(old, d) {
			break
		}
	}
	t.started.Store(0)
}

func (w *Worker) logCallbackTiming(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count := w.timing.count.Load()
			var mean, active time.Duration
			if count > 0 {
				mean = time.Duration(w.timing.total.Load() / count)
			}
			if start := w.timing.started.Load(); start != 0 {
				active = time.Since(time.Unix(0, start))
			}
			log.Printf("mqtt_timing broker=%s count=%d mean_us=%d max_us=%d active_ms=%d", w.cfg.BrokerName, count, mean.Microseconds(), time.Duration(w.timing.max.Load()).Microseconds(), active.Milliseconds())
		}
	}
}
