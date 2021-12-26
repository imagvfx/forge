package main

import (
	"context"
	"errors"
	"testing"

	"github.com/imagvfx/forge"
)

var testEntryTypes = []string{
	"show",
	"category",
	"group",
	"shot",
	"part",
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

var testProps = []testEntry{}

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
		err = server.AddEntryType(ctx, typ)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, def := range testDefaults {
		err := server.AddDefault(ctx, def.typ, def.ctg, def.k, def.t, def.v)
		if !equalError(def.want, err) {
			t.Fatalf("want err %q, got %q", errorString(def.want), errorString(err))
		}
	}
	for _, ent := range testEntries {
		got := server.AddEntry(ctx, ent.path, ent.typ)
		if !equalError(ent.want, got) {
			t.Fatalf("want err %q, got %q", errorString(ent.want), errorString(got))
		}
	}
}
