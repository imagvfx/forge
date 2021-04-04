package property

import (
	"fmt"
	"strconv"
	"strings"
)

func Validate(typ, val string) error {
	validate := map[string]func(string) error{
		"timecode": validateTimecode,
		"text":     validateText,
		"user":     validateUser,
	}
	fn := validate[typ]
	if fn == nil {
		return fmt.Errorf("unknown type of property: %v", typ)
	}
	return fn(val)
}

func validateText(s string) error {
	// every string is valid text
	return nil
}

func validateUser(s string) error {
	// TODO: validate when User is implemented
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
