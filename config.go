package forge

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Forge     *ForgeConfig
	EntryType *EntryTypeConfig
}

type ForgeConfig struct {
	UserdataRoot string `toml:"userdata_root"`
}

type EntryTypeConfig struct {
	Types []EntryTypeInfo
}

type EntryTypeInfo struct {
	Name       string
	SubEntries []KeyTypeValue `toml:"sub_entries"`
	Properties []KeyTypeValue
	Environs   []KeyTypeValue
}

type KeyTypeValue struct {
	Key   string
	Type  string
	Value string
}

func (ktv *KeyTypeValue) UnmarshalText(text []byte) error {
	raw := string(text)
	var k, t, v string
	kt_v := strings.SplitN(raw, "=", 2)
	if len(kt_v) == 2 {
		v = strings.TrimSpace(kt_v[1])
	}
	k_t := strings.Fields(kt_v[0])
	if len(k_t) != 2 {
		return fmt.Errorf("invalid key, type, value definition: %v", raw)
	}
	k = k_t[0]
	t = k_t[1]
	*ktv = KeyTypeValue{
		Key:   k,
		Type:  t,
		Value: v,
	}
	return nil
}

type KeyValue struct {
	Key   string
	Value string
}

func LoadConfig(configDir string) (*Config, error) {
	c := &Config{}
	forgeToml := filepath.Join(configDir, "forge.toml")
	_, err := toml.DecodeFile(forgeToml, &c.Forge)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %v: %v", forgeToml, err)
	}
	entryToml := filepath.Join(configDir, "entry.toml")
	_, err = toml.DecodeFile(entryToml, &c.EntryType)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %v: %v", entryToml, err)
	}
	if len(c.EntryType.Types) == 0 || c.EntryType.Types[0].Name != "root" {
		return nil, fmt.Errorf("failed to decode %v: root entry type is not defined", entryToml)
	}
	return c, nil
}
