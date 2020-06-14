package vpnexiter

import (
	"bytes"
	"fmt"
	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"strings"
	"time"
)

/*
 * Updates the config on a remote system via SSH
 */
func update_ssh(vendor string, ipaddr string) error {
	v := viper.GetViper()

	config_file := v.GetString("router.config_file")

	cfile, err := create_config(vendor, ipaddr)
	if err != nil {
		log.Printf("unable to create_config")
		return err
	}
	defer os.Remove(cfile)

	log.Printf("create_config: %s", cfile)

	router, clientConfig := ssh_config()
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
func ssh_config() (string, ssh.ClientConfig) {
	v := viper.GetViper()
	router := fmt.Sprintf("%s:%d",
		v.GetString("router.host"),
		v.GetInt("router.port"))

	username := v.GetString("router.user")
	password := v.GetString("router.password")
	clientConfig, _ := auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())
	return router, clientConfig
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
func restart_ipsec_ssh(vendor string) error {
	seconds := 5
	v := viper.GetViper()
	router, config := ssh_config()
	conn, _ := ssh.Dial("tcp", router, &config)
	defer conn.Close()
	_, err := exec_ssh_command(*conn, v.GetString("router.stop_command"))
	if err != nil {
		return err
	}
	_, err = exec_ssh_command(*conn, v.GetString("router.start_command"))
	if err != nil {
		return err
	}
	var vpn_up bool = false
	for i := 0; i < seconds; i++ {
		out, err := exec_ssh_command(*conn, v.GetString("router.status_command"))
		if err != nil {
			return err
		}
		if !strings.Contains(out.String(), "1 up") {
			vpn_up = true
			break
		}
		duration, _ := time.ParseDuration("1s")
		time.Sleep(duration)
	}
	if !vpn_up {
		return fmt.Errorf("VPN to %s did not come up after %d seconds", vendor, seconds)
	}
	return nil
}
