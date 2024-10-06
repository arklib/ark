package util

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cast"
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
		newKeys = append(newKeys, cast.ToString(key))
	}
	return strings.Join(newKeys, ":")
}

func SplitSuffix(s, sep string) ([]string, string) {
	paths := strings.Split(s, sep)
	name := paths[len(paths)-1]
	return paths, name
}

func ExecTime(name string, f func()) {
	start := time.Now()
	f()
	elapsed := time.Since(start)
	fmt.Printf("[%s] exec.time: %v\n", name, elapsed)
}
