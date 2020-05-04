package vpnexiter

import (
	"github.com/spf13/viper"
	"github.com/synfinatic/vpnexiter/scp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"os"
	"text/template"
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
			// failed to resolve, so store as-is
			log.Printf("Unable to resolve: %s", s)
			l := []string{s}
			slist.IPs[s] = l
			continue
		}
		slist.IPs[s] = ips
	}
	return &slist, nil
}

type Config struct {
	IPAddr string
}

func Update(vendor string, ipaddr string) error {
	mode := v.GetString("mode")
	if mode == "ssh" {
		return update_ssh(vendor, ipaddr)
	} else if mode == "local" {
		return update_local(vendor, ipaddr)
	} else {
		return fmt.Error("Unsupported mode: %s", mode)
	}
}

func create_config(vendor string, ipaddr string) (string, error) {
	v := viper.GetViper()
	tmpl := viper.GetString(vendor + ".config_template")
	output := viper.GetString("router.config_file")
	conf := Config{
		IPAddr: ipaddr,
	}
	tfile, err := template.ParseFiles(tmpl)
	if err != nil {
		return nil, err
	}
	out, err := ioutil.TempFile("", "vpnexiter")
	if err != nil {
		return nil, err
	}
	defer out.Close()

	err := tfile.Execute(out, conf)
	if err != nil {
		return nil, err
	}
	return out.Name(), nil
}

func update_local(vendor string, ipaddr string) error {
	v := viper.GetViper()
	cfile, err := create_config(vendor, ipaddr)
	if err != nil {
		return err
	}

	err := os.Rename(cfile.Name(), v.GetString(vendor+".config_template"))
	if err != nil {
		return err
	}
	return nil
}

func update_ssh(vendor string, ipaddr string) error {
	v := viper.GetViper()
	router := fmt.Sprintf("%s:%d",
		v.GetString("router.host"),
		v.GetInt("router.port"))

	username := v.GetString("router.user")
	password := v.GetString("router.password")
	config_file := v.GetString("router.config_file")

	cfile, err := create_config(vendor, ipaddr)
	if err != nil {
		return err
	}

	client, err := ssh.Dial("tcp", router, &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIngoreHostKey(), // FIXME
	})
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	err := scp.CopyPath(cfile, config_file, session)
	if _, err := os.Stat(config_file); os.IsNotExist(err) {
		return err
	} else {
		log.Printf("Succes copying to %s", config_file)
	}
	return nil
}
