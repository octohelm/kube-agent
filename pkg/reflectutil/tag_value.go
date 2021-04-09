package reflectutil

import (
	"bytes"
	"encoding/csv"
	"strings"
)

func NewTagValue(tag string) *TagValue {
	i := strings.Index(tag, ",")

	if i == -1 {
		return &TagValue{name: tag}
	}

	tv := &TagValue{name: tag[0:i], flags: map[string][]string{}}

	if i < len(tag)-1 {
		flags := tag[i+1:]

		if len(flags) > 0 {
			r := csv.NewReader(bytes.NewBufferString(flags))
			list, err := r.Read()
			if err == nil {
				for i := range list {
					flag := list[i]

					j := strings.Index(flag, "=")
					if j == -1 {
						tv.AddFlag(flag)
					} else {
						k := flag[0:j]

						if j+1 < len(flag) {
							tv.AddFlag(k, flag[j+1:])
						} else {
							tv.AddFlag(k)
						}
					}
				}
			}
		}

	}

	return tv
}

type TagValue struct {
	name  string
	flags map[string][]string
}

func (t *TagValue) Name() (string, bool) {
	return t.name, t.name != ""
}

func (t *TagValue) IsIgnore() bool {
	return t.name == "-"
}

func (t *TagValue) LookupFlag(n string) (string, bool) {
	v, ok := t.flags[n]
	if len(v) > 0 {
		return v[0], ok
	}
	return "", ok
}

func (t *TagValue) Flags() map[string][]string {
	return t.flags
}

func (t *TagValue) AddFlag(k string, values ...string) {
	t.flags[k] = append(t.flags[k], values...)
}
