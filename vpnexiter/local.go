package vpnexiter

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

/*
 * Updates the IPSec config on a local system
 */
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

/*
 * Exec a command locally
 * Returns stdout on success or stderr on error
 */
func exec_local_command(command string) (bytes.Buffer, error) {
	exec_cmd := strings.Split(command, " ")
	name := exec_cmd[0]
	exec_cmd = exec_cmd[1:]
	cmd := exec.Command(name, exec_cmd...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("error running %s: %s", command, err)
		log.Printf("-- stderr:\n%s", stderr.String())
		return stderr, err
	}
	log.Printf("success running %s:", command)
	log.Printf("-- stdout:\n%s", stdout.String())
	return stdout, nil
}

/*
 * Restart IPSec on the local host
 */
func RestartIpSecLocal(vendor string) error {
	seconds := 5
	v := viper.GetViper()
	out, err := exec_local_command(v.GetString("router.stop_command"))
	if err != nil {
		return err
	}
	out, err = exec_local_command(v.GetString("router.start_command"))
	if err != nil {
		return err
	}
	var vpn_up bool = false
	for i := 0; i < seconds; i++ {
		out, err = exec_local_command(v.GetString("router.status_command"))
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
		return fmt.Errorf("VPN to %s did not come up after %d seconds", seconds)
	}
	return nil
}
