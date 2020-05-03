package vpnexiter

import (
	"log"
	"net"
)

type ServerList struct {
	Name string
	IPs  map[string][]string
}

func Server2ServerList(vendor string, path []string) (*ServerList, error) {
	slist := ServerList{}
	slist.IPs = make(map[string][]string)
	name, err := GetPath(vendor, path)
	if err != nil {
		return nil, err
	}
	slist.Name = name
	servers, err := GetServers(vendor, path)
	if err != nil {
		return nil, err
	}

	for _, s := range servers {
		ips, err := net.LookupHost(s)
		if err != nil {
			log.Printf("Unable to resolve: %s", s)
			continue
		}
		slist.IPs[s] = ips
	}
	return &slist, nil
}
