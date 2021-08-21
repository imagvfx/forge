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
	Type       string
	SubEntries []KeyTypeValue
	Properties []KeyTypeValue
	Environs   []KeyTypeValue
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
	defaultMap := map[string][]KeyTypeValue{
		"sub_entries": make([]KeyTypeValue, 0),
		"properties":  make([]KeyTypeValue, 0),
		"environs":    make([]KeyTypeValue, 0),
	}
	for def, ktvs := range defaultMap {
		defTree := typTree.(*toml.Tree).GetArray(def)
		if defTree == nil {
			continue
		}
		raws, ok := defTree.([]string)
		if !ok {
			return nil, fmt.Errorf("cannot convert [%v].%v as []string", typ, def)
		}
		for _, r := range raws {
			var k, t, v string
			kt_v := strings.SplitN(r, "=", 2)
			if len(kt_v) == 2 {
				v = strings.TrimSpace(kt_v[1])
			}
			k_t := strings.Fields(kt_v[0])
			if len(k_t) != 2 {
				return nil, fmt.Errorf("invalid key, type, value definition: %v", r)
			}
			k = k_t[0]
			t = k_t[1]
			ktv := KeyTypeValue{
				Key:   k,
				Type:  t,
				Value: v,
			}
			ktvs = append(ktvs, ktv)
		}
		defaultMap[def] = ktvs
	}
	s := &EntryStruct{}
	s.Type = typ
	s.SubEntries = defaultMap["sub_entries"]
	s.Properties = defaultMap["properties"]
	s.Environs = defaultMap["environs"]
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
	entryToml := filepath.Join(configDir, "entry.toml")
	entryConfig, err := toml.LoadFile(entryToml)
	if err != nil {
		return nil, fmt.Errorf("failed to load %v: %v", entryToml, err)
	}
	for _, k := range orderedKeys(entryConfig) {
		s, err := getEntryStruct(entryConfig, k)
		if err != nil {
			return nil, fmt.Errorf("failed to load %v: %v", entryToml, err)
		}
		c.Structs = append(c.Structs, s)
	}
	if len(c.Structs) == 0 || c.Structs[0].Type != "root" {
		return nil, fmt.Errorf("root entry structure is not defined")
	}
	return c, nil
}
