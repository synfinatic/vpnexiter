package vpnexiter

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type VendorConfig struct {
	Name     string
	Template string
	Levels   []string
	Servers  ServerMap
}

/*
 * Loads the vendor: map configuration
 */
func LoadVendors() map[string]*VendorConfig {
	vcmap := map[string]*VendorConfig{}

	for _, vendor := range Konf.Strings("vendors") {
		log.Printf("Loading: %s", vendor)

		vcmap[vendor] = &VendorConfig{
			Name:    vendor,
			Levels:  []string{},
			Servers: *newServerMap(nil, "", vendor, false),
		}

		if Konf.Exists(vendor + ".levels") {
			vcmap[vendor].Levels = append(vcmap[vendor].Levels, Konf.Strings(vendor+".levels")...)
		}

		start := []string{vendor, "servers"}
		if len(vcmap[vendor].Levels) == 0 {
			search := strings.Join(start, ".")
			vcmap[vendor].Servers.appendList(Konf.Strings(search))
		} else {
			resolve := Konf.Bool(vendor + ".resolve_servers")
			build_server_map(&vcmap[vendor].Servers, vendor, start, vcmap[vendor].Levels, resolve)
		}
	}
	return vcmap
}

/*
 * helper for build_server_map().  Was intended to execute as a go-routine to
 * speed up DNS queries, but turns out that doesn't work on OSX because
 * net.LookupHost() is just a proxy for gethostbyname() which is not re-entrant.
 *
 * More info: https://golang.org/pkg/net/#hdr-Name_Resolution
 */
func (sm *ServerMap) load_servers(vendor string, search string, key string, resolve bool) {
	server_search := fmt.Sprintf("%s.%s", search, key)
	servers := Konf.Strings(server_search)
	if resolve {
		l := newServerMap(sm, key, vendor, true)
		for _, fqdn := range servers {
			svrs := []string{}
			if net.ParseIP(fqdn) == nil {
				addrs, err := net.LookupHost(fqdn)
				if err != nil {
					log.Printf("Error resolving %s: %s", fqdn, err.Error())
					continue
				} else {
					svrs = append(svrs, addrs...)
				}
			} else {
				svrs = append(svrs, fqdn) // just an IP
			}
			l.addList(fqdn, svrs)
		}
		err := sm.addMap(key, l)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := sm.addList(key, servers)
		if err != nil {
			log.Fatal(err)
		}
	}
}

/*
 * Recursive function to populate the ServerMap with the config
 * data from Viper
 */
func build_server_map(sm *ServerMap, vendor string, location []string, levels []string, resolve bool) {
	level_cnt := len(levels)
	search := strings.Join(location, ".")

	if level_cnt == 1 {
		for _, key := range Konf.MapKeys(search) {
			sm.load_servers(vendor, search, key, resolve)
		}
	} else {
		// pop off the next level
		levels := levels[1:len(levels)]
		// Iterate over our Config Viper map[string]inteface{}
		for _, key := range Konf.MapKeys(search) {
			loc := append(location, key)
			new_map := newServerMap(sm, key, vendor, false)
			// attach our new_map to ourself
			sm.addMap(key, new_map)
			// recurse
			build_server_map(new_map, vendor, loc, levels, resolve)
		}
	}
}
