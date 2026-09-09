// Temporary aggregate-only diagnostics for issue #116. No SQL values are logged.
package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryTimingTracer struct{}
type queryTimingKey struct{}
type queryTiming struct {
	start time.Time
	name  string
}

func diagnosticQueryName(sql string) string {
	fields := strings.Fields(strings.SplitN(sql, "\n", 2)[0])
	if len(fields) < 3 || fields[0] != "--" || fields[1] != "name:" {
		return "unnamed"
	}
	for _, r := range fields[2] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return "unnamed"
		}
	}
	return fields[2]
}

func (queryTimingTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryTimingKey{}, queryTiming{time.Now(), diagnosticQueryName(data.SQL)})
}

func (queryTimingTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	timing, ok := ctx.Value(queryTimingKey{}).(queryTiming)
	if !ok {
		return
	}
	duration := time.Since(timing.start)
	if duration < 500*time.Millisecond && data.Err == nil {
		return
	}
	code := "none"
	var pgError *pgconn.PgError
	if errors.As(data.Err, &pgError) {
		code = pgError.Code
	}
	log.Printf("db_timing operation=%s elapsed_us=%d failed=%t sqlstate=%s", timing.name, duration.Microseconds(), data.Err != nil, code)
}

func logPoolTiming(ctx context.Context, pool *pgxpool.Pool) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s := pool.Stat()
			log.Printf("pool_timing acquires=%d acquire_us=%d waits=%d wait_us=%d canceled=%d acquired=%d total=%d", s.AcquireCount(), s.AcquireDuration().Microseconds(), s.EmptyAcquireCount(), s.EmptyAcquireWaitTime().Microseconds(), s.CanceledAcquireCount(), s.AcquiredConns(), s.TotalConns())
		}
	}
}
