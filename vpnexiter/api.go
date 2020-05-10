package vpnexiter

import (
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"text/template"
)

// Everything that belongs in the config template needs to be here
type ConfigTemplate struct {
	VpnServer string
}

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
			// failed to resolve
			is_ip, _ := regexp.Match(`\d+\.\d+\.\d+\.\d+`, []byte(s))
			if !is_ip {
				// weren't able to resolve a FQDN
				log.Printf("Unable to resolve: %s", s)
				continue
			}
			// IP addresses get added as-is
			l := []string{s}
			slist.IPs[s] = l
			continue
		}
		slist.IPs[s] = ips
	}
	return &slist, nil
}

func Update(vendor string, ipaddr string) error {
	v := viper.GetViper()
	switch mode := v.GetString("router.mode"); mode {
	case "ssh":
		log.Printf("Updating via ssh %s / %s", vendor, ipaddr)
		return update_ssh(vendor, ipaddr)
	case "local":
		log.Printf("Updating via local %s / %s", vendor, ipaddr)
		return update_local(vendor, ipaddr)
	default:
		return fmt.Errorf("Unsupported mode: %s", mode)
	}
}

/*
 * Helper function to create the IPSec config for a given vendor
 * Returns the name of a tempfile containing the contents of the config file
 */
func create_config(vendor string, ipaddr string) (string, error) {
	v := viper.GetViper()
	tmpl := v.GetString(vendor + ".config_template")
	conf := ConfigTemplate{
		VpnServer: ipaddr,
	}
	tfile, err := template.ParseFiles(tmpl)
	if err != nil {
		return "", err
	}
	out, err := ioutil.TempFile("", "vpnexiter")
	if err != nil {
		return "", err
	}
	defer out.Close()

	err = tfile.Execute(out, conf)
	if err != nil {
		return "", err
	}
	log.Printf("Success generating temporary config file: %s", out.Name())
	return out.Name(), nil
}
