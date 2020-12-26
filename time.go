package funcmaps

import "time"

// This file copied from https://github.com/pthethanh/template/blob/master/time.go

// FormatTime format the given date
//
// Date can be a `time.Time` or an `int, int32, int64`.
// In the later case, it is treated as seconds since UNIX
// epoch.
func FormatTime(fmt string, zone string, date interface{}) string {
	if zone == "" {
		zone = "Local"
	}
	return formateDate(fmt, date, zone)
}

func formateDate(fmt string, date interface{}, zone string) string {
	var t time.Time
	switch date := date.(type) {
	default:
		t = time.Now()
	case time.Time:
		t = date
	case *time.Time:
		t = *date
	case int64:
		t = time.Unix(date, 0)
	case int:
		t = time.Unix(int64(date), 0)
	case int32:
		t = time.Unix(int64(date), 0)
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc, _ = time.LoadLocation("UTC")
	}

	return t.In(loc).Format(fmt)
}
