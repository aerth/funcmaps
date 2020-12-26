package funcmaps

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type FuncMap map[string]interface{}

func Default() FuncMap {
	return FuncMap{
		"toLower":     strings.ToLower,
		"toUpper":     strings.ToUpper,
		"toTitle":     strings.ToTitle,
		"string":      func(v interface{}) string { return fmt.Sprintf("%v", v) },
		"trim":        func(c, s string) string { return strings.Trim(s, c) },
		"trimspace":   strings.TrimSpace,
		"trim_left":   func(c, s string) string { return strings.TrimLeft(s, c) },
		"trim_right":  func(c, s string) string { return strings.TrimRight(s, c) },
		"trim_prefix": func(c, s string) string { return strings.TrimPrefix(s, c) },
		"trim_suffix": func(c, s string) string { return strings.TrimSuffix(s, c) },
		"title":       strings.Title,
		"fields":      strings.Fields,
		"wc":          func(s string) int { return len(strings.Fields(s)) },
		"has_prefix":  func(c, s string) bool { return strings.HasPrefix(s, c) },
		"has_suffix":  func(c, s string) bool { return strings.HasSuffix(s, c) },
		"replace":     func(old, new string, n int, s string) string { return strings.Replace(s, old, new, n) },
		"replace_all": func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
		"count":       func(sub, s string) int { return strings.Count(s, sub) },
		"split":       func(sep, s string) []string { return strings.Split(s, sep) },
		"split_n":     func(sep string, n int, s string) []string { return strings.SplitN(s, sep, n) },
		"backtick":    func(s interface{}) string { return fmt.Sprintf("`%v`", s) },
		"backticks":   func(lang string, s interface{}) string { return fmt.Sprintf("```%s\n%v\n```", lang, s) },
		"date":        FormatTime,
		"contains":    strings.Contains,
		"now":         time.Now,
		"NOW":         func() string { return time.Now().String() },
		"json": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
		"prettyjson": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "  ")
			return string(a)
		},
		"indent": func(s, prefix string) string {
			lines := strings.Split(s, "\n")
			for idx, line := range lines {
				lines[idx] = prefix + line
			}
			return strings.Join(lines, "\n")
		},
		// yaml
		// xml
		// toml
		"join": strings.Join,
		"unexport": func(input string) string {
			return fmt.Sprintf("%s%s", strings.ToLower(input[0:1]), input[1:])
		},
		"add":   func(a, b int) int { return a + b },
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"rev": func(v interface{}) string {
			runes := []rune(v.(string))
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return string(runes)
		},
		"int": func(v interface{}) string {
			a, err := strconv.Atoi(v.(string))
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return fmt.Sprintf("%d", a)
		},
		"is_true":    IsTrue,
		"is_empty":   IsEmpty,
		"is_default": IsDefault,
		"yesno":      YesNo,
		"ternary":    YesNo,
		"coalesce":   Coalesce,
		"env":        os.Getenv,
		"has":        Has,
		"has_any":    HasAny,
		"file_size":  FileSizeFormat,
		"uuid":       UUID,
		"repeat":     Repeat,
		"join2":      Join2,
		"eq_any":     EqualAny,
		"deep_eq":    reflect.DeepEqual,
		"map":        Map,
	}
}

func Combined(fs ...FuncMap) FuncMap {
	m := FuncMap{}
	for _, fm := range fs {
		for k, v := range fm {
			m[k] = v
		}
	}
	return m
}
