package main

import (
	"context"
	"errors"
	"testing"

	"github.com/imagvfx/forge"
)

type testEntryType struct {
	name string
	want error
}

var testEntryTypes = []testEntryType{
	{name: "show"},
	{name: "category"},
	{name: "group"},
	{name: "shot"},
	{name: "part"},
	{name: "part.mdl"},
	{name: "part.ani"},
	{name: "part.lgt"},
	{name: "having space", want: errors.New("entry type name cannot have spaces")},
	{name: "shot.double.dot", want: errors.New("entry type name is allowed only one dot for override type")},
}

type testDefault struct {
	typ     string
	ctg     string
	k, t, v string
	want    error
}

var testDefaults = []testDefault{
	{typ: "show", ctg: "property", k: "sup", t: "user", v: ""},
	{typ: "shot", ctg: "property", k: "cg", t: "text", v: ""},
	{typ: "part", ctg: "property", k: "assignee", t: "user", v: ""},
	{typ: "lol", ctg: "property", k: "assignee", t: "user", v: "", want: errors.New("entry type not found: lol")},
	{typ: "", ctg: "property", k: "assignee", t: "user", v: "", want: errors.New("default entry type not specified")},
}

type testEntry struct {
	path  string
	typ   string
	props []forge.KeyTypeValue
	want  error
}

var testEntries = []testEntry{
	{path: "/test", typ: "show"},
	{path: "/test/shot", typ: "category"},
	{path: "/test/shot/cg", typ: "group"},
	{path: "/test/shot/cg/0010", typ: "shot"},
	{path: "/test/shot/cg/0010/mdl", typ: "part"},
	{path: "/test/shot/cg/0010/ani", typ: "part"},
	{path: "/test/shot/cg/0010/lgt", typ: "part"},
	// Cannot create entry that is existing.
	{path: "/test/shot/cg/0010/lgt", typ: "part", want: errors.New("entry exists: /test/shot/cg/0010/lgt")},
	// Trailing slashes should be removed.
	{path: "/test/shot/cg/0010/lgt//", typ: "part", want: errors.New("entry exists: /test/shot/cg/0010/lgt")},
}

type testProperty struct {
	path    string
	k, t, v string
	want    error
}

var testUpdateProps = []testProperty{
	{path: "/test", k: "sup", t: "user", v: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010/ani", k: "assignee", t: "user", v: ""},
	{path: "/test/shot/cg/0010/ani", k: "assignee", t: "user", v: "not-exist@imagvfx.com", want: errors.New("user not found: not-exist@imagvfx.com")},
	{path: "/test/shot/cg/0010/ani", k: "assignee", t: "user", v: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010/lgt", k: "assignee", t: "user", v: "admin@imagvfx.com"},
}

type testSearch struct {
	path    string
	typ     string
	query   string
	wantRes []string
	wantErr error
}

var testSearches = []testSearch{
	{path: "/", typ: "", query: "admin", wantRes: []string{"/test", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/test", typ: "", query: "admin", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "show", query: "admin@imagvfx.com", wantRes: []string{"/test"}},
	{path: "/", typ: "shot", query: "admin@imagvfx.com", wantRes: []string{}},
	{path: "/", typ: "part", query: "admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "some random words", wantRes: []string{}},
	{path: "/", typ: "", query: "cg/0010/", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "cg/ mdl", wantRes: []string{"/test/shot/cg/0010/mdl"}},
	{path: "/", typ: "", query: "assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "assignee=", wantRes: []string{"/test/shot/cg/0010/mdl"}},
	// Unexpected Result
	// {path: "/", typ: "", query: "assignee:", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "", typ: "", query: "cg/ mdl", wantErr: errors.New("entry path not specified")},
}

func TestAddEntries(t *testing.T) {
	db, server, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	err = addAdmin(server, ctx)
	if err != nil {
		t.Fatal(err)
	}
	ctx = forge.ContextWithUserName(ctx, testAdmin)
	for _, typ := range testEntryTypes {
		err := server.AddEntryType(ctx, typ.name)
		if !equalError(typ.want, err) {
			t.Fatalf("want err %q, got %q", errorString(typ.want), errorString(err))
		}
	}
	for _, def := range testDefaults {
		err := server.AddDefault(ctx, def.typ, def.ctg, def.k, def.t, def.v)
		if !equalError(def.want, err) {
			t.Fatalf("want err %q, got %q", errorString(def.want), errorString(err))
		}
	}
	for _, ent := range testEntries {
		err := server.AddEntry(ctx, ent.path, ent.typ)
		if !equalError(ent.want, err) {
			t.Fatalf("want err %q, got %q", errorString(ent.want), errorString(err))
		}
	}
	for _, prop := range testUpdateProps {
		err := server.UpdateProperty(ctx, prop.path, prop.k, prop.v)
		if !equalError(prop.want, err) {
			t.Fatalf("want err %q, got %q", errorString(prop.want), errorString(err))
		}
	}
	for i, search := range testSearches {
		wantMap := make(map[string]bool)
		for _, path := range search.wantRes {
			wantMap[path] = true
		}
		ents, err := server.SearchEntries(ctx, search.path, search.typ, search.query)
		if !equalError(search.wantErr, err) {
			t.Fatalf("search: %v: want err %q, got %q", i, errorString(search.wantErr), errorString(err))
		}
		for _, e := range ents {
			if !wantMap[e.Path] {
				t.Fatalf("search: %v: got unexpected entry: %v", i, e.Path)
			}
			delete(wantMap, e.Path)
		}
		if len(wantMap) != 0 {
			t.Fatalf("search: %v: got unmatched entries: %v", i, wantMap)
		}
	}
}
