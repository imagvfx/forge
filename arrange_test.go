package forge

import (
	"reflect"
	"testing"
)

func TestArrangeString(t *testing.T) {
	defaultKey := func(a string) string { return a }
	cases := []struct {
		label string
		elems []string
		el    string
		idx   int
		key   func(a string) string
		want  []string
	}{
		{
			label: "insert a to nil",
			elems: nil,
			el:    "a",
			idx:   0,
			key:   defaultKey,
			want:  []string{"a"},
		},
		{
			label: "remove a from nil",
			elems: nil,
			el:    "a",
			idx:   -1,
			key:   defaultKey,
			want:  nil,
		},
		{
			label: "remove a",
			elems: []string{"a", "b", "c"},
			el:    "a",
			idx:   -1,
			key:   defaultKey,
			want:  []string{"b", "c"},
		},
		{
			label: "insert a at 0",
			elems: []string{"a", "b", "c"},
			el:    "a",
			idx:   0,
			key:   defaultKey,
			want:  []string{"a", "b", "c"},
		},
		{
			label: "insert a at 1",
			elems: []string{"a", "b", "c"},
			el:    "a",
			idx:   1,
			key:   defaultKey,
			want:  []string{"b", "a", "c"},
		},
		{
			label: "insert a at 2",
			elems: []string{"a", "b", "c"},
			el:    "a",
			idx:   2,
			key:   defaultKey,
			want:  []string{"b", "c", "a"},
		},
		{
			label: "insert a at 3",
			elems: []string{"a", "b", "c"},
			el:    "a",
			idx:   3,
			key:   defaultKey,
			want:  []string{"b", "c", "a"},
		},
		{
			label: "remove d",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   -1,
			key:   defaultKey,
			want:  []string{"a", "b", "c"},
		},
		{
			label: "insert d at 0",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   0,
			key:   defaultKey,
			want:  []string{"d", "a", "b", "c"},
		},
		{
			label: "insert d at 1",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   1,
			key:   defaultKey,
			want:  []string{"a", "d", "b", "c"},
		},
		{
			label: "insert d at 2",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   2,
			key:   defaultKey,
			want:  []string{"a", "b", "d", "c"},
		},
		{
			label: "insert d at 3",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   3,
			key:   defaultKey,
			want:  []string{"a", "b", "c", "d"},
		},
		{
			label: "insert d at 4",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   4,
			key:   defaultKey,
			want:  []string{"a", "b", "c", "d"},
		},
		{
			label: "nil key",
			elems: []string{"a", "b", "c"},
			el:    "d",
			idx:   4,
			key:   nil,
			want:  []string{"a", "b", "c"},
		},
	}
	for _, c := range cases {
		got := Arrange(c.elems, c.el, c.idx, c.key, false)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%v: got %v, want %v", c.label, got, c.want)
		}
	}
}

func TestArrangeStringKV(t *testing.T) {
	key := func(a StringKV) string {
		return a.K
	}
	cases := []struct {
		label    string
		elems    []StringKV
		el       StringKV
		idx      int
		override bool
		want     []StringKV
	}{
		{
			label: "move {a} to 0",
			elems: []StringKV{
				{K: "a", V: "apple"},
				{K: "b", V: "banana"},
			},
			el:       StringKV{K: "a"},
			idx:      0,
			override: false,
			want: []StringKV{
				{K: "a", V: "apple"},
				{K: "b", V: "banana"},
			},
		},
		{
			label: "move {a} to 1",
			elems: []StringKV{
				{K: "a", V: "apple"},
				{K: "b", V: "banana"},
			},
			el:       StringKV{K: "a"},
			idx:      1,
			override: false,
			want: []StringKV{
				{K: "b", V: "banana"},
				{K: "a", V: "apple"},
			},
		},
		{
			label: "insert {a:alena} at 0",
			elems: []StringKV{
				{K: "a", V: "apple"},
				{K: "b", V: "banana"},
			},
			el:       StringKV{K: "a", V: "alena"},
			idx:      0,
			override: true,
			want: []StringKV{
				{K: "a", V: "alena"},
				{K: "b", V: "banana"},
			},
		},
		{
			label: "insert {a:alena} at 1",
			elems: []StringKV{
				{K: "a", V: "apple"},
				{K: "b", V: "banana"},
			},
			el:       StringKV{K: "a", V: "alena"},
			idx:      1,
			override: true,
			want: []StringKV{
				{K: "b", V: "banana"},
				{K: "a", V: "alena"},
			},
		},
	}
	for _, c := range cases {
		got := Arrange(c.elems, c.el, c.idx, key, c.override)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%v: got %v, want %v", c.label, got, c.want)
		}
	}
}
