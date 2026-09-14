package project

import "testing"

func TestSprintDates(t *testing.T) {
	for _, tc := range []struct {
		start, end string
		days       int64
	}{
		{"2026-09-01", "2026-09-14", 14},
		{"2026-09-01", "2026-09-01", 1},
		{"2028-02-28", "2028-03-01", 3},
		{"2026-09-02", "2026-09-01", 0},
		{"", "2026-09-01", 0},
		{"2026-09-01", "", 0},
		{"2026-02-30", "2026-03-01", 0},
		{"0000-01-01", "2026-03-01", 0},
	} {
		first, last, err := sprintDates(tc.start, tc.end)
		if tc.days == 0 {
			if err == nil {
				t.Errorf("accepted invalid dates %q %q", tc.start, tc.end)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := (last.Time.Unix()-first.Time.Unix())/86400 + 1; got != tc.days {
			t.Errorf("duration=%d, want %d", got, tc.days)
		}
	}
}
