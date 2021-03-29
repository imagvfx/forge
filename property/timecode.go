package property

import (
	"fmt"
	"strconv"
	"strings"
)

type Timecode struct {
	name  string
	value string
}

func NewTimecode(name string) *Timecode {
	t := &Timecode{
		name: name,
	}
	return t
}

func (t *Timecode) Name() string {
	return t.name
}

func (t *Timecode) Value() string {
	return t.value
}

func (t *Timecode) Set(s string) error {
	err := validateTimecode(s)
	if err != nil {
		return err
	}
	t.t = s
	return nil
}

func validateTimecode(s string) error {
	// 00:00:00:00
	if s == "" {
		// unset
		return nil
	}
	toks := strings.Split(s, ":")
	if len(toks) != 4 {
		return fmt.Errorf("invalid timecode string: %v", s)
	}
	for _, t := range toks {
		i, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("invalid timecode string: %v", s)
		}
		if i < 0 || i > 100 {
			return fmt.Errorf("invalid timecode string: %v", s)
		}
	}
	return nil
}
