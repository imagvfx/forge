package forge

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml"
)

type Config struct {
	UserdataRoot string
	Structs      []*EntryStruct
}

func NewConfig() *Config {
	c := &Config{
		Structs: make([]*EntryStruct, 0),
	}
	return c
}

type EntryStruct struct {
	Type          string
	SubEntryTypes []string
	SubEntries    []KeyTypeValue
	Properties    []KeyTypeValue
	Environs      []KeyTypeValue
}

type KeyTypeValue struct {
	Key   string
	Type  string
	Value string
}

type KeyValue struct {
	Key   string
	Value string
}

func getEntryStruct(t *toml.Tree, typ string) (*EntryStruct, error) {
	typTree := t.Get(typ)
	if typTree == nil {
		return nil, fmt.Errorf("unknown type: %v", typ)
	}
	s := &EntryStruct{}
	s.Type = typ
	s.SubEntryTypes = make([]string, 0)
	s.SubEntries = make([]KeyTypeValue, 0)
	s.Properties = make([]KeyTypeValue, 0)
	s.Environs = make([]KeyTypeValue, 0)
	subtypsTree := typTree.(*toml.Tree).GetArray("sub_entry_types")
	if subtypsTree != nil {
		subtyps, ok := subtypsTree.([]string)
		if !ok {
			return nil, fmt.Errorf("cannot convert [%v].sub_entry_types as []string", s.Type)
		}
		for _, t := range subtyps {
			if t == "root" {
				return nil, fmt.Errorf("root type cannot be refered in sub_entry_types: %v", s.Type)
			}
		}
		s.SubEntryTypes = subtyps
	}
	subEntsTree := typTree.(*toml.Tree).GetArray("sub_entries")
	if subEntsTree != nil {
		subEnts, ok := subEntsTree.([]string)
		if !ok {
			return nil, fmt.Errorf("cannot convert [%v].sub_entries as []string", s.Type)
		}
		for _, ent := range subEnts {
			k_t := strings.SplitN(ent, " ", 2)
			if len(k_t) != 2 {
				return nil, fmt.Errorf("sub_entries should have entry in a 'entry_name entry_type' form")
			}
			k := k_t[0]
			t := k_t[1]
			ktv := KeyTypeValue{
				Key:  k,
				Type: t,
			}
			s.SubEntries = append(s.SubEntries, ktv)
		}
	}
	propsTree := typTree.(*toml.Tree).GetArray("properties")
	if propsTree != nil {
		props, ok := propsTree.([]string)
		if !ok {
			return nil, fmt.Errorf("cannot convert [%v].properties as []string", s.Type)
		}
		for _, p := range props {
			var k, t, v string
			kt_v := strings.SplitN(p, "=", 2)
			if len(kt_v) == 2 {
				v = kt_v[1]
			}
			k_t := strings.SplitN(kt_v[0], " ", 2)
			k = k_t[0]
			if len(k_t) == 2 {
				t = k_t[1]
			}
			ktv := KeyTypeValue{
				Key:   k,
				Type:  t,
				Value: v,
			}
			s.Properties = append(s.Properties, ktv)
		}
	}
	envsTree := typTree.(*toml.Tree).GetArray("environs")
	if envsTree != nil {
		envs, ok := envsTree.([]string)
		if !ok {
			return nil, fmt.Errorf("cannot convert [%v].environs as []string", s.Type)
		}
		for _, e := range envs {
			var k, t, v string
			kt_v := strings.SplitN(e, "=", 2)
			if len(kt_v) == 2 {
				v = kt_v[1]
			}
			k_t := strings.SplitN(kt_v[0], " ", 2)
			k = k_t[0]
			if len(k_t) == 2 {
				t = k_t[1]
			}
			ktv := KeyTypeValue{
				Key:   k,
				Type:  t,
				Value: v,
			}
			s.Environs = append(s.Environs, ktv)
		}
	}
	return s, nil
}

// Match go-toml.Tree keys to the same order with the file.
// Would be great it can be done with go-toml package, but didn't find the way.
func orderedKeys(t *toml.Tree) []string {
	type keyPos struct {
		Key  string
		Line int
		Col  int
	}
	keys := t.Keys()
	poses := make([]keyPos, len(keys))
	for i, k := range keys {
		subt := t.Get(k).(*toml.Tree)
		p := keyPos{
			Key:  k,
			Line: subt.Position().Line,
			Col:  subt.Position().Col,
		}
		poses[i] = p
	}
	sort.Slice(poses, func(i, j int) bool {
		if poses[i].Line < poses[j].Line {
			return true
		}
		if poses[i].Line > poses[j].Line {
			return false
		}
		if poses[i].Col < poses[j].Col {
			return true
		}
		return false
	})
	ordkeys := make([]string, len(poses))
	for i, p := range poses {
		ordkeys[i] = p.Key
	}
	return ordkeys
}

func LoadConfig(configDir string) (*Config, error) {
	c := NewConfig()
	forgeToml := filepath.Join(configDir, "forge.toml")
	forgeConfig, err := toml.LoadFile(forgeToml)
	if err != nil {
		return nil, fmt.Errorf("failed to load %v: %v", forgeToml, err)
	}
	userdataRoot, ok := forgeConfig.Get("userdata_root").(string)
	if !ok {
		return nil, fmt.Errorf("%v: cannot convert userdata_root value as string", forgeToml)
	}
	c.UserdataRoot = userdataRoot
	structToml := filepath.Join(configDir, "struct.toml")
	structConfig, err := toml.LoadFile(structToml)
	if err != nil {
		return nil, fmt.Errorf("failed to load %v: %v", structToml, err)
	}
	for _, k := range orderedKeys(structConfig) {
		s, err := getEntryStruct(structConfig, k)
		if err != nil {
			return nil, fmt.Errorf("failed to load %v: %v", structToml, err)
		}
		c.Structs = append(c.Structs, s)
	}
	if len(c.Structs) == 0 || c.Structs[0].Type != "root" {
		return nil, fmt.Errorf("root entry structure is not defined")
	}
	return c, nil
}
