package sales_forecast_uc

import (
	"fmt"
	"time"
)

// weekToDate returns a time.Time representing the Monday of the given ISO week.
func weekToDate(year, week int) (time.Time, error) {
	if week < 1 || week > 53 {
		return time.Time{}, fmt.Errorf("week %d is out of range [1, 53]", week)
	}
	// Jan 4 is always in week 1 per ISO 8601.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	// Find the Monday of that week.
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, 1-weekday)
	// Add (week-1) weeks to get the target Monday.
	result := monday.AddDate(0, 0, (week-1)*7)
	return result, nil
}
