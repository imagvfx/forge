package sqlite

import (
	"time"
)

type Time time.Time

func (t *Time) Scan(i interface{}) error {
	switch v := i.(type) {
	case time.Time:
		*t = Time(v)
	case string:
		vt, err := time.Parse("2006-01-02 15:04:05.9-07:00", v)
		if err != nil {
			return err
		}
		*t = Time(vt)
	}
	return nil
}
