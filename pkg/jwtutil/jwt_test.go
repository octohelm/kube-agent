package jwtutil

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/lestrrat-go/jwx/jwk"
)

func Test(t *testing.T) {
	ks := NewKeySet(func(ctx context.Context) (jwk.Set, error) {
		return jwk.Parse([]byte(`{"keys":[{"alg":"RS256","e":"AQAB","kid":"CEhTfFd1unw","kty":"RSA","n":"scIRD_Fj9D7AlixEYuwPGvVqHs-r7tJ7-BFy08UNFmrRT_4ySr_FPdfwlTnN7AH8um1_8UqtUyvBwvvWtn5gOmVgxRlTttueU5ZKB_HSoEbuqZfu1umKyO2Y-9ElYSrrEfkDSp8tR-NSP1AWQvkbkGej-RDuyDWaHVu_zhTfz2s"}]}`))
	})
	tok, err := ks.Validate(context.Background(), "eyJhbGciOiJSUzI1NiIsImtpZCI6IkNFaFRmRmQxdW53IiwidHlwIjoiSldUIn0.eyJhdWQiOlsiaHctZGV2Il0sImV4cCI6MTYyMjg4NTE1MCwiaWF0IjoxNjIwMjkzMTUwLCJpc3MiOiJvY3RvaGVsbS50ZWNoIiwianRpIjoiMTQyNDU5ODcyMjk3MDkxMzc3Iiwic3ViIjoiS1VCRV9BR0VOVCJ9.S--kdlu5tFAMMfL1Ofmnfzlcv8D2-zK9VWkpVyr6bgnA8uPNlalRRLvzISIomTDpybuNEeCSui_p6CrOgEBzmaGgVk-EnIUl300V5FUDIPw2mglRJC3ZGs9jMlzqd6qCGDYM67N-7x1PK46Vj6aqMyZlevwXD7xhUvuYGNjSngA")

	spew.Dump(err)
	spew.Dump(tok.Issuer())
	spew.Dump(tok.Subject())
}
