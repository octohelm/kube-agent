package auth

import (
	"net/http"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func attr(method string, p string) authorizer.Attributes {
	a, _ := RequestAttributesFromRequest(&http.Request{
		Method: method,
		URL: &url.URL{
			Path: p,
		},
	})
	return a
}

func TestRuleMatches(t *testing.T) {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Verbs:     []string{"list"},
			Resources: []string{"*"},
		},
		{
			APIGroups:     []string{""},
			Verbs:         []string{"get"},
			Resources:     []string{"*"},
			ResourceNames: []string{"a"},
		},
	}

	NewWithT(t).Expect(RulesAllow(attr(http.MethodGet, "/api/v1/namespaces/default/pods"), rules...)).To(BeTrue())
	NewWithT(t).Expect(RulesAllow(attr(http.MethodGet, "/api/v1/namespaces/default/pods/a/logs"), rules...)).To(BeTrue())
	NewWithT(t).Expect(RulesAllow(attr(http.MethodGet, "/api/v1/namespaces/default/pods/b/logs"), rules...)).To(BeFalse())

	NewWithT(t).Expect(RulesAllow(attr(http.MethodGet, "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews"), rules...)).To(BeFalse())
	NewWithT(t).Expect(RulesAllow(attr(http.MethodPost, "/api/v1/namespaces/default/pods"), rules...)).To(BeFalse())
}
