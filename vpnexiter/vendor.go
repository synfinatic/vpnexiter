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

/*
 * Loads the vendor: map configuration
 */
func LoadVendors() map[string]*VendorConfig {
	v := viper.GetViper()
	vcmap := map[string]*VendorConfig{}

	for _, vendor := range v.GetStringSlice("vendors") {
		log.Printf("Loading: %s", vendor)

		vcmap[vendor] = &VendorConfig{
			Name:    vendor,
			Levels:  []string{},
			Servers: *newServerMap(),
		}

		if v.IsSet(vendor + ".levels") {
			vcmap[vendor].Levels = append(vcmap[vendor].Levels, v.GetStringSlice(vendor+".levels")...)
		}

		start := []string{vendor, "servers"}
		if len(vcmap[vendor].Levels) == 0 {
			search := strings.Join(start, ".")
			vcmap[vendor].Servers.appendList(v.GetStringSlice(search))
		} else {
			build_server_map(&vcmap[vendor].Servers, start, vcmap[vendor].Levels)
		}

		b, err := vcmap[vendor].Servers.GenHTML()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s\n", b)
	}
	return vcmap
}

/*
 * Recursive function to populate the ServerMap with the config
 * data from Viper
 */
func build_server_map(sm *ServerMap, location []string, levels []string) {
	v := viper.GetViper()
	level_cnt := len(levels)
	search := strings.Join(location, ".")

	if level_cnt == 1 {
		// key := location[len(location)-1]
		for key, _ := range v.GetStringMap(search) {
			server_search := fmt.Sprintf("%s.%s", search, key)
			servers := v.GetStringSlice(server_search)
			err := sm.addList(key, servers)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		// pop off the next level
		levels := levels[1:len(levels)]
		// Iterate over our Config Viper map[string]inteface{}
		for key, _ := range v.GetStringMap(search) {
			loc := append(location, key)
			new_map := newServerMap()
			// attach our new_map to ourself
			sm.addMap(key, new_map)
			// recurse
			build_server_map(new_map, loc, levels)
		}
	}
}