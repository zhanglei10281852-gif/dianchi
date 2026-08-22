package clock

import "time"

type Window struct {
	Start, End time.Time
	Location   *time.Location
}

func NewWindow(start, end time.Time, loc *time.Location) Window {
	if loc == nil {
		loc = time.UTC
	}
	return Window{Start: start.In(loc), End: end.In(loc), Location: loc}
}
func (w Window) Contains(t time.Time) bool {
	t = t.In(w.Location)
	return !t.Before(w.Start) && t.Before(w.End)
}
func (w Window) Duration() time.Duration    { return w.End.Sub(w.Start) }
func (w Window) Expired(now time.Time) bool { return !now.In(w.Location).Before(w.End) }
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	v := t.In(loc)
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, loc)
}
func EndOfDay(t time.Time, loc *time.Location) time.Time {
	return StartOfDay(t, loc).Add(24 * time.Hour)
}
