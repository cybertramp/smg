package tuimain

import (
	"testing"
	"time"
)

func TestAgo(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		ts   int64
		want string
	}{
		{0, ""},
		{now.Unix(), "just now"},
		{now.Add(-5 * time.Minute).Unix(), "5m ago"},
		{now.Add(-3 * time.Hour).Unix(), "3h ago"},
		{now.Add(-50 * time.Hour).Unix(), "2d ago"},
	} {
		if got := ago(tc.ts); got != tc.want {
			t.Errorf("ago(%d) = %q, want %q", tc.ts, got, tc.want)
		}
	}
}
