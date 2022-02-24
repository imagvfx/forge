package sqlite

import (
	"time"
)

type Time time.Time

func (t *Time) Scan(v interface{}) error {
	vt, err := time.Parse("2006-01-02 15:04:05.000000000-07:00", v.(string))
	if err != nil {
		return err
	}
	*t = Time(vt)
	return nil
}
