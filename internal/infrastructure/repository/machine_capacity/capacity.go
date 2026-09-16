package machine_capacity

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

// Hours returns actual calendar hours. Output quantity and performance efficiency
// never reduce this figure: performance is applied exactly once to processing time.
func Hours(ctx context.Context, pool *pgxpool.Pool, workCenterID int64, day *time.Time) (float64, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var h float64
	if day == nil {
		err = pool.QueryRow(ctx, `SELECT COALESCE(SUM(COALESCE(m.available_hours_per_day,mt.capacity_hours)),0)
  FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.enterprise_id=m.enterprise_id
  WHERE mt.id=$1 AND m.enterprise_id=$2 AND m.is_active AND mt.is_active`, workCenterID, e).Scan(&h)
		return h, err
	}
	// range_agg merges overlapping shifts and downtimes instead of counting twice.
	err = pool.QueryRow(ctx, `WITH machine_windows AS (
 SELECT m.id,CASE WHEN m.calendar_id IS NULL THEN
  CASE WHEN EXTRACT(ISODOW FROM $3::date)<6 THEN tsmultirange(tsrange($3::date::timestamp,$3::date::timestamp+make_interval(secs=>(COALESCE(m.available_hours_per_day,mt.capacity_hours)*3600)::double precision),'[)')) ELSE '{}'::tsmultirange END
 ELSE COALESCE((SELECT range_agg(tsrange($3::date+i.start_time,$3::date+i.end_time+CASE WHEN i.end_time<=i.start_time THEN INTERVAL '1 day' ELSE INTERVAL '0' END,'[)')) FROM machine_calendar_intervals i JOIN machine_calendars c ON c.id=i.calendar_id AND c.enterprise_id=$2 WHERE i.calendar_id=m.calendar_id AND i.weekday=EXTRACT(DOW FROM $3::date)), '{}'::tsmultirange) END AS windows
 FROM machines m JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.enterprise_id=m.enterprise_id
 WHERE mt.id=$1 AND m.enterprise_id=$2 AND m.is_active AND mt.is_active
 ), remaining AS (
 SELECT w.id,CASE WHEN EXISTS(SELECT 1 FROM maintenance_orders o LEFT JOIN maintenance_plans p ON p.id=o.plan_id WHERE COALESCE(o.machine_id,p.machine_id)=w.id AND o.scheduled_date=$3::date AND o.is_active AND o.status IN ('PLANNED','IN_PROGRESS')) THEN '{}'::tsmultirange ELSE w.windows-COALESCE((SELECT range_agg(tsrange(d.starts_at::timestamp,d.ends_at::timestamp,'[)')) FROM machine_downtimes d WHERE d.machine_id=w.id AND d.enterprise_id=$2 AND d.starts_at<$3::date+INTERVAL '2 days' AND d.ends_at>$3::date),'{}'::tsmultirange) END AS windows FROM machine_windows w
 ), hours AS (SELECT r.id,COALESCE(SUM(EXTRACT(EPOCH FROM upper(span)-lower(span))/3600),0) AS h FROM remaining r LEFT JOIN LATERAL unnest(r.windows) span ON TRUE GROUP BY r.id)
 SELECT COALESCE(SUM(h),0) FROM hours`, workCenterID, e, *day).Scan(&h)
	return h, err
}
