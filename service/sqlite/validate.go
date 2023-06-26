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
func validateProperty(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p == nil {
		return fmt.Errorf("unable to validate nil property")
	}
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
	val := strings.ReplaceAll(p.Value, "\r\n", "\n")
	user := forge.UserNameFromContext(ctx)
	setting, err := getUserSetting(tx, ctx, user)
	if err != nil {
		return err
	}
	remap := setting.CopyPathRemap
	toks := strings.Split(remap, ";")
	if len(toks) != 2 {
		p.RawValue = val
		return nil
	}
	remapFrom := strings.TrimSpace(toks[0])
	remapTo := strings.TrimSpace(toks[1])
	if remapFrom == "" && remapTo == "" {
		p.RawValue = val
		return nil
	}
	// reverse the copy path mapping
	// maybe it's worth having better mapping mechanism.
	// but this is what I have now.
	lines := strings.Split(val, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, remapTo) {
			line = strings.Replace(line, remapTo, remapFrom, 1)
		}
		newLines = append(newLines, line)
	}
	val = strings.Join(newLines, "\n")
	p.RawValue = val
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
	// 00:00:00:00
	if p.Value == "" {
		// unset
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
	p.RawValue = strings.Join(
		[]string{
			tc[0:2], tc[2:4], tc[4:6], tc[6:8],
		},
		":",
	)
	return nil
}

func validateEntryPath(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	// It will save 'val' entry as it's id.
	if p.Value == "" {
		// unset
		p.RawValue = ""
		return nil
	}
	if p.Value == "." {
		// "." indicates the entry itself.
		// Note that it doesn't require 'entry' path, so "." can be used in defaults.
		p.RawValue = "0"
		return nil
	}
	if !path.IsAbs(p.Value) {
		// make abs path
		p.Value = path.Join(p.EntryPath, p.Value)
	}
	id, err := getEntryID(tx, ctx, p.Value)
	if err != nil {
		return err
	}
	p.RawValue = strconv.Itoa(id)
	return nil
}

// Entry name property accepts path of an entry and returns it's name.
func validateEntryName(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	// It will save 'val' entry as it's id.
	// So validation process is same with 'validateEntryPath'.
	// Difference comes from evaluation.
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
			have[pth] = true
		case '-':
			// remove
			delete(have, pth)
		}
	}
	pths := make([]string, 0, len(have))
	for pth := range have {
		pths = append(pths, pth)
	}
	sort.Strings(pths)
	val := strings.Join(pths, "\n")
	if val != "" {
		// for line matching search in sqlite
		val = "\n" + val + "\n"
	}
	p.RawValue = val
	return nil
}

func validateDate(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		// unset
		p.RawValue = ""
		return nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true,
	}
	date := ""
	for _, r := range p.Value {
		ch := string(r)
		if isDigit[ch] {
			date += ch
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
	p.RawValue = val
	return nil
}

func validateInt(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		// unset
		p.RawValue = ""
		return nil
	}
	n, err := strconv.Atoi(p.Value)
	if err != nil {
		return fmt.Errorf("cannot convert to int: %v", p.Value)
	}
	p.RawValue = strconv.Itoa(n)
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
			have[v] = true
		case '-':
			// remove
			delete(have, v)
		}
	}
	newlines := make([]string, 0, len(have))
	for v := range have {
		newlines = append(newlines, v)
	}
	sort.Strings(newlines)
	val := strings.Join(newlines, "\n")
	if val != "" {
		val = "\n" + val + "\n"
	}
	p.RawValue = val
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
	value := strings.TrimSpace(p.Value)
	lines := strings.Split(value, "\n")
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
	p.RawValue = strings.Join(newlines, "\n")
	return nil
}
