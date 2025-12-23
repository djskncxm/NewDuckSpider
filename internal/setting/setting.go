package setting

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

type Setting struct {
	Spider struct {
		Worker   int    `yaml:"Worker"`
		LOGLEVEL string `yaml:"LOGLEVEL"`
		Timeout  int    `yaml:"Timeout"`
		Retry    int    `yaml:"Retry"`
		Proxy    string `yaml:"Proxy"`
		Delay    int    `yaml:"Delay"`
	} `yaml:"Spider"`
	Headers struct {
		UserAgent string `yaml:"UserAgent"`
	} `yaml:"Headers"`
	Log struct {
		LogFile string `yaml:"LogFile"`
	} `yaml:"Log"`
}

type SettingsManager struct {
	mu       sync.RWMutex
	Settings map[string]string
}

func NewSettingsManager() *SettingsManager {
	return &SettingsManager{
		Settings: make(map[string]string),
	}
}

func (sm *SettingsManager) GetSetting(key string) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.Settings[key]
	return val, ok
}

func (sm *SettingsManager) SetSetting(key, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.Settings[key] = value
}

func (sm *SettingsManager) GetInt(key string, defaultVal int) int {
	val, ok := sm.GetSetting(key)
	if !ok {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

func (sm *SettingsManager) GetBool(key string, defaultVal bool) bool {
	val, ok := sm.GetSetting(key)
	if !ok {
		return defaultVal
	}
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return boolVal
}

// 递归加载结构体到 map[string]string
func (sm *SettingsManager) LoadFromSetting(s interface{}) {
	sm.loadStruct(reflect.ValueOf(s), "")
}

func (sm *SettingsManager) loadStruct(v reflect.Value, prefix string) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		val := v.Field(i)

		key := field.Name
		if prefix != "" {
			key = prefix + "." + key
		}

		switch val.Kind() {
		case reflect.Struct:
			sm.loadStruct(val, key) // 递归
		case reflect.String:
			sm.SetSetting(key, val.String())
		case reflect.Bool:
			sm.SetSetting(key, strconv.FormatBool(val.Bool()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			sm.SetSetting(key, strconv.FormatInt(val.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			sm.SetSetting(key, strconv.FormatUint(val.Uint(), 10))
		default:
			sm.SetSetting(key, fmt.Sprintf("%v", val.Interface()))
		}
	}
}
