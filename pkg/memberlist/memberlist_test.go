package memberlist

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/octohelm/kube-agent/pkg/netutil"
)

func Test(t *testing.T) {
	ip := netutil.ExposedIP()

	ports := []int{3456, 3457, 3458}

	for i := range ports {
		go func(port int) {
			opt := Member{
				Name:     fmt.Sprintf("%d", port),
				BindIP:   ip,
				BindPort: port,
			}

			ml := NewMemberList(opt, []string{"localhost:3456"})

			go func() {
				time.Sleep(200 * time.Millisecond)

				fmt.Println(ml.Members())
			}()

			if err := ml.Serve(context.Background()); err != nil {
				fmt.Println(err)
			}
		}(ports[i])
	}

	time.Sleep(500 * time.Millisecond)
}
