package jwtutil

import (
	"testing"
)

func TestAuthorizations(t *testing.T) {
	auths := Authorizations{}

	auths.Add("Bearer", "xxxxx")
	auths.Add("WechatBearer", "yyyyy")

	t.Log(auths.String())
}
