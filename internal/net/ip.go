package net

import (
	"errors"
	"net"
)

// HostnameToIP4 - transforms a hostname to IP4
func HostnameToIP4(hostname string) (string, error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return "", nil
	}

	for _, ip := range ips {
		ip4 := ip.To4()
		if ip4 != nil {
			return ip4.String(), nil
		}
	}

	return "", errors.New("hostname cannot be resolved")
}
