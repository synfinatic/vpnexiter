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
func (vs *VpnServer) updateConfigSsh() error {

	configFile := vs.Konf.String("router.config_file")

	cfile, err := vs.createConfig()
	if err != nil {
		log.Printf("unable to createConfig")
		return err
	}
	defer os.Remove(cfile)

	log.Printf("createConfig: %s", cfile)

	router, clientConfig := vs.sshConfig()
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

	err = client.CopyFile(f, configFile, "0644")
	if err != nil {
		log.Printf("failed client.CopyFile() %s", err.Error())
		return err
	}
	log.Printf("Success copying %s to %s/%s", cfile, router, configFile)

	return nil
}

/*
 * Builds a ssh.ClientConfig for a ssh connection
 */
func (vs *VpnServer) sshConfig() (string, ssh.ClientConfig) {
	router := fmt.Sprintf("%s:%d",
		vs.Konf.String("router.host"),
		vs.Konf.Int("router.port"))

	username := vs.Konf.String("router.user")
	password := vs.Konf.String("router.password")
	clientConfig, _ := auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())
	return router, clientConfig
}

func (vs *VpnServer) getUpSsh() tribool.Tribool {
	router, config := vs.sshConfig()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()
	return vs.checkSsh(conn)
}

func (vs *VpnServer) checkSsh(conn *ssh.Client) tribool.Tribool {
	tmpl := vs.Konf.String("router.check.command")
	cmd, err := vs.renderGsTemplate("ssh.check.command", tmpl)
	if err != nil {
		log.Printf("Unable to render template: %s", tmpl)
		return tribool.Maybe
	}

	log.Printf("running %s\n", cmd)
	out, err := execSshCommand(conn, cmd)
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

func (vs *VpnServer) statusSsh() (bytes.Buffer, error) {
	var buf bytes.Buffer
	router, config := vs.sshConfig()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()

	cmd, err := vs.renderGsTemplate("ssh_status", vs.Konf.String("router.status_command"))
	if err != nil {
		return buf, err
	}
	buf, err = execSshCommand(conn, cmd)
	if err != nil {
		return buf, err
	}
	return buf, nil
}

/*
 * Run a command on a ssh connection
 * Returns: stdout on success or stderr on error
 */
func execSshCommand(conn *ssh.Client, command string) (bytes.Buffer, error) {
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
func (vs *VpnServer) restartVpnSsh() (bool, error) {
	var vpnUp bool = false

	router, config := vs.sshConfig()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()

	_, err := execSshCommand(conn, vs.Konf.String("router.stop_command"))
	if err != nil {
		return vpnUp, err
	}
	_, err = execSshCommand(conn, vs.Konf.String("router.start_command"))
	if err != nil {
		return vpnUp, err
	}

	var buf bytes.Buffer
	for i := 0; i < vs.WaitSeconds; i++ {
		ret := vs.checkSsh(conn)
		if ret != tribool.True {
			duration, _ := time.ParseDuration("1s")
			time.Sleep(duration)
			continue
		} else {
			vpnUp = true
			break
		}
	}
	if !vpnUp {
		return vpnUp, fmt.Errorf(
			"%s VPN to %s did not come up after %d seconds: %s",
			vs.Exit, vs.Vendor, vs.WaitSeconds, buf.String())
	}
	return vpnUp, nil
}
