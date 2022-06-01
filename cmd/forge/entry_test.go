package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
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
	{name: "asset"},
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
	{typ: "part", ctg: "property", k: "status", t: "text", v: ""},
	{typ: "part", ctg: "property", k: "direction", t: "text", v: ""},
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
	{path: "/test/shot/cg/0010/match", typ: "part"},
	{path: "/test/shot/cg/0010/ani", typ: "part"},
	{path: "/test/shot/cg/0010/lgt", typ: "part"},
	// Cannot create entry that is existing.
	{path: "/test/shot/cg/0010/lgt", typ: "part", want: errors.New("entry exists: /test/shot/cg/0010/lgt")},
	// Trailing slashes should be removed.
	{path: "/test/shot/cg/0010/lgt//", typ: "part", want: errors.New("entry exists: /test/shot/cg/0010/lgt")},
	{path: "/test/shot/not-exist/0010/fx", typ: "part", want: errors.New("check parent: entry not found: /test/shot/not-exist/0010")},
	{path: "/test/shot/cg/0010/a part", typ: "part", want: errors.New("entry name has invalid character ' ': /test/shot/cg/0010/a part")},
	{path: "\\test\\shot\\not-exist\\0010\\fx", typ: "part", want: errors.New("entry name has invalid character '\\': \\test\\shot\\not-exist\\0010\\fx")},
	{path: "/test/shot/cg/0010/#fx", typ: "part", want: errors.New("entry name has invalid character '#': /test/shot/cg/0010/#fx")},
	// validation of parent path checks it's existance. no check to invalid characters.
	{path: "/test/shot/#cg/0010/lgt", typ: "part", want: errors.New("check parent: entry not found: /test/shot/#cg/0010")},
	{path: "/test/shot/cg/0020", typ: "shot"},
	{path: "/test/shot/cg/0030", typ: "shot"},
	{path: "/test/asset", typ: "category"},
	{path: "/test/asset/char", typ: "group"},
	{path: "/test/asset/char/yb", typ: "asset"},
}

type testProperty struct {
	path    string
	k, t, v string
	want    error
}

var testUpdateProps = []testProperty{
	{path: "/test", k: "sup", v: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010", k: "cg", v: "remove"},
	{path: "/test/shot/cg/0010/match", k: "status", v: "omit"},
	{path: "/test/shot/cg/0010/ani", k: "assignee", v: ""},
	{path: "/test/shot/cg/0010/ani", k: "assignee", v: "not-exist@imagvfx.com", want: errors.New("user not found: not-exist@imagvfx.com")},
	{path: "/test/shot/cg/0010/ani", k: "assignee", v: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010/ani", k: "status", v: "done"},
	{path: "/test/shot/cg/0010/lgt", k: "assignee", v: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010/lgt", k: "status", v: "inprogress"},
	{path: "/test/shot/cg/0010/lgt", k: "direction", v: "make the whole scene brighter"},
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
	{path: "/", typ: "", query: "cg/0010/", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "cg/ mdl", wantRes: []string{"/test/shot/cg/0010/mdl"}},
	{path: "/", typ: "", query: "assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "assignee=", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match"}},
	{path: "/", typ: "", query: "assignee=admin@imagvfx.com,", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match"}},
	{path: "/", typ: "", query: "assignee:", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "ani.assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "", query: "mdl.assignee:admin", wantRes: []string{}},
	{path: "/", typ: "shot", query: "(sub).assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "shot", query: "(sub).assignee=", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "shot", query: "(sub).assignee=xyz@imagvfx.com", wantRes: []string{}},
	{path: "/", typ: "", query: "assignee:admin status=inprogress", wantRes: []string{"/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "status=,inprogress,done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "status!=", wantRes: []string{"/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "status!=omit status!=done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "status!=omit,done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "(sub).sup=admin@imagvfx.com", wantRes: []string{"/"}},
	{path: "/", typ: "", query: "(sub).sup!=,admin@imagvfx.com", wantRes: []string{}},
	{path: "/", typ: "", query: "(sub).cg:remove", wantRes: []string{"/test/shot/cg"}},
	{path: "", typ: "", query: "cg/ mdl", wantRes: []string{}, wantErr: errors.New("entry path not specified")},
	{path: "/", typ: "", query: ":", wantRes: []string{}},
	{path: "/", typ: "", query: ":val", wantRes: []string{}},
	{path: "/", typ: "", query: "=", wantRes: []string{}},
	{path: "/", typ: "", query: "=val", wantRes: []string{}},
	{path: "/", typ: "", query: ".=", wantRes: []string{}},
	{path: "/", typ: "", query: ".=val", wantRes: []string{}},
	{path: "/", typ: "", query: ".cg=val", wantRes: []string{}},
	{path: "/", typ: "", query: "(sub).=val", wantRes: []string{}},
	{path: "/", typ: "", query: "comp.=val", wantRes: []string{}},
	{path: "/", typ: "", query: "comp.x=val", wantRes: []string{}},
	{path: "/", typ: "", query: "comp.x=val", wantRes: []string{}},
	{path: "/test", typ: "", query: "path:/test/shot", wantRes: []string{"/test/shot", "/test/shot/cg", "/test/shot/cg/0010", "/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/test", typ: "", query: "path!:/test/shot", wantRes: []string{"/test/asset", "/test/asset/char", "/test/asset/char/yb"}},
	{path: "/test", typ: "shot", query: "path=/test/shot/cg/0010", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/test", typ: "shot", query: "path!=/test/shot/cg/0010", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", typ: "shot", query: "name=0010", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "shot", query: "name!=0010", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", typ: "part", query: "name=ani", wantRes: []string{"/test/shot/cg/0010/ani"}},
}

func TestAddEntries(t *testing.T) {
	db, server, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	// first user who was added to the db becomes an admin
	for _, user := range []string{"admin@imagvfx.com", "readwriter@imagvfx.com", "reader@imagvfx.com", "blocked@imagvfx.com"} {
		err = server.AddUser(ctx, &forge.User{Name: user})
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx = forge.ContextWithUserName(ctx, "admin@imagvfx.com")
	groupMembers := map[string][]string{
		"readers": {"reader@imagvfx.com", "readwriter@imagvfx.com"},
		"writers": {"readwriter@imagvfx.com"},
	}
	for group, members := range groupMembers {
		err = server.AddGroup(ctx, &forge.Group{Name: group})
		if err != nil {
			t.Fatal(err)
		}
		for _, member := range members {
			err = server.AddGroupMember(ctx, group, member)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
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
		ents, err := server.SearchEntries(ctx, search.path, search.typ, search.query)
		if !equalError(search.wantErr, err) {
			t.Fatalf("search: %v: got error %q, want %q", i, errorString(err), errorString(search.wantErr))
		}
		got := make([]string, 0)
		for _, e := range ents {
			got = append(got, e.Path)
		}
		sort.Strings(got)
		sort.Strings(search.wantRes)
		if !reflect.DeepEqual(got, search.wantRes) {
			label := fmt.Sprintf("searched %q from %q", search.query, search.path)
			if search.typ != "" {
				label += fmt.Sprintf("of type %q", search.typ)
			}
			t.Fatalf(label+": got %q, want %q", got, search.wantRes)
		}
	}
}
