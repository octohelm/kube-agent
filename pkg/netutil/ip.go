package netutil

import (
	"net"
	"os"
)

func ExposedIP() net.IP {
	hostname, ok := os.LookupEnv("HOSTNAME")
	if !ok {
		hostname = "localhost"
	}

	addrList, err := net.LookupIP(hostname)
	if err != nil {
		panic(err)
	}

	for i := range addrList {
		if ipv4 := addrList[i].To4(); ipv4 != nil {
			return ipv4
		}
	}
	return nil
}
