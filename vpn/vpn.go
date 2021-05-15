package vpn

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"text/template"

	"github.com/knadh/koanf"
	"gopkg.in/grignaak/tribool.v1"
)

type VpnServer struct {
	Type        string
	ConfigFile  string
	Konf        *koanf.Koanf
	WaitSeconds int
	// These values are only for Type == ssh
	Host     string
	Port     int
	Username string
	Password string
	// These values are modified at runtime
	Vendor string
	Exit   string
}

func NewVpn(konf *koanf.Koanf) *VpnServer {
	var vs *VpnServer
	if konf.String("router.mode") == "local" {
		vs = &VpnServer{
			Type:        "local",
			Konf:        konf,
			WaitSeconds: 5,
			ConfigFile:  konf.String("router.konfig_file"),
		}
	} else if konf.String("router.mode") == "ssh" {
		vs = &VpnServer{
			Type:        "ssh",
			Konf:        konf,
			ConfigFile:  konf.String("router.konfig_file"),
			WaitSeconds: 5,
			Host:        konf.String("router.host"),
			Port:        konf.Int("router.port"),
			Username:    konf.String("router.username"),
			Password:    konf.String("router.password"),
		}
	}
	return vs
}

func (vs *VpnServer) UpdateConfig(vendor string, exit string) error {
	if vs.Type == "ssh" {
		vs.Vendor = vendor
		vs.Exit = exit
		return vs.updateConfigSsh()
	} else if vs.Type == "local" {
		vs.Vendor = vendor
		vs.Exit = exit
		return vs.updateConfigLocal()
	}
	return fmt.Errorf("Unsupported VpnServer.Type: %s", vs.Type)
}

func (vs *VpnServer) IsUp() (tribool.Tribool, error) {
	if vs.Type == "ssh" {
		return vs.getUpSsh(), nil
	} else if vs.Type == "local" {
		return vs.getUpLocal(), nil
	}
	return tribool.Maybe, fmt.Errorf("Unsupported VpnServer.Type: %s", vs.Type)
}

func (vs *VpnServer) Restart() (bool, error) {
	if vs.Type == "ssh" {
		return vs.restartVpnSsh()
	} else if vs.Type == "local" {
		return vs.restartVpnLocal()
	}
	return false, fmt.Errorf("Unsupported VpnServer.Type: %s", vs.Type)
}

func (vs *VpnServer) Status() (bytes.Buffer, error) {
	if vs.Type == "ssh" {
		return vs.statusSsh()
	} else if vs.Type == "local" {
		return vs.statusLocal()
	}
	var buf bytes.Buffer // can't use nil in return
	return buf, fmt.Errorf("Unsupported VpnServer.Type: %s", vs.Type)
}

// Everything that belongs in the config template needs to be here
type ConfigTemplate struct {
	VpnServer string
	Vendor    string
}

/*
 * Helper function to create the IPSec config for a given vendor
 * Returns the name of a tempfile containing the contents of the config file
 */
func (vs *VpnServer) createConfig() (string, error) {
	tmpl := vs.Konf.String(vs.Vendor + ".config_template")
	conf := ConfigTemplate{
		VpnServer: vs.Exit,
		Vendor:    vs.Vendor,
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

/*
 * Shared function for ssh to do variable interpolation for
 * commands.  Users can use `Vendor`, `Exit` or any
 * exported value in GlobalState
 */
func (vs *VpnServer) renderGsTemplate(name string, templ string) (string, error) {
	t, err := template.New(name).Parse(templ)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, *vs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
