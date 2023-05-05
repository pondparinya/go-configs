package configs

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

type confEntry struct {
	v *viper.Viper
}

func LoadConfigs(path, name string, cfg interface{}) error {
	entry, err := initiate(path, name)
	if err != nil {
		return err
	}

	if err := entry.bindEnvs(cfg); err != nil {
		return fmt.Errorf("failed to bind env variables : %+v", err)
	}

	if err := entry.v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal configs to struct : %+v", err)
	}

	// detect missing keys
	return entry.checkMissing()
}

func initiate(path, name string) (*confEntry, error) {
	v := viper.New()
	v.SetConfigName(name)
	v.AddConfigPath(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read configs : %+v", err)
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return &confEntry{v}, nil
}

func (e *confEntry) bindEnvs(cfg interface{}) error {
	if pt := reflect.TypeOf(cfg).Kind(); pt != reflect.Ptr {
		return fmt.Errorf("invalid type, should be pointer instead of %v", pt)
	}

	t := reflect.Indirect(reflect.ValueOf(cfg)).Type()
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("invalid type, should be struct instead of %v", t.Kind())
	}

	e.bindEnvsInternal(t)
	return nil
}

func (e *confEntry) bindEnvsInternal(ptype reflect.Type, parts ...string) {
	for i := 0; i < ptype.NumField(); i++ {
		field := ptype.Field(i)
		newParts := make([]string, len(parts))

		tv, ok := field.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}

		copy(newParts, parts)
		if tv != ",squash" {
			newParts = append(newParts, tv)
		}

		switch field.Type.Kind() {
		case reflect.Struct:
			e.bindEnvsInternal(field.Type, newParts...)
		default:
			_ = e.v.BindEnv(strings.Join(newParts, "."))
		}
	}
}
func (e *confEntry) checkMissing() error {
	var missingKeys []string
	keys := e.v.AllKeys()
	for _, v := range keys {
		if e.v.Get(v) == nil {
			missingKeys = append(missingKeys, strings.Replace(v, ".", "_", -1))
		}
	}

	if len(missingKeys) > 0 {
		sort.Strings(missingKeys)
		return fmt.Errorf("missing env: %v", strings.Join(missingKeys, ","))
	}

	return nil
}
