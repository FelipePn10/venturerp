package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Record is one audit-log row as returned to API consumers.
type Record struct {
	ID         int64     `json:"id"`
	OccurredAt time.Time `json:"occurred_at"`
	RequestID  string    `json:"request_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	UserRole   string    `json:"user_role,omitempty"`
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	Path       string    `json:"path"`
	Query      string    `json:"query,omitempty"`
	Status     int       `json:"status"`
	IP         string    `json:"ip,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	LatencyMS  int64     `json:"latency_ms"`
}

// Filter narrows an audit query. Zero values mean "no constraint".
type Filter struct {
	UserID string
	Route  string
	From   time.Time
	To     time.Time
	Limit  int
	Offset int
}

// Reader queries the audit trail. Kept separate from the write path so reads
// never share state with the async writer.
type Reader struct {
	pool *pgxpool.Pool
}

// NewReader builds an audit reader over the given pool.
func NewReader(pool *pgxpool.Pool) *Reader {
	return &Reader{pool: pool}
}

// List returns audit records matching f, newest first. Limit is clamped to
// [1, 500] with a default of 100.
func (r *Reader) List(ctx context.Context, f Filter) ([]Record, error) {
	conds := []string{}
	args := []any{}
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if f.UserID != "" {
		add("user_id = $%d", f.UserID)
	}
	if f.Route != "" {
		add("route = $%d", f.Route)
	}
	if !f.From.IsZero() {
		add("occurred_at >= $%d", f.From)
	}
	if !f.To.IsZero() {
		add("occurred_at <= $%d", f.To)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	q := fmt.Sprintf(`
		SELECT id, occurred_at, COALESCE(request_id,''), COALESCE(user_id,''), COALESCE(user_role,''),
		       method, route, path, COALESCE(query,''), status, COALESCE(ip,''), COALESCE(user_agent,''), latency_ms
		FROM public.audit_log
		%s
		ORDER BY occurred_at DESC, id DESC
		LIMIT %d OFFSET %d`, where, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Record, 0, limit)
	for rows.Next() {
		var rec Record
		if err := rows.Scan(
			&rec.ID, &rec.OccurredAt, &rec.RequestID, &rec.UserID, &rec.UserRole,
			&rec.Method, &rec.Route, &rec.Path, &rec.Query, &rec.Status, &rec.IP, &rec.UserAgent, &rec.LatencyMS,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
