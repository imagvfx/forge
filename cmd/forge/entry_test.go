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

type testEntry struct {
	path string
	typ  string
	want error
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
	{path: "/test/shot/cg/0010/lgt", typ: "part", want: errors.New("UNIQUE constraint failed: entries.path")},
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
		err = server.AddEntryType(ctx, typ)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, ent := range testEntries {
		got := server.AddEntry(ctx, ent.path, ent.typ)
		if !equalError(ent.want, got) {
			t.Fatalf("want err %q, got %q", ent.want, got)
		}
	}
}
