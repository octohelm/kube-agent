package idgen

import (
	"context"
	"net"
	"time"

	"github.com/go-courier/snowflakeid"
	"github.com/go-courier/snowflakeid/workeridutil"
)

var startTime, _ = time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
var sff = snowflakeid.NewSnowflakeFactory(16, 8, 5, startTime)

func FromIP(ip net.IP) (IDGen, error) {
	return sff.NewSnowflake(workeridutil.WorkerIDFromIP(ip))
}

type IDGen interface {
	ID() (uint64, error)
}

type contextKeyIDGen int

func WithIDGen(ctx context.Context, gen IDGen) context.Context {
	return context.WithValue(ctx, contextKeyIDGen(1), gen)
}

func FromContext(ctx context.Context) IDGen {
	if gen, ok := ctx.Value(contextKeyIDGen(1)).(IDGen); ok {
		return gen
	}
	return nil
}
