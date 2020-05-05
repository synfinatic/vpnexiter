package vpnexiter

import (
	"fmt"
	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/spf13/viper"
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
	VpnServer string
}

func Update(vendor string, ipaddr string) error {
	v := viper.GetViper()
	mode := v.GetString("router.mode")
	if mode == "ssh" {
		log.Printf("Updating via ssh %s / %s", vendor, ipaddr)
		return update_ssh(vendor, ipaddr)
	} else if mode == "local" {
		log.Printf("Updating via local %s / %s", vendor, ipaddr)
		return update_local(vendor, ipaddr)
	} else {
		return fmt.Errorf("Unsupported mode: %s", mode)
	}
}

func create_config(vendor string, ipaddr string) (string, error) {
	v := viper.GetViper()
	tmpl := v.GetString(vendor + ".config_template")
	conf := Config{
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

func update_local(vendor string, ipaddr string) error {
	v := viper.GetViper()
	cfile, err := create_config(vendor, ipaddr)
	if err != nil {
		return err
	}

	config_file := v.GetString("router.config_file")
	err = os.Rename(cfile, config_file)
	if err != nil {
		return err
	}
	log.Printf("Success moving %s to %s", cfile, config_file)
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
		log.Printf("unable to create_config")
		return err
	}

	log.Printf("create_config: %s", cfile)
	clientConfig, _ := auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())

	client := scp.NewClient(router, &clientConfig)

	err = client.Connect()
	if err != nil {
		log.Printf("client.Connect() failed")
		return err
	}
	log.Printf("client.Connect()")

	defer client.Close()

	f, _ := os.Open(cfile)
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		log.Printf("Unable to Stat() %s", f.Name())
		return err
	}
	err = client.CopyFile(f, config_file, "0644")
	if err != nil {
		log.Printf("failed client.CopyFile() %s", err.Error())
		return err
	}
	log.Printf("Success copying %s to %s/%s", cfile, router, config_file)
	err = os.Remove(cfile)
	if err != nil {
		log.Printf("Error deleting %s", cfile)
	}

	return nil
}
