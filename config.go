package forge

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml"
)

type Config struct {
	Struct StructConfig
}

func NewConfig() *Config {
	c := &Config{
		Struct: StructConfig{},
	}
	return c
}

type StructConfig map[string]*EntryStruct

type EntryStruct struct {
	Type          string
	Properties    []KeyTypeValue
	Environs      []KeyTypeValue
	SubEntryTypes []string
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
	s.Properties = make([]KeyTypeValue, 0)
	s.Environs = make([]KeyTypeValue, 0)
	s.SubEntryTypes = make([]string, 0)
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
	return s, nil
}

func LoadConfig(configDir string) (*Config, error) {
	c := NewConfig()
	structToml := filepath.Join(configDir, "struct.toml")
	structConfig, err := toml.LoadFile(structToml)
	if err != nil {
		return nil, fmt.Errorf("failed to load %v: %v", structToml, err)
	}
	for _, k := range structConfig.Keys() {
		s, err := getEntryStruct(structConfig, k)
		if err != nil {
			return nil, fmt.Errorf("failed to load %v: %v", structToml, err)
		}
		c.Struct[s.Type] = s
	}
	if c.Struct["root"] == nil {
		return nil, fmt.Errorf("root entry structure is not defined")
	}
	return c, nil
}
