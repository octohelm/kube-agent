package jwtutil

import (
	"bytes"
	"net/http"
)

func ParseAuthorization(s string) (auths Authorizations) {
	auths = Authorizations{}
	if len(s) == 0 {
		return
	}
	tokens := bytes.Split([]byte(s), []byte(";"))
	for _, token := range tokens {
		kv := bytes.Split(bytes.TrimSpace(token), []byte(" "))
		v := ""
		if len(kv) == 2 {
			v = string(bytes.TrimSpace(kv[1]))
		}
		auths[http.CanonicalHeaderKey(string(bytes.TrimSpace(kv[0])))] = v
	}
	return
}

type Authorizations map[string]string

func (auths Authorizations) Add(k string, v string) {
	auths[http.CanonicalHeaderKey(k)] = v
}

func (auths Authorizations) Get(k string) string {
	if v, ok := auths[http.CanonicalHeaderKey(k)]; ok {
		return v
	}
	return ""
}

func (auths Authorizations) String() string {
	buf := bytes.Buffer{}

	count := 0
	for tpe, token := range auths {
		if count > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(http.CanonicalHeaderKey(tpe))
		buf.WriteString(" ")
		buf.WriteString(token)
		count++
	}
	return buf.String()
}
