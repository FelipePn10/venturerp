// Package audit provides an append-only audit trail for mutating API requests.
// Events are captured at the HTTP layer (by the Audit middleware) and written
// asynchronously to the audit_log table, so recording never sits on the request
// path or fails a business operation.
package audit

import (
	"context"
	"sync"
	"time"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is a single audited action: who did what, when, and with what outcome.
type Event struct {
	OccurredAt time.Time
	RequestID  string
	UserID     string
	UserRole   string
	Method     string
	Route      string
	Path       string
	Query      string
	Status     int
	IP         string
	UserAgent  string
	LatencyMS  int64
}

// Sink receives audit events. Implementations must be non-blocking and safe for
// concurrent use; the HTTP layer depends only on this interface.
type Sink interface {
	Record(Event)
}

// PgSink persists events to PostgreSQL via a background worker. A bounded buffer
// decouples request handling from database writes; if the buffer is full (DB
// slow/unavailable) events are dropped with a warning rather than blocking
// traffic — auditing must never take the application down.
type PgSink struct {
	pool   *pgxpool.Pool
	log    *applogger.Logger
	events chan Event
	wg     sync.WaitGroup
	once   sync.Once
}

const auditBufferSize = 2048

// NewPgSink starts the background writer and returns the sink. Call Close on
// shutdown to flush buffered events.
func NewPgSink(pool *pgxpool.Pool, log *applogger.Logger) *PgSink {
	s := &PgSink{
		pool:   pool,
		log:    log,
		events: make(chan Event, auditBufferSize),
	}
	s.wg.Add(1)
	go s.run()
	return s
}

// Record enqueues an event. Never blocks: a full buffer drops the event.
func (s *PgSink) Record(e Event) {
	select {
	case s.events <- e:
	default:
		s.log.Warn("audit buffer full, dropping event", "route", e.Route, "method", e.Method)
	}
}

func (s *PgSink) run() {
	defer s.wg.Done()
	for e := range s.events {
		s.insert(e)
	}
}

func (s *PgSink) insert(e Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const q = `
		INSERT INTO public.audit_log
			(occurred_at, request_id, user_id, user_role, method, route, path, query, status, ip, user_agent, latency_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	if _, err := s.pool.Exec(ctx, q,
		e.OccurredAt, nullable(e.RequestID), nullable(e.UserID), nullable(e.UserRole),
		e.Method, e.Route, e.Path, nullable(e.Query), e.Status, nullable(e.IP),
		nullable(e.UserAgent), e.LatencyMS,
	); err != nil {
		s.log.Error("audit insert failed", "error", err, "route", e.Route)
	}
}

// Close stops accepting events and waits for the buffer to drain. Idempotent.
func (s *PgSink) Close() {
	s.once.Do(func() {
		close(s.events)
		s.wg.Wait()
	})
}

// nullable maps an empty string to a SQL NULL so optional columns stay clean.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
