package vpn

import (
	"bytes"
	"fmt"
	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
	"gopkg.in/grignaak/tribool.v1"
	"log"
	"os"
	"strings"
	"time"
)

/*
 * Updates the config on a remote system via SSH
 */
func (vs *VpnServer) update_config_ssh() error {

	config_file := vs.Konf.String("router.config_file")

	cfile, err := vs.create_config()
	if err != nil {
		log.Printf("unable to create_config")
		return err
	}
	defer os.Remove(cfile)

	log.Printf("create_config: %s", cfile)

	router, clientConfig := vs.ssh_config()
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

	err = client.CopyFile(f, config_file, "0644")
	if err != nil {
		log.Printf("failed client.CopyFile() %s", err.Error())
		return err
	}
	log.Printf("Success copying %s to %s/%s", cfile, router, config_file)

	return nil
}

/*
 * Builds a ssh.ClientConfig for a ssh connection
 */
func (vs *VpnServer) ssh_config() (string, ssh.ClientConfig) {
	router := fmt.Sprintf("%s:%d",
		vs.Konf.String("router.host"),
		vs.Konf.Int("router.port"))

	username := vs.Konf.String("router.user")
	password := vs.Konf.String("router.password")
	clientConfig, _ := auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())
	return router, clientConfig
}

func (vs *VpnServer) get_up_ssh() tribool.Tribool {
	router, config := vs.ssh_config()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()
	return vs.check_ssh(conn)
}

func (vs *VpnServer) check_ssh(conn *ssh.Client) tribool.Tribool {
	tmpl := vs.Konf.String("router.check.command")
	cmd, err := vs.render_gs_template("ssh.check.command", tmpl)
	if err != nil {
		log.Printf("Unable to render template: %s", tmpl)
		return tribool.Maybe
	}

	log.Printf("running %s\n", cmd)
	out, err := exec_ssh_command(*conn, cmd)
	if err != nil {
		log.Printf("error running: %s\n", cmd)
		return tribool.False
	}
	if strings.Contains(out.String(), vs.Konf.String("router.check.match")) {
		log.Printf("Matched!\n")
		return tribool.True
	}
	log.Printf("No match :(\n")
	return tribool.False
}

func (vs *VpnServer) status_ssh() (bytes.Buffer, error) {
	var buf bytes.Buffer
	router, config := vs.ssh_config()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()

	cmd, err := vs.render_gs_template("ssh_status", vs.Konf.String("router.status_command"))
	if err != nil {
		return buf, err
	}
	buf, err = exec_ssh_command(*conn, cmd)
	if err != nil {
		return buf, err
	}
	return buf, nil
}

/*
 * Run a command on a ssh connection
 * Returns: stdout on success or stderr on error
 */
func exec_ssh_command(conn ssh.Client, command string) (bytes.Buffer, error) {
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	err := session.Run(command)
	if err != nil {
		log.Printf("Error running `%s`: %s\n%s", command, err.Error(), stderrBuf.String())
		return stderrBuf, err
	}
	return stdoutBuf, nil
}

/*
 * SSH to the router and restart IPSec
 */
func (vs *VpnServer) restart_vpn_ssh() (bool, error) {
	var vpn_up bool = false

	router, config := vs.ssh_config()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()

	_, err := exec_ssh_command(*conn, vs.Konf.String("router.stop_command"))
	if err != nil {
		return vpn_up, err
	}
	_, err = exec_ssh_command(*conn, vs.Konf.String("router.start_command"))
	if err != nil {
		return vpn_up, err
	}

	var buf bytes.Buffer
	for i := 0; i < vs.WaitSeconds; i++ {
		ret := vs.check_ssh(conn)
		if ret != tribool.True {
			duration, _ := time.ParseDuration("1s")
			time.Sleep(duration)
			continue
		} else {
			vpn_up = true
			break
		}
	}
	if !vpn_up {
		return vpn_up, fmt.Errorf(
			"%s VPN to %s did not come up after %d seconds: %s",
			vs.Exit, vs.Vendor, vs.WaitSeconds, buf.String())
	}
	return vpn_up, nil
}
