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
		"chat":       validateChat,
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
	// if the value starts with + or -, it will change the current date
	val := p.Value
	if val == "0" {
		t := time.Now().Local()
		p.Value = t.Format("2006/01/02")
		p.RawValue = p.Value
		return nil
	}
	possiblePrefix := rune(val[0])
	if possiblePrefix == '+' || possiblePrefix == '-' {
		day, err := strconv.Atoi(val[1:])
		if err != nil {
			return fmt.Errorf("invalid date operation: +/- operation needs digits only, got: %v", val[1:])
		}
		if possiblePrefix == '-' {
			day *= -1
		}
		t := time.Now().Local()
		if old.Value != "" {
			var err error
			t, err = time.Parse("2006/01/02", old.Value)
			if err != nil {
				return fmt.Errorf("invalid date string: %v", err)
			}
		}
		t = t.AddDate(0, 0, day)
		p.Value = t.Format("2006/01/02")
		p.RawValue = p.Value
		return nil
	}
	// the value should be a plain date.
	// need 8 digits in what ever form.
	isDigit := map[rune]bool{
		'0': true, '1': true, '2': true, '3': true, '4': true,
		'5': true, '6': true, '7': true, '8': true, '9': true,
	}
	date := ""
	for _, r := range p.Value {
		if isDigit[r] {
			date += string(r)
		}
	}
	if len(date) != 8 {
		return fmt.Errorf("invalid date string: want yyyy/mm/dd, got %v", p.Value)
	}
	rawval := strings.Join(
		[]string{
			date[0:4], date[4:6], date[6:8],
		},
		"/",
	)
	_, err := time.Parse("2006/01/02", rawval)
	if err != nil {
		return fmt.Errorf("invalid date string: %v", err)
	}
	p.Value = rawval
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
		for _, v := range strings.Split(old.Value, "\n") {
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
		// new form of tag adds '[' and ']' to not allow sqlite
		// to trim newline characters at start and end.
		// (when the value only have a int)
		// those are needed for search.

		rawVal = "[\n" + rawVal + "\n]"
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

func validateChat(tx *sql.Tx, ctx context.Context, p, old *forge.Property) error {
	if p.Value == "" {
		return nil
	}
	ctxID := forge.ContextIDFromContext(ctx)
	if ctxID == "" {
		return fmt.Errorf("context id needed for a chat")
	}
	// only takes '+', '-', '>' as a prefix.
	val := p.Value
	op := val[0]
	if op != '+' && op != '-' && op != '>' {
		return fmt.Errorf("invalid op for chat: %s", string(op))
	}
	user := forge.UserNameFromContext(ctx)
	t := forge.TimeFromContext(ctx)
	now := t.Local()
	stamp := now.Format(time.RFC3339)
	output := old.RawValue
	switch op {
	case '+':
		// input:
		// +msg
		// output:
		// *id user stamp
		// |msg
		chatID := ctxID
		output += "\n*" + chatID + " " + user + " " + stamp
		msg := strings.TrimSpace(val[1:])
		for _, line := range strings.Split(msg, "\n") {
			output += "\n|" + line
		}
	case '-', '>':
		// case '-'
		// delete chat
		// input:
		// -id
		// output:
		// (removed the chat from val)

		// case '>'
		// reply to a chat
		// input:
		// >id msg
		// output:
		// *id user stamp
		// |msg
		// |*reply_id user stamp
		// ||msg
		newOutput := ""
		val := strings.TrimSpace(val[1:])
		chatID, reply, _ := strings.Cut(val, " ")
		if chatID == "" {
			if op == '-' {
				return fmt.Errorf("no chat id to delete")
			} else {
				return fmt.Errorf("no chat id to reply")
			}
		}
		if op == '>' && reply == "" {
			return fmt.Errorf("no message to reply")
		}
		// f deletes or replies to ch. ch will be replaced by it's output.
		f := func(ch string, d int) (string, error) {
			if op == '-' {
				// TODO: check if replies are exists
				return "", nil
			}
			// '>'
			if d >= 1 {
				// Structuring it is not a problem but displaying is.
				return "", fmt.Errorf("cannot reply to a reply: %v", chatID)
			}
			replyID := ctxID
			ch += "\n" + strings.Repeat("|", d+1) + "*" + replyID + " " + user + " " + stamp
			for _, line := range strings.Split(reply, "\n") {
				ch += "\n" + strings.Repeat("|", d+2) + line
			}
			return ch, nil
		}
		newOutput, done, err := traverseChat(output, chatID, f, 0)
		if err != nil {
			return err
		}
		if !done {
			return fmt.Errorf("chat not found: %s", chatID)
		}
		output = newOutput
	}
	p.Value = val
	p.RawValue = output
	return nil
}

// traverseChat traverses chats to find a chat with 'id'. Then f will be applied to the chat.
// Other chats will not be affected.
func traverseChat(chat string, id string, f func(string, int) (string, error), d int) (string, bool, error) {
	sep := "\n" + strings.Repeat("|", d) + "*"
	replySep := "\n" + strings.Repeat("|", d+1) + "*"
	done := false
	outs := make([]string, 0)
	for _, ch := range strings.Split(chat, sep) {
		if done {
			outs = append(outs, ch)
			continue
		}
		if strings.HasPrefix(ch, id) {
			done = true
			out, err := f(ch, d)
			if err != nil {
				return "", false, err
			}
			if out != "" {
				outs = append(outs, out)
			}
			continue
		}
		// This chat isn't what we are finding. But one of it's replies might.
		thisChat, replyMessages, ok := strings.Cut(ch, replySep)
		if !ok {
			outs = append(outs, ch)
			continue
		}
		out, ok, err := traverseChat(replyMessages, id, f, d+1)
		if err != nil {
			return "", false, err
		}
		if out != "" {
			thisChat = thisChat + replySep + out
		}
		outs = append(outs, thisChat)
		done = ok
	}
	output := strings.Join(outs, sep)
	return output, done, nil
}
