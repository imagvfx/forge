package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/imagvfx/forge/service"
)

func validateProperty(tx *sql.Tx, ctx context.Context, entry, typ, val string) (string, error) {
	validateFn := map[string]func(*sql.Tx, context.Context, string, string) (string, error){
		"timecode":   validateTimecode,
		"text":       validateText,
		"user":       validateUser,
		"entry_path": validateEntryPath,
		"entry_name": validateEntryName,
		"date":       validateDate,
		"int":        validateInt,
	}
	validate := validateFn[typ]
	if validate == nil {
		return "", fmt.Errorf("unknown type of property: %v", typ)
	}
	var err error
	val, err = validate(tx, ctx, entry, val)
	if err != nil {
		return "", err
	}
	return val, nil
}

func validateText(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	return val, nil
}

func validateUser(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	if val == "" {
		return "", nil
	}
	id, err := getUserID(tx, ctx, val)
	if err != nil {
		return "", err
	}
	val = strconv.Itoa(id)
	return val, nil
}

func validateTimecode(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	// 00:00:00:00
	if val == "" {
		// unset
		return "", nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true,
	}
	tc := ""
	for _, r := range val {
		ch := string(r)
		if isDigit[ch] {
			tc += ch
		}
	}
	if len(tc) > 8 {
		return "", fmt.Errorf("invalid timecode string: %v", val)
	}
	extra := 8 - len(tc)
	extraZero := strings.Repeat("0", extra)
	tc = extraZero + tc
	val = strings.Join(
		[]string{
			tc[0:2], tc[2:4], tc[4:6], tc[6:8],
		},
		":",
	)
	return val, nil
}

func validateEntryPath(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	// It will save 'val' entry as it's id.
	if val == "" {
		// unset
		return "", nil
	}
	if !path.IsAbs(val) {
		// make abs path
		val = path.Join(entry, val)
	}
	id, err := getEntryID(tx, ctx, val)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(id), nil
}

// Entry name property accepts path of an entry and returns it's name.
func validateEntryName(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	// It will save 'val' entry as it's id.
	// So validation process is same with 'validateEntryPath'.
	// Difference comes from evaluation.
	return validateEntryPath(tx, ctx, entry, val)
}

func validateDate(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	if val == "" {
		// unset
		return "", nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true,
	}
	date := ""
	for _, r := range val {
		ch := string(r)
		if isDigit[ch] {
			date += ch
		}
	}
	if len(date) != 8 {
		return "", fmt.Errorf("invalid date string: want yyyy/mm/dd, got %v", val)
	}
	val = strings.Join(
		[]string{
			date[0:4], date[4:6], date[6:8],
		},
		"/",
	)
	_, err := time.Parse("2006/01/02", val)
	if err != nil {
		return "", fmt.Errorf("invalid date string: %v", err)
	}
	return val, nil
}

func validateInt(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	if val == "" {
		// unset
		return "", nil
	}
	_, err := strconv.Atoi(val)
	if err != nil {
		return "", fmt.Errorf("cannot convert to int: %v", val)
	}
	return val, nil
}

// validateSpecialProperty validates special properties that starts with dot (.).
// For normal properties, it will return the input value unmodified.
func validateSpecialProperty(tx *sql.Tx, ctx context.Context, name, val string) (string, error) {
	// TODO: .sub_entry_types
	switch name {
	case ".predefined_sub_entries":
		subNameType := make(map[string]int)
		for _, nt := range strings.Split(val, ",") {
			nt = strings.TrimSpace(nt)
			toks := strings.Split(nt, ":")
			if len(toks) != 2 {
				return "", fmt.Errorf(".predefined_sub_entries value should consists of 'subent:type' tokens: %v", nt)
			}
			sub := strings.TrimSpace(toks[0])
			typ := strings.TrimSpace(toks[1])
			// Save the type id, instead.
			id, err := getEntryTypeID(tx, ctx, typ)
			if err != nil {
				var e *service.NotFoundError
				if !errors.As(err, &e) {
					return "", err
				}
				return "", fmt.Errorf("not found the entry type defined for '%v' in .predefined_sub_entries: %v", name, typ)
			}
			subNameType[sub] = id
		}
		subNames := make([]string, 0, len(subNameType))
		for sub := range subNameType {
			subNames = append(subNames, sub)
		}
		sort.Slice(subNames, func(i, j int) bool { return subNames[i] < subNames[j] })
		val = ""
		for i, sub := range subNames {
			if i != 0 {
				val += ", "
			}
			val += sub + ":" + strconv.Itoa(subNameType[sub])
		}
		return val, nil
	}
	return val, nil
}
