package main

import (
	"os"
	"testing"
	"time"
)

func TestGetDurationFromEnv(t *testing.T) {
	const defaultPollInterval = 120 * time.Second
	defer os.Unsetenv("POLL_INTERVAL_SEC")

	for _, test := range []struct {
		have string
		want time.Duration
	}{
		{
			have: "",
			want: defaultPollInterval,
		},
		{
			have: "1000",
			want: time.Second * time.Duration(1000),
		},
	} {
		if test.have != "" {
			os.Setenv("POLL_INTERVAL_SEC", test.have)
		}
		got, err := getDurationFromEnv("POLL_INTERVAL_SEC", defaultPollInterval)
		if err != nil {
			t.Errorf("error getting poll interval: %s", err)
		}
		if got != test.want {
			t.Errorf("poller interval: wanted %s got %s", test.want, got)
		}
	}
}
