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

	"github.com/imagvfx/forge"
)

// validateProperty validates a property with related infos.
// It saves the result to p.RawValue when it has processed well.
// It modifies p.Value to make it better form to log.
// Every type should allow empty Value while its meaning can be different on the type.
func validateProperty(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p == nil {
		return fmt.Errorf("unable to validate nil property")
	}
	// cleanup
	p.Value = strings.TrimSpace(p.Value)
	p.Value = strings.ReplaceAll(p.Value, "\r\n", "\n")
	// note that 'old' will be nil, if there is no previous value
	handled, err := validateSpecialProperty(tx, ctx, p, old)
	if handled {
		return err
	}
	validateFn := map[string]func(*sql.Tx, context.Context, *forge.Property, *forge.Property) error{
		"timecode":   validateTimecode,
		"text":       validateText,
		"user":       validateUser,
		"entry_path": validateEntryPath,
		"entry_name": validateEntryName,
		"entry_link": validateEntryLink,
		"date":       validateDate,
		"int":        validateInt,
		"tag":        validateTag,
		"search":     validateSearch,
	}
	validate := validateFn[p.Type]
	if validate == nil {
		return fmt.Errorf("unknown type of property: %v", p.Type)
	}
	return validate(tx, ctx, p, old)
}

// validateSpecialProperty validates special properties those Forge treats specially.
// For normal properties, it will return the input value unmodified.
func validateSpecialProperty(tx *sql.Tx, ctx context.Context, p, old *forge.Property) (bool, error) {
	switch p.Name {
	case ".predefined_sub_entries":
		err := func() error {
			subNameType := make(map[string]int)
			for _, nt := range strings.Split(p.Value, ",") {
				nt = strings.TrimSpace(nt)
				toks := strings.Split(nt, ":")
				if len(toks) != 2 {
					return fmt.Errorf(".predefined_sub_entries value should consists of 'subent:type' tokens: %v", nt)
				}
				sub := strings.TrimSpace(toks[0])
				typ := strings.TrimSpace(toks[1])
				// Save the type id, instead.
				id, err := getEntryTypeID(tx, ctx, typ)
				if err != nil {
					var e *forge.NotFoundError
					if !errors.As(err, &e) {
						return err
					}
					return fmt.Errorf("not found the entry type defined for '%v' in .predefined_sub_entries", typ)
				}
				subNameType[sub] = id
			}
			subNames := make([]string, 0, len(subNameType))
			for sub := range subNameType {
				subNames = append(subNames, sub)
			}
			sort.Slice(subNames, func(i, j int) bool { return subNames[i] < subNames[j] })
			val := ""
			for i, sub := range subNames {
				if i != 0 {
					val += ", "
				}
				val += sub + ":" + strconv.Itoa(subNameType[sub])
			}
			p.RawValue = val
			return nil
		}()
		return true, err
	}
	return false, nil
}

func validateText(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	user := forge.UserNameFromContext(ctx)
	setting, err := getUserSetting(tx, ctx, user)
	if err != nil {
		return err
	}
	remap := setting.CopyPathRemap
	// reverse the copy path mapping
	// maybe it's worth having better mapping mechanism.
	// but this is what I have now.
	toks := strings.Split(remap, ";")
	if len(toks) != 2 {
		p.RawValue = p.Value
		return nil
	}
	remapFrom := strings.TrimSpace(toks[0])
	remapTo := strings.TrimSpace(toks[1])
	if remapFrom == "" && remapTo == "" {
		p.RawValue = p.Value
		return nil
	}
	if remapFrom == "" || remapTo == "" {
		// only one of remapFrom,remapTo is defined
		return fmt.Errorf("user path mapping needs both from and to sides for now")
	}
	lines := strings.Split(p.Value, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, remapTo) {
			line = strings.Replace(line, remapTo, remapFrom, 1)
		}
		newLines = append(newLines, line)
	}
	p.Value = strings.Join(newLines, "\n")
	p.RawValue = p.Value
	return nil
}

func validateUser(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		p.RawValue = ""
		return nil
	}
	id, err := getUserID(tx, ctx, p.Value)
	if err != nil {
		return err
	}
	p.RawValue = strconv.Itoa(id)
	return nil
}

func validateTimecode(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		p.RawValue = ""
		return nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true,
	}
	tc := ""
	for _, r := range p.Value {
		ch := string(r)
		if isDigit[ch] {
			tc += ch
		}
	}
	if len(tc) != 8 {
		return fmt.Errorf("invalid timecode string: %v", p.Value)
	}
	// make the value a formal form of timecode. ex) 00:00:00:00
	p.Value = strings.Join(
		[]string{
			tc[0:2], tc[2:4], tc[4:6], tc[6:8],
		},
		":",
	)
	p.RawValue = p.Value
	return nil
}

func validateEntryPath(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	// It will save 'val' entry as it's id.
	if p.Value == "" {
		p.RawValue = ""
		return nil
	}
	if p.Value == "." {
		// "." indicates the entry itself.
		// Note that it doesn't require 'entry' path, so "." can be used in defaults.
		p.RawValue = "0"
		return nil
	}
	pth := p.Value
	if !path.IsAbs(p.Value) {
		// make abs path
		pth = path.Join(p.EntryPath, p.Value)
	}
	id, err := getEntryID(tx, ctx, pth)
	if err != nil {
		return err
	}
	p.RawValue = strconv.Itoa(id)
	return nil
}

// Entry name property accepts path of an entry and returns it's name.
func validateEntryName(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	return validateEntryPath(tx, ctx, p, old)
}

func validateEntryLink(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	have := make(map[string]bool)
	if old != nil {
		for _, pth := range strings.Split(old.RawValue, "\n") {
			if strings.TrimSpace(pth) == "" {
				continue
			}
			have[pth] = true
		}
	}
	val := ""
	lines := strings.Split(p.Value, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		op := line[0]
		valid := false
		for _, o := range "+-" {
			if rune(op) == o {
				valid = true
				continue
			}
		}
		if !valid {
			continue
		}
		pth := strings.TrimSpace(line[1:])
		if pth == "" {
			continue
		}
		switch op {
		case '+':
			// add
			val += "+" + pth + "\n"
			have[pth] = true
		case '-':
			// remove
			val += "-" + pth + "\n"
			delete(have, pth)
		}
	}
	// update p.Value so it only logs differences, not everything.
	p.Value = strings.TrimSpace(val)
	pths := make([]string, 0, len(have))
	for pth := range have {
		pths = append(pths, pth)
	}
	sort.Strings(pths)
	rawVal := strings.Join(pths, "\n")
	if rawVal != "" {
		// for line matching search in sqlite
		rawVal = "\n" + rawVal + "\n"
	}
	p.RawValue = rawVal
	return nil
}

func validateDate(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		p.RawValue = ""
		return nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[rune]bool{
		'0': true, '1': true, '2': true, '3': true, '4': true,
		'5': true, '6': true, '7': true, '8': true, '9': true,
	}
	// if the value starts with + or -, it will change the current date
	possiblePrefix := rune(p.Value[0])
	if possiblePrefix == '+' || possiblePrefix == '-' {
		day, err := strconv.Atoi(p.Value[1:])
		if err != nil {
			return fmt.Errorf("invalid date operation: +/- operation needs digits only, got: %v", p.Value[1:])
		}
		if possiblePrefix == '-' {
			day *= -1
		}
		if old.Value == "" {
			// TODO: would it better to use today instead?
			return fmt.Errorf("invalid date operation: +/- operation could be applied only non-empty date: %v", p.Value[1:])
		}
		t, err := time.Parse("2006/01/02", old.Value)
		if err != nil {
			return fmt.Errorf("invalid date string: %v", err)
		}
		t = t.AddDate(0, 0, day)
		val := t.Format("2006/01/02")
		p.RawValue = val
		return nil
	}
	// the value should be a date
	date := ""
	for _, r := range p.Value {
		if isDigit[r] {
			date += string(r)
		}
	}
	if len(date) != 8 {
		return fmt.Errorf("invalid date string: want yyyy/mm/dd, got %v", p.Value)
	}
	val := strings.Join(
		[]string{
			date[0:4], date[4:6], date[6:8],
		},
		"/",
	)
	_, err := time.Parse("2006/01/02", val)
	if err != nil {
		return fmt.Errorf("invalid date string: %v", err)
	}
	p.Value = val
	p.RawValue = p.Value
	return nil
}

func validateInt(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		p.RawValue = ""
		return nil
	}
	n, err := strconv.Atoi(p.Value)
	if err != nil {
		return fmt.Errorf("cannot convert to int: %v", p.Value)
	}
	p.Value = strconv.Itoa(n)
	p.RawValue = p.Value
	return nil
}

func validateTag(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	have := make(map[string]bool)
	if old != nil {
		for _, v := range strings.Split(old.RawValue, "\n") {
			if strings.TrimSpace(v) == "" {
				continue
			}
			have[v] = true
		}
	}
	val := ""
	lines := strings.Split(p.Value, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if len(ln) == 0 {
			continue
		}
		op := ln[0]
		v := strings.TrimSpace(ln[1:])
		if len(v) == 0 {
			continue
		}
		v = strings.TrimSpace(v)
		v = strings.ReplaceAll(v, " ", "_") // no space in tag for equal-search
		v = strings.ReplaceAll(v, "+", "_") // avoid confuse caused by op vs val
		v = strings.ReplaceAll(v, "-", "_") // avoid confuse caused by op vs val
		v = strings.ReplaceAll(v, ",", "_") // avoid comma in a tag for search like 'tag:a,b,c'
		switch op {
		case '+':
			// add
			val += "+" + v + "\n"
			have[v] = true
		case '-':
			// remove
			delete(have, v)
			val += "-" + v + "\n"
		}
	}
	// update p.Value so it only logs differences, not everything.
	p.Value = strings.TrimSpace(val)
	newlines := make([]string, 0, len(have))
	for v := range have {
		newlines = append(newlines, v)
	}
	sort.Strings(newlines)
	rawVal := strings.Join(newlines, "\n")
	if rawVal != "" {
		// for line matching search in sqlite
		rawVal = "\n" + rawVal + "\n"
	}
	p.RawValue = rawVal
	return nil
}

func validateSearch(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	// search can have multiple search queries.
	// part before '|' is name of a search query, after it is the query.
	//
	// ex)
	// 	shots|type=shot
	// 	assets|type=asset
	// 	scene1 shots|type=shot path:/shot/scene1
	// 	environ assets|type=asset path:/asset/environ
	lines := strings.Split(p.Value, "\n")
	newlines := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			// allow empty line
			newlines = append(newlines, ln)
			continue
		}
		name, query, ok := strings.Cut(ln, "|")
		if !ok {
			return fmt.Errorf("search should be name|query form: got %s", ln)
		}
		name = strings.TrimSpace(name)
		query = strings.TrimSpace(query)
		if name == "" {
			return fmt.Errorf("search name shouldn't be empty: got %s", ln)
		}
		if query == "" {
			return fmt.Errorf("search query shouldn't be empty: got %s", ln)
		}
		newlines = append(newlines, name+"|"+query)
	}
	p.Value = strings.Join(newlines, "\n")
	p.RawValue = p.Value
	return nil
}
