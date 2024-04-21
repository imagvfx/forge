package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

var pageHandlerFuncs = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	"add": func(a, b int) int {
		return a + b
	},
	"addString": func(a, b string) string {
		return a + b
	},
	"strIndex": strings.Index,
	"min": func(a, b int) int {
		if a < b {
			return a
		}
		return b
	},
	"has": func(s, tok string) bool {
		return strings.Contains(s, tok)
	},
	"hasPrefix":  strings.HasPrefix,
	"trim":       strings.TrimSpace,
	"trimPrefix": strings.TrimPrefix,
	"remapFrom": func(s string) string {
		return strings.Split(s, ";")[0]
	},
	"remapTo": func(s string) string {
		if !strings.Contains(s, ";") {
			return ""
		}
		return strings.Split(s, ";")[1]
	},
	"formatTime": func(t time.Time) string {
		return t.Format(time.RFC3339)
	},
	"subEntryName": func(name string) template.HTML {
		// wrap entry name after underscore or before camel cases
		wasUnder := false
		wasUpper := false
		word := ""
		words := make([]string, 0)
		for _, r := range name {
			s := string(r)
			under := false
			upper := false
			if s == "_" {
				under = true
			} else if strings.ToUpper(s) == s {
				upper = true
			}
			if (wasUnder && !under) || (!wasUpper && upper) {
				words = append(words, word)
				word = ""
			}
			wasUnder = under
			wasUpper = upper
			word += s
		}
		if word != "" {
			words = append(words, word)
		}
		text := strings.Join(words, "<wbr>")
		return template.HTML(text)
	},
	"pathLinks": func(path string) (template.HTML, error) {
		if !strings.HasPrefix(path, "/") {
			return "", fmt.Errorf("path should start with /")
		}
		full := ""
		link := ""
		ps := strings.Split(path[1:], "/")
		for _, p := range ps {
			p = template.HTMLEscapeString("/" + p)
			link += p
			full += fmt.Sprintf(`<a class="pathLink" href="%v">%v</a>`, link, p)
		}
		return template.HTML(full), nil
	},
	"handleAsset": func(asset string) template.HTML {
		lines := strings.Split(asset, "\n")
		res := ""
		for _, ln := range lines {
			ln = strings.TrimSpace(ln)
			if !strings.HasPrefix(ln, "/") {
				continue
			}
			toks := strings.Split(ln, "/")
			name := toks[len(toks)-1]
			if len(ln) == 1 {
				// show root as /
				name = "/"
			}
			res += fmt.Sprintf(`<div class="assetLink copyable" data-copy-field="entryPath" data-entry-path="%s"><div class="assetStatus statusDot"></div>%s</div>`, ln, name)
		}
		return template.HTML(res)
	},
	"handleKeyshot": func(asset string) template.HTML {
		lines := strings.Split(asset, "\n")
		res := ""
		for _, ln := range lines {
			ln = strings.TrimSpace(ln)
			if !strings.HasPrefix(ln, "/") {
				continue
			}
			toks := strings.Split(ln, "/")
			name := toks[len(toks)-1]
			if len(ln) == 1 {
				// show root as /
				name = "/"
			}
			res += fmt.Sprintf(`<div class="keyshotLink copyable" data-copy-field="entryPath" data-entry-path="%s"><div class="keyshotStatus statusDot"></div>%s</div>`, ln, name)
		}
		return template.HTML(res)
	},
	"sortProperty": func(p string) string {
		if len(p) == 0 {
			return ""
		}
		if len(p) == 1 {
			// only sort order defined
			return ""
		}
		return p[1:]
	},
	"sortDesc": func(p string) bool {
		if len(p) == 0 {
			return false
		}
		order := p[0]
		// '+' means ascending, '-' means descending order
		return order == '-'
	},
	"marshalJS": func(v any) (template.JS, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return template.JS(string(b)), nil
	},
	"infoValueElement": func(p *forge.Property) template.HTML {
		// If you are going to modify this function,
		// You should also modify 'refreshInfoValue' function in tmpl/entry.bml.js.
		if p.ValueError != nil {
			return template.HTML("<div class='invalid infoValue'>" + p.ValueError.Error() + "</div>")
		}
		t := ""
		lines := strings.Split(p.Eval, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				t += "<br>"
				continue
			}
			if p.Type == "tag" {
				t += "<div class='tagLink' data-tag-name='" + p.Name + "' data-tag-value='" + line + "'>" + line + "</div>"
			} else if p.Type == "entry_link" {
				t += "<a class='entryLink' href='" + line + "'>" + line + "</a>"
			} else if p.Type == "search" {
				name, query, ok := strings.Cut(line, "|")
				if !ok {
					continue
				}
				t += "<div class='searchLink' data-search-from='" + p.EntryPath + "' data-search-query='" + query + "'>" + name + "</div>"
			} else {
				// p.Type == text, etc...
				line = template.HTMLEscapeString(line)
				if strings.HasPrefix(line, "/") {
					t += "<div class='pathText'>" + line + "</div>"
				} else if strings.HasPrefix(line, "https://") {
					t += "<div class='externalLinkContainer'><div class='externalLinkIcon'></div><a class='externalLink' href='" + line + "' target='_blank'>" + line[8:] + "</a></div>"
				} else {
					t += "<div>" + line + "</div>"
				}
			}
		}
		return template.HTML("<div class='infoValue'>" + t + "</div>")
	},
	"escapeQuery": func(s string) string {
		return url.QueryEscape(s)
	},
	// topName is name of the entry directly under the root for given path.
	// Which indicates a show name.
	"topName": func(s string) string {
		return strings.Split(s, "/")[1]
	},
	"setAlphaToColor": func(c string, alpha float32) string {
		c = strings.TrimSpace(c)
		hexChar := "0123456789ABCDEF"
		if strings.HasPrefix(c, "#") {
			cc := c[1:]
			if len(cc) != 3 && len(cc) != 6 {
				return c
			}
			long := len(cc) == 6
			a := int(alpha * 255)
			astr := string(hexChar[a/16])
			if long {
				astr += string(hexChar[a%16])
			}
			return c + astr
		}
		return c
	},
	"recent": func(t time.Time, days int) (bool, error) {
		delta := time.Now().UTC().Sub(t)
		if delta < time.Duration(days)*24*time.Hour {
			return true, nil
		}
		return false, nil
	},
	"dayLeft": func(timestr string) string {
		if timestr == "" {
			return ""
		}
		t, err := time.ParseInLocation("2006/01/02", timestr, time.Local)
		if err != nil {
			return "!"
		}
		remain := time.Now().Sub(t)
		day := int(remain / (24 * time.Hour))
		if remain < 0 {
			day -= 1
		}
		left := strconv.Itoa(day)
		if day == 0 {
			left = "D-" + left
		} else if day < 0 {
			left = "D" + left
		} else {
			left = "D+" + left
		}
		return left
	},
	"dir":  filepath.Dir,
	"base": filepath.Base,
	"statusSummary": func(entGroups [][]*forge.Entry) map[string]map[string]int {
		// to know what is entGroups, see grandSubEntGroups
		summary := make(map[string]map[string]int) // map[entryType][status]occurence
		for _, ents := range entGroups {
			for _, ent := range ents {
				if summary[ent.Type] == nil {
					summary[ent.Type] = make(map[string]int)
				}
				stat := ""
				if ent.Property["status"] != nil {
					stat = ent.Property["status"].Eval
				}
				summary[ent.Type][stat]++
			}
		}
		return summary
	},
}
