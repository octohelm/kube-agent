package jwtutil

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

var (
	ErrKeyIDNotFound = errors.New("key id not exists")
	ErrTokenExpired  = errors.New("token is expired")
)

func NewKeySet(fetchSet FetchSet) *KeySet {
	return &KeySet{
		Set:      jwk.NewSet(),
		fetchSet: fetchSet,
	}
}

type FetchSet = func(ctx context.Context) (jwk.Set, error)

func SyncRemote(remote string) FetchSet {
	return func(ctx context.Context) (jwk.Set, error) {
		return jwk.Fetch(context.Background(), remote)
	}
}

type KeySet struct {
	jwk.Set
	fetchSet FetchSet
}

func (c *KeySet) LookupKeyID(s string) (jwk.Key, bool) {
	if key, ok := c.Set.LookupKeyID(s); ok {
		return key, ok
	}

	if err := c.Sync(context.Background()); err != nil {
		return nil, false
	}

	return c.Set.LookupKeyID(s)
}

func (c *KeySet) Sync(ctx context.Context) error {
	s, err := c.fetchSet(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < s.Len(); i++ {
		if k, ok := s.Get(i); ok {
			c.Add(k)
		}
	}

	return nil
}

func (c *KeySet) Validate(ctx context.Context, tokenStr string) (jwt.Token, error) {
	tok, err := c.validate(tokenStr)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func (c *KeySet) validate(tokenStr string) (jwt.Token, error) {
	tok, err := jwt.ParseString(tokenStr, jwt.WithKeySet(c))
	if err != nil {
		fmt.Printf("%+v\n", err)
		return nil, err
	}
	if time.Until(tok.Expiration()) < 0 {
		return nil, ErrTokenExpired
	}
	return tok, nil
}
