package vpnexiter

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"strings"
)

type VendorConfig struct {
	Name     string
	Template string
	Levels   []string
	Servers  ServerMap
}

type ServerMap struct {
	L0 []string
	L1 map[string][]string
	L2 map[string]map[string][]string
	L3 map[string]map[string]map[string][]string
	L4 map[string]map[string]map[string]map[string][]string
	L5 map[string]map[string]map[string]map[string]map[string][]string
}

/*
 * Loads the vendor: map configuration
 */
func LoadVendors() map[string]*VendorConfig {
	v := viper.GetViper()
	vcmap := map[string]*VendorConfig{}

	for _, vendor := range v.GetStringSlice("vendors") {
		log.Printf("loading %s", vendor)
		vcmap[vendor] = &VendorConfig{
			Name:   vendor,
			Levels: []string{},
			Servers: ServerMap{
				L0: []string{},
				L1: map[string][]string{},
			},
		}
		if v.IsSet(vendor + ".levels") {
			vcmap[vendor].Levels = v.GetStringSlice(vendor + ".levels")
		}
		vcmap[vendor].Servers.build_server_map(vendor, vcmap[vendor].Levels)
		log.Printf("vcmap for %s = %s", vendor, vcmap[vendor].Servers)
	}
	return vcmap
}

/*
 * Loads the vendor.servers into a ServerMap
 */
func (sm *ServerMap) build_server_map(vendor string, levels []string) {
	level_cnt := len(levels)
	start := []string{vendor, "servers"}
	sm.get_servers(start, 0, level_cnt)
}

/*
 * Recursive helper for build_server_map() to walk the tree
 */
func (sm *ServerMap) get_servers(location []string, level int, max_levels int) {
	v := viper.GetViper()
	search := strings.Join(location, ".")
	if level < max_levels {
		log.Printf("processing %s", search)
		for k, _ := range v.GetStringMap(search) {
			// recurse further
			loc := append(location, k)
			sm.get_servers(loc, level+1, max_levels)
		}
	} else {
		// End of the line or when len(levels) == 0
		log.Printf("assigning %s", search)
		key := strings.Join(location[2:], "$")
		sm.set_level(max_levels, key, v.GetStringSlice(search))
	}
}

/*
 * Actually does the work of setting our values in the ServerMap
 */
func (sm *ServerMap) set_level(max_levels int, key string, values []string) error {
	switch max_levels {
	case 0:
		// max_levels = 0 is a special case
		log.Printf("Setting sm.L0 = %s", values)
		sm.L0 = values
		return nil
	case 1, 2, 3, 4, 5:
		// right now we use '$' to delim fields and put everything in L1
		// rather than a multi-dimentional L2, L3...
		sm.L1[key] = values
		return nil
	default:
		return fmt.Errorf("Invalid max_levels (%d)  Must be 0-5", max_levels)
	}
}
