package vpn

import (
	"bytes"
	"fmt"
	"gopkg.in/grignaak/tribool.v1"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

/*
 * Updates the IPSec config on a local system
 */
func (vs *VpnServer) update_config_local() error {
	cfile, err := vs.create_config()
	if err != nil {
		return err
	}

	config_file := vs.Konf.String("router.config_file")
	err = os.Rename(cfile, config_file)
	if err != nil {
		return err
	}
	log.Printf("Success moving %s to %s", cfile, config_file)
	return nil
}

func (vs *VpnServer) get_up_local() tribool.Tribool {

	tmpl := vs.Konf.String("router.check.command")
	cmd, err := vs.render_gs_template("local.check.command", tmpl)
	if err != nil {
		log.Printf("Unable to render template: %s", tmpl)
		return tribool.Maybe
	}
	log.Printf("running %s\n", cmd)
	out, err := exec_local_command(cmd)
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
 * returns the output of the `status_command`
 * on error, return stderr and the error
 * on success, return stdout and nil
 */
func (vs *VpnServer) status_local() (bytes.Buffer, error) {
	cmd, err := vs.render_gs_template("local_status", vs.Konf.String("router.status_command"))
	if err != nil {
		var buf bytes.Buffer
		return buf, err
	}
	out, err := exec_local_command(cmd)
	if err != nil {
		return out, err
	}
	return out, nil

}

/*
 * Restart IPSec on the local host
 */
func (vs *VpnServer) restart_vpn_local() (bool, error) {
	var vpn_up bool = false

	_, err := exec_local_command(vs.Konf.String("router.stop_command"))
	if err != nil {
		return vpn_up, err
	}
	_, err = exec_local_command(vs.Konf.String("router.start_command"))
	if err != nil {
		return vpn_up, err
	}
	// wait for VPN to come up
	for i := 0; i < vs.WaitSeconds; i++ {
		_, err = exec_local_command(vs.Konf.String("router.check_command"))
		if err != nil {
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
			"%s VPN to %s did not come up after %d seconds",
			vs.Exit, vs.Vendor, vs.WaitSeconds)
	}
	return vpn_up, nil
}
