package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
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
		begin := time.Now()
		vcmap[vendor] = &VendorConfig{
			Name:    vendor,
			Levels:  []string{},
			Servers: *newServerMap(nil, "", vendor, false),
		}

		if Konf.Exists(vendor + ".levels") {
			vcmap[vendor].Levels = append(vcmap[vendor].Levels, Konf.Strings(vendor+".levels")...)
		}

		start := []string{vendor, "servers"}
		search := strings.Join(start, ".")
		resolve := Konf.Bool(vendor + ".resolve_servers")
		if len(vcmap[vendor].Levels) == 0 {
			vcmap[vendor].Servers.load_servers(search, "", resolve)
		} else {
			build_server_map(&vcmap[vendor].Servers, start, vcmap[vendor].Levels, resolve)
		}
		t := time.Now()
		log.Printf("Finished loading %s in %.2fsec", vendor, t.Sub(begin).Seconds())
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
func (sm *ServerMap) load_servers(search string, key string, resolve bool) {
	server_search := search

	/*
	 * if key is empty, then we don't want to add another level to sm, but rather
	 * we want to add items directly to sm
	 */
	if len(key) > 0 {
		server_search = fmt.Sprintf("%s.%s", search, key)
	}
	servers := Konf.Strings(server_search)
	if resolve {
		l := newServerMap(sm, key, sm.Vendor, true)
		for _, fqdn := range servers {
			svrs := []string{}
			if net.ParseIP(fqdn) == nil {
				// is a FQDN, so should be fqdn => [ip1, ip2]
				addrs, err := net.LookupHost(fqdn)
				if err != nil {
					log.Printf("Error resolving %s: %s", fqdn, err.Error())
					continue
				} else {
					svrs = append(svrs, addrs...)
				}
			} else {
				// is an IP address so should be => [ip1, ip2]
				ip := []string{fqdn}
				l.appendList(ip)
			}
			// if we have one or more servers, add a level
			if len(svrs) > 0 {
				l.addList(fqdn, svrs)
			}
		}
		// if we have a key add a level to sm
		if len(key) > 0 {
			sm.addMap(key, l)
		} else {
			// no key?  move the elments of l into sm
			sm.appendList(l.getList())
			for k, v := range l.getMap() {
				sm.addMap(k, &v)
			}
		}
	} else {
		// DNS resolution is off
		if len(key) > 0 {
			sm.addList(key, servers)
		} else {
			sm.appendList(servers)
		}
	}
}

/*
 * Recursive function to populate the ServerMap with the config
 * data from Viper
 */
func build_server_map(sm *ServerMap, location []string, levels []string, resolve bool) {
	level_cnt := len(levels)
	search := strings.Join(location, ".")

	if level_cnt == 1 {
		for _, key := range Konf.MapKeys(search) {
			sm.load_servers(search, key, resolve)
		}
	} else {
		// pop off the next level
		levels := levels[1:len(levels)]
		// Iterate over our Config Viper map[string]inteface{}
		for _, key := range Konf.MapKeys(search) {
			loc := append(location, key)
			new_map := newServerMap(sm, key, sm.Vendor, false)
			// attach our new_map to ourself
			sm.addMap(key, new_map)
			// recurse
			build_server_map(new_map, loc, levels, resolve)
		}
	}
}
