package sqlite

import (
	"log"
	"testing"
	"time"
)

func TestParseTimeScan(t *testing.T) {
	// sqlite uses variable length of string for micro seconds.
	// Time.Scan should convert these string into time without having error.
	cases := []struct {
		s interface{}
	}{
		{s: "0000-01-01 00:00:00.123456789+00:00"},
		{s: "0000-01-01 00:00:00.123456+00:00"},
		{s: "0000-01-01 00:00:00.123+00:00"},
		{s: "0000-01-01 00:00:00+00:00"},
		{s: time.Now()},
	}
	for _, c := range cases {
		t := Time{}
		err := t.Scan(c.s)
		if err != nil {
			log.Fatal(err)
		}
	}
}
