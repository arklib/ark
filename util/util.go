package util

import (
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var fnNameRE = regexp.MustCompile(`\.(\w+)`)

func GetFnName(handler any) string {
	pointer := reflect.ValueOf(handler).Pointer()
	name := runtime.FuncForPC(pointer).Name()
	if name == "" {
		return ""
	}
	return fnNameRE.FindString(name)[1:]
}

func ForEachMapBySort[V any](in map[string]V, iteratee func(key string, value V)) {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		iteratee(key, in[key])
	}
}

func MakeStrKey(keys ...any) string {
	var newKeys = make([]string, len(keys))
	for _, key := range keys {
		switch k := key.(type) {
		case string:
			newKeys = append(newKeys, k)
		case uint:
		case uint8:
		case uint16:
		case uint32:
		case uint64:
			newKeys = append(newKeys, strconv.FormatUint(uint64(k), 10))
		default:
			return ""
		}
	}
	return strings.Join(newKeys, ":")
}
