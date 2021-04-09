package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
)

func RequestAttributesFromRequest(r *http.Request, prefixes ...string) (authorizer.Attributes, error) {
	prefixApis := strings.Join(append(prefixes, "apis"), "/")
	prefixApi := strings.Join(append(prefixes, "api"), "/")

	rif := &apirequest.RequestInfoFactory{
		APIPrefixes:          sets.NewString(prefixApis, prefixApi),
		GrouplessAPIPrefixes: sets.NewString(prefixApi),
	}

	ri, err := rif.NewRequestInfo(r)
	if err != nil {
		return nil, err
	}

	return &RequestInfoAttrs{RequestInfo: *ri}, nil
}

type PolicyRule = rbacv1.PolicyRule

type Scope struct {
	Namespaces []string            `json:"namespaces,omitempty"`
	Rules      []rbacv1.PolicyRule `json:"rules"`
}

type Scopes map[string]Scope

func ScopesFromMap(m map[string]interface{}) Scopes {
	s := Scopes{}
	b := bytes.NewBuffer(nil)
	_ = json.NewEncoder(b).Encode(m)
	_ = json.NewDecoder(b).Decode(&s)
	return s
}
