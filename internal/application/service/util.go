package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseUint converts the string form of an id (as carried in a PASETO
// payload or a route parameter) back into the uint primary key used by
// the repository layer. Used for both user ids and company ids.
func parseUint(s string) (uint, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func timeToPrettyFormat(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < 0 {
		d = -d
	}

	days := d / (24 * time.Hour)
	d %= 24 * time.Hour

	hours := d / time.Hour
	d %= time.Hour

	minutes := d / time.Minute
	d %= time.Minute

	seconds := d / time.Second
	d %= time.Second

	// millis := d / time.Millisecond

	var result string
	if days > 0 {
		result += fmt.Sprintf("%dd ", days)
	}
	if hours > 0 {
		result += fmt.Sprintf("%dh ", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dm ", minutes)
	}
	if seconds > 0 {
		result += fmt.Sprintf("%ds ", seconds)
	}
	// if millis > 0 && days == 0 && hours == 0 {
	// 	result += fmt.Sprintf("%dms", millis)
	// }

	return strings.TrimSpace(result)
}
