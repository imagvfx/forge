package main

import (
	"context"
	"errors"
	"fmt"
	"path"
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
	{path: "/test/shot/cg/0020/ani", typ: "part"},
	{path: "/test/shot/cg/0030", typ: "shot"},
	{path: "/test/asset", typ: "category"},
	{path: "/test/asset/char", typ: "group"},
	{path: "/test/asset/char/yb", typ: "asset"},
	// check case sensitive search for entries,
	{path: "/TEST", typ: "show"},
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
	{path: "/test/shot/cg/0020/ani", k: "assignee", v: "reader@imagvfx.com"},
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
	{path: "/", typ: "", query: "assignee:", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", typ: "", query: "ani.assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "", query: "mdl.assignee:admin", wantRes: []string{}},
	{path: "/", typ: "shot", query: "(sub).assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "shot", query: "(sub).assignee=", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "shot", query: "(sub).assignee=xyz@imagvfx.com", wantRes: []string{}},
	{path: "/", typ: "", query: "assignee:admin status=inprogress", wantRes: []string{"/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "status=,inprogress,done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", typ: "", query: "status!=", wantRes: []string{"/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", typ: "", query: "status!=omit status!=done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", typ: "", query: "status!=omit,done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
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
	{path: "/test", typ: "shot", query: "ani.status!=done", wantRes: []string{"/test/shot/cg/0020"}},
	{path: "/test", typ: "", query: "path:/test/shot", wantRes: []string{"/test/shot", "/test/shot/cg", "/test/shot/cg/0010", "/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020", "/test/shot/cg/0020/ani", "/test/shot/cg/0030"}},
	{path: "/test", typ: "", query: "path!:/test/shot", wantRes: []string{"/test/asset", "/test/asset/char", "/test/asset/char/yb"}},
	{path: "/test", typ: "shot", query: "path=/test/shot/cg/0010", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/test", typ: "shot", query: "path!=/test/shot/cg/0010", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", typ: "shot", query: "name=0010", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", typ: "shot", query: "name!=0010", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", typ: "part", query: "name=ani", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0020/ani"}},
}

type testRename struct {
	path    string
	newName string
	wantErr error
}

var testRenames = []testRename{
	{
		path:    "/TEST",
		newName: "test",
		wantErr: errors.New("rename target path already exists: /test"),
	},
}

type testDelete struct {
	path    string
	wantErr error
}

var testDeletes = []testDelete{
	{
		path: "/TEST",
	},
}

func TestAddEntries(t *testing.T) {
	db, server, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	// first user who was added to the db becomes an admin
	users := []string{"admin@imagvfx.com", "readwriter@imagvfx.com", "reader@imagvfx.com", "uninvited@imagvfx.com"}
	for _, user := range users {
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
	access := map[string]string{
		"readers": "r",
		"writers": "rw",
	}
	for group, access := range access {
		err = server.AddAccess(ctx, "/", group, access)
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

	// test renames and revert it back.
	ctx = forge.ContextWithUserName(ctx, "admin@imagvfx.com")
	for _, rename := range testRenames {
		dir := path.Dir(rename.path)
		oldName := path.Base(rename.path)
		err := server.RenameEntry(ctx, rename.path, rename.newName)
		if !equalError(rename.wantErr, err) {
			t.Fatalf("rename %q to %q: want err %q, got %q", rename.path, rename.newName, errorString(rename.wantErr), errorString(err))
		}
		if err != nil {
			// The rename wasn't done, no need to revert.
			continue
		}
		// revert
		newPath := path.Join(dir, rename.newName)
		err = server.RenameEntry(ctx, newPath, oldName)
		if err != nil {
			t.Fatalf("rename %q to %q: revert got unwanted err: %v", rename.path, rename.newName, err)
		}
	}

	errorLabel := func(s testSearch) string {
		l := fmt.Sprintf("searched %q from %q", s.query, s.path)
		if s.typ != "" {
			l += fmt.Sprintf("of type %q", s.typ)
		}
		return l
	}
	whoCanRead := []string{"admin@imagvfx.com", "readwriter@imagvfx.com", "reader@imagvfx.com"}
	for _, user := range whoCanRead {
		ctx = forge.ContextWithUserName(ctx, user)
		for _, search := range testSearches {
			ents, err := server.SearchEntries(ctx, search.path, search.typ, search.query)
			if !equalError(search.wantErr, err) {
				t.Fatalf("%v: got error %q, want %q", errorLabel(search), errorString(err), errorString(search.wantErr))
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
				t.Fatalf("%v: got %q, want %q", errorLabel(search), got, search.wantRes)
			}
		}
	}
	whoCannotRead := []string{"uninvited@imagvfx.com"}
	for _, user := range whoCannotRead {
		ctx = forge.ContextWithUserName(ctx, user)
		for _, search := range testSearches {
			ents, _ := server.SearchEntries(ctx, search.path, search.typ, search.query)
			got := make([]string, 0)
			for _, e := range ents {
				got = append(got, e.Path)
			}
			if len(got) != 0 {
				if len(got) == 1 && got[0] == "/" {
					// anyone can search root, even to an uninvited user
					continue
				}
				label := fmt.Sprintf("searched %q from %q", search.query, search.path)
				if search.typ != "" {
					label += fmt.Sprintf("of type %q", search.typ)
				}
				t.Fatalf(label+": uninvited user shouldn't be able to search child entries, got: %v", got)
			}
		}
	}

	// test delete
	ctx = forge.ContextWithUserName(ctx, "admin@imagvfx.com")
	for _, delete := range testDeletes {
		err := server.DeleteEntry(ctx, delete.path)
		if !equalError(delete.wantErr, err) {
			t.Fatalf("delete %q: want err %q, got %q", delete.path, errorString(delete.wantErr), errorString(err))
		}
	}
}
