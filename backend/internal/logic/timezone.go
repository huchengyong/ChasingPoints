package logic

import "time"

const UTC8Layout = "2006-01-02 15:04:05"

var UTC8Location = time.FixedZone("UTC+8", 8*60*60)

func NowUTC8() time.Time {
	return time.Now().In(UTC8Location)
}

func InUTC8(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(UTC8Location)
}

func FormatUTC8Time(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return InUTC8(t).Format(UTC8Layout)
}

func FormatUTC8TimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return FormatUTC8Time(*t)
}
