package memberlist

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/memberlist"
)

func NewMemberList(m Member, seeds []string) *MemberList {
	return &MemberList{Member: m, seeds: seeds}
}

type Member struct {
	Name     string
	BindIP   net.IP
	BindPort int
}

func (m Member) Addr() string {
	return fmt.Sprintf("%s:%d", m.BindIP, m.BindPort)
}

type MemberList struct {
	Member
	seeds    []string
	list     *memberlist.Memberlist
	nodeLock sync.RWMutex
}

func (l *MemberList) SetMemberList(list *memberlist.Memberlist) {
	l.nodeLock.Lock()
	defer l.nodeLock.Unlock()

	l.list = list
}

func (l *MemberList) Members() (list []string) {
	l.nodeLock.Lock()
	defer l.nodeLock.Unlock()

	members := l.list.Members()
	for i := range members {
		list = append(list, members[i].Name)
	}
	return
}

func (l *MemberList) Serve(ctx context.Context) error {
	c := memberlist.DefaultLocalConfig()

	c.Name = l.Member.Name
	c.BindPort = l.Member.BindPort
	c.AdvertisePort = l.Member.BindPort
	c.LogOutput = io.Discard

	list, err := memberlist.Create(c)
	if err != nil {
		return err
	}

	l.SetMemberList(list)

	go func() {
		for {
			//try to join member until ready
			if _, err := l.list.Join(append(l.seeds, l.Member.Addr())); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			return
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	<-stopCh

	timeout := 5 * time.Second
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return list.Shutdown()
}
