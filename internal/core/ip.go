package core

import (
	"net"
	"sort"
	"strings"
)

// AllIPAddresses returns all non-loopback IPv4 addresses.
// It prioritizes local network addresses starting with "192".
func AllIPAddresses() (ips []string, err error) {
	var (
		ifaces []net.Interface
	)

	ifaces, err = net.Interfaces()
	if err != nil {
		return []string{"127.0.0.1"}, err
	}

	for _, iface := range ifaces {
		var (
			addrs []net.Addr
		)

		// skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err = iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// only IPv4
			if ip == nil || ip.To4() == nil {
				continue
			}

			ips = append(ips, ip.String())
		}
	}

	// sort
	sort.SliceStable(ips, func(i, j int) bool {
		ip1IsLocal := strings.HasPrefix(ips[i], "192.")
		ip2IsLocal := strings.HasPrefix(ips[j], "192.")

		if ip1IsLocal && !ip2IsLocal {
			return true
		}
		if !ip1IsLocal && ip2IsLocal {
			return false
		}
		return ips[i] < ips[j]
	})

	// 127.0.0.1 as default
	if len(ips) == 0 {
		ips = append(ips, "127.0.0.1")
	}

	return ips, nil
}
