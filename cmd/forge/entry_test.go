package main

import (
	"context"
	"errors"
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
	{typ: "shot", ctg: "property", k: "direction", t: "text", v: ""},
	{typ: "shot", ctg: "property", k: "due", t: "date", v: ""},
	{typ: "shot", ctg: "property", k: "timecode", t: "timecode", v: ""},
	{typ: "shot", ctg: "property", k: "tag", t: "tag", v: ""},
	{typ: "shot", ctg: "property", k: "duration", t: "int", v: ""},
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
	expect  string
}

var testUpdateProps = []testProperty{
	// below for pure validation. clean-up after set.
	{path: "/test/shot/cg/0010", k: "direction", v: "title\r\n\r\n", expect: "title\n\n"},
	{path: "/test/shot/cg/0010", k: "direction", v: "", expect: ""},
	{path: "/test/shot/cg/0010", k: "due", v: "2023/05/21", expect: "2023/05/21"},
	{path: "/test/shot/cg/0010", k: "due", v: "20230521", expect: "2023/05/21"},
	{path: "/test/shot/cg/0010", k: "due", v: "2023/99/99", want: errors.New("invalid date string: parsing time \"2023/99/99\": month out of range")},
	{path: "/test/shot/cg/0010", k: "due", v: "2023", want: errors.New("invalid date string: want yyyy/mm/dd, got 2023")},
	{path: "/test/shot/cg/0010", k: "due", v: "", expect: ""},
	{path: "/test/shot/cg/0010", k: "timecode", v: "00:00:00:00", expect: "00:00:00:00"},
	{path: "/test/shot/cg/0010", k: "timecode", v: "00000000", expect: "00:00:00:00"},
	{path: "/test/shot/cg/0010", k: "timecode", v: "00:00", want: errors.New("invalid timecode string: 00:00")},
	{path: "/test/shot/cg/0010", k: "timecode", v: "", expect: ""},
	{path: "/test/shot/cg/0010", k: "duration", v: "24", expect: "24"},
	{path: "/test/shot/cg/0010", k: "duration", v: "24.1", want: errors.New("cannot convert to int: 24.1")},
	{path: "/test/shot/cg/0010", k: "duration", v: "not a number", want: errors.New("cannot convert to int: not a number")},
	{path: "/test/shot/cg/0010", k: "duration", v: "", expect: ""},
	{path: "/test/shot/cg/0010", k: "tag", v: "+a", expect: "a"},
	{path: "/test/shot/cg/0010", k: "tag", v: "+b", expect: "a\nb"},
	{path: "/test/shot/cg/0010", k: "tag", v: "-a", expect: "b"},
	{path: "/test/shot/cg/0010", k: "tag", v: "-b\n+c", expect: "c"},
	{path: "/test/shot/cg/0010", k: "tag", v: "-b\n+c", expect: "c"},
	{path: "/test/shot/cg/0010", k: "tag", v: "ab", expect: "c"},
	{path: "/test/shot/cg/0010", k: "tag", v: "+a,b", expect: "a_b\nc"},
	{path: "/test/shot/cg/0010", k: "tag", v: "-a,b", expect: "c"},
	{path: "/test/shot/cg/0010", k: "tag", v: "+a\n+b", expect: "a\nb\nc"},
	{path: "/test/shot/cg/0010", k: "tag", v: "-a\n-b\n-c", expect: ""},

	// below properties for search.
	{path: "/test", k: "sup", v: "admin@imagvfx.com", expect: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010", k: "cg", v: "remove", expect: "remove"},
	{path: "/test/shot/cg/0010", k: "tag", v: "+due=2023/05/21\n+important", expect: "due=2023/05/21\nimportant"},
	{path: "/test/shot/cg/0020", k: "tag", v: "+due=2023/08/12", expect: "due=2023/08/12"},
	{path: "/test/shot/cg/0030", k: "tag", v: "+test", expect: "test"},
	{path: "/test/shot/cg/0010/match", k: "status", v: "omit", expect: "omit"},
	{path: "/test/shot/cg/0010/ani", k: "assignee", v: "", expect: ""},
	{path: "/test/shot/cg/0010/ani", k: "assignee", v: "not-exist@imagvfx.com", want: errors.New("user not found: not-exist@imagvfx.com")},
	{path: "/test/shot/cg/0010/ani", k: "assignee", v: "admin@imagvfx.com", expect: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010/ani", k: "status", v: "done", expect: "done"},
	{path: "/test/shot/cg/0010/lgt", k: "assignee", v: "admin@imagvfx.com", expect: "admin@imagvfx.com"},
	{path: "/test/shot/cg/0010/lgt", k: "status", v: "inprogress", expect: "inprogress"},
	{path: "/test/shot/cg/0010/lgt", k: "direction", v: "make the whole scene brighter", expect: "make the whole scene brighter"},
	{path: "/test/shot/cg/0020/ani", k: "assignee", v: "reader@imagvfx.com", expect: "reader@imagvfx.com"},
}

type testSearch struct {
	path    string
	query   string
	wantRes []string
	wantErr error
}

var testSearches = []testSearch{
	{path: "/", query: "admin", wantRes: []string{"/test", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/test", query: "admin", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", query: "type=show admin@imagvfx.com", wantRes: []string{"/test"}},
	{path: "/", query: "type=shot admin@imagvfx.com", wantRes: []string{}},
	{path: "/", query: "type=part admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", query: "some random words", wantRes: []string{}},
	{path: "/", query: "cg/0010/", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", query: "cg/ mdl", wantRes: []string{"/test/shot/cg/0010/mdl"}},
	{path: "/", query: "assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", query: "assignee=", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match"}},
	{path: "/", query: "assignee=admin@imagvfx.com,", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match"}},
	{path: "/", query: "assignee:", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", query: "ani.assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "mdl.assignee:admin", wantRes: []string{}},
	{path: "/", query: "type=shot (sub).assignee=admin@imagvfx.com", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "type=shot (sub).assignee=", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "type=shot (sub).assignee=xyz@imagvfx.com", wantRes: []string{}},
	{path: "/", query: "assignee:admin status=inprogress", wantRes: []string{"/test/shot/cg/0010/lgt"}},
	{path: "/", query: "status=,inprogress,done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", query: "status!=", wantRes: []string{"/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt"}},
	{path: "/", query: "status!=omit status!=done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", query: "status!=omit,done", wantRes: []string{"/test/shot/cg/0010/mdl", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020/ani"}},
	{path: "/", query: "(sub).sup=admin@imagvfx.com", wantRes: []string{"/"}},
	{path: "/", query: "(sub).sup!=,admin@imagvfx.com", wantRes: []string{}},
	{path: "/", query: "(sub).cg:remove", wantRes: []string{"/test/shot/cg"}},
	{path: "", query: "cg/ mdl", wantRes: []string{}, wantErr: errors.New("entry path not specified")},
	{path: "/", query: ":", wantRes: []string{}},
	{path: "/", query: ":val", wantRes: []string{}},
	{path: "/", query: "=", wantRes: []string{}},
	{path: "/", query: "=val", wantRes: []string{}},
	{path: "/", query: ".=", wantRes: []string{}},
	{path: "/", query: ".=val", wantRes: []string{}},
	{path: "/", query: ".cg=val", wantRes: []string{}},
	{path: "/", query: "(sub).=val", wantRes: []string{}},
	{path: "/", query: "comp.=val", wantRes: []string{}},
	{path: "/", query: "comp.x=val", wantRes: []string{}},
	{path: "/", query: "comp.x=val", wantRes: []string{}},
	{path: "/test", query: "type=shot ani.status!=done", wantRes: []string{"/test/shot/cg/0020"}},
	{path: "/test", query: "path:/test/shot", wantRes: []string{"/test/shot", "/test/shot/cg", "/test/shot/cg/0010", "/test/shot/cg/0010/mdl", "/test/shot/cg/0010/match", "/test/shot/cg/0010/ani", "/test/shot/cg/0010/lgt", "/test/shot/cg/0020", "/test/shot/cg/0020/ani", "/test/shot/cg/0030"}},
	{path: "/test", query: "path!:/test/shot", wantRes: []string{"/test/asset", "/test/asset/char", "/test/asset/char/yb"}},
	{path: "/test", query: "type=shot path=/test/shot/cg/0010", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/test", query: "type=shot path!=/test/shot/cg/0010", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", query: "type=shot name=0010", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "type=shot name!=0010", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", query: "type=part name=ani", wantRes: []string{"/test/shot/cg/0010/ani", "/test/shot/cg/0020/ani"}},
	{path: "/", query: "tag=due=2023/05/21", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "tag!=due=2023/05/21", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", query: "tag:due", wantRes: []string{"/test/shot/cg/0010", "/test/shot/cg/0020"}},
	{path: "/", query: "tag!:due", wantRes: []string{"/test/shot/cg/0030"}},
	{path: "/", query: "tag:due tag=important", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "tag=important", wantRes: []string{"/test/shot/cg/0010"}},
	{path: "/", query: "tag!=important", wantRes: []string{"/test/shot/cg/0020", "/test/shot/cg/0030"}},
	{path: "/", query: "tag=important,test", wantRes: []string{"/test/shot/cg/0010", "/test/shot/cg/0030"}},
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

type testUserData struct {
	label   string
	ctxUser string
	user    string
	section string
	key     string
	value   string
	wantErr error
}

var userDataCases = []testUserData{
	// assumes the context user is admin@imagvfx.com
	{
		label:   "option1",
		user:    "admin@imagvfx.com",
		section: "app1",
		key:     "option1",
		value:   "1",
	},
	{
		label:   "option2",
		user:    "admin@imagvfx.com",
		section: "app1",
		key:     "option2",
		value:   "0",
	},
	{
		label:   "option3",
		user:    "admin@imagvfx.com",
		section: "app1",
		key:     "option3",
		value:   "",
	},
	{
		label:   "different user",
		user:    "reader@imagvfx.com",
		section: "app1",
		key:     "option3",
		value:   "",
		wantErr: errors.New("cannot set user-data to another user"),
	},
	{
		label:   "no section",
		user:    "admin@imagvfx.com",
		section: "",
		key:     "option1",
		value:   "",
		wantErr: errors.New("user data section cannot be empty"),
	},
	{
		label:   "no key",
		user:    "admin@imagvfx.com",
		section: "app1",
		key:     "",
		value:   "",
		wantErr: errors.New("user data key cannot be empty"),
	},
	{
		label:   "update option1",
		user:    "admin@imagvfx.com",
		section: "app1",
		key:     "option1",
		value:   "",
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
		if prop.want != nil {
			continue
		}
		got, err := server.GetProperty(ctx, prop.path, prop.k)
		if err != nil {
			t.Fatalf("couldn't get updated property: %v", err)
		}
		if got.Value != prop.expect {
			t.Fatalf("want value %q, got %q", prop.expect, got.Value)
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

	whoCanRead := []string{"admin@imagvfx.com", "readwriter@imagvfx.com", "reader@imagvfx.com"}
	for _, user := range whoCanRead {
		ctx = forge.ContextWithUserName(ctx, user)
		for _, s := range testSearches {
			ents, err := server.SearchEntries(ctx, s.path, s.query)
			if !equalError(s.wantErr, err) {
				t.Fatalf("searched %q from %q: got error %q, want %q", s.query, s.path, errorString(err), errorString(s.wantErr))
			}
			got := make([]string, 0)
			for _, e := range ents {
				got = append(got, e.Path)
			}
			sort.Strings(got)
			sort.Strings(s.wantRes)
			if !reflect.DeepEqual(got, s.wantRes) {
				t.Fatalf("searched %q from %q: got %q, want %q", s.query, s.path, got, s.wantRes)
			}
		}
	}
	whoCannotRead := []string{"uninvited@imagvfx.com"}
	for _, user := range whoCannotRead {
		ctx = forge.ContextWithUserName(ctx, user)
		for _, s := range testSearches {
			ents, _ := server.SearchEntries(ctx, s.path, s.query)
			got := make([]string, 0)
			for _, e := range ents {
				got = append(got, e.Path)
			}
			if len(got) != 0 {
				if len(got) == 1 && got[0] == "/" {
					// anyone can search root, even to an uninvited user
					continue
				}
				t.Fatalf("searched %q from %q: uninvited user shouldn't be able to search child entries, got: %v", s.query, s.path, got)
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

	// test user data
	ctx = forge.ContextWithUserName(ctx, "admin@imagvfx.com")
	for _, c := range userDataCases {
		err := server.SetUserData(ctx, c.user, c.section, c.key, c.value)
		if !equalError(c.wantErr, err) {
			t.Fatalf("add %q: want err %q, got %q", c.label, errorString(c.wantErr), errorString(err))
		}
		if c.wantErr != nil {
			continue
		}
		value, err := server.GetUserData(ctx, c.user, c.section, c.key)
		if err != nil {
			t.Fatalf("get %q: %v", c.label, err)
		}
		if value != c.value {
			t.Fatalf("get %q: want %q, got %q", c.label, c.value, value)
		}
		err = server.SetUserData(ctx, c.user, c.section, c.key, "")
		if err != nil {
			t.Fatalf("update %q: %v", c.label, err)
		}
		value, err = server.GetUserData(ctx, c.user, c.section, c.key)
		if err != nil {
			t.Fatalf("get after update %q: %v", c.label, err)
		}
		if value != "" {
			t.Fatalf("get after update %q: want empty string, got %q", c.label, value)
		}
	}
	for _, c := range userDataCases {
		if c.wantErr != nil {
			continue
		}
		err = server.DeleteUserData(ctx, c.user, c.section, c.key)
		if err != nil {
			t.Fatalf("delete %q: %v", c.label, err)
		}
	}
	data, err := server.FindUserData(ctx, forge.UserDataFinder{User: "admin@imagvfx.com"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("find: want section length 0, got %v", len(data))
	}
}
