package vpn

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/grignaak/tribool.v1"
)

/*
 * Updates the IPSec config on a local system
 */
func (vs *VpnServer) updateConfigLocal() error {
	cfile, err := vs.createConfig()
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

func (vs *VpnServer) getUpLocal() tribool.Tribool {

	tmpl := vs.Konf.String("router.check.command")
	cmd, err := vs.renderGsTemplate("local.check.command", tmpl)
	if err != nil {
		log.Printf("Unable to render template: %s", tmpl)
		return tribool.Maybe
	}
	log.Printf("running %s\n", cmd)
	out, err := execLocalCommand(cmd)
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
func execLocalCommand(command string) (bytes.Buffer, error) {
	execCmd := strings.Split(command, " ")
	name := execCmd[0]
	execCmd = execCmd[1:]
	cmd := exec.Command(name, execCmd...)
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
func (vs *VpnServer) statusLocal() (bytes.Buffer, error) {
	cmd, err := vs.renderGsTemplate("local_status", vs.Konf.String("router.status_command"))
	if err != nil {
		var buf bytes.Buffer
		return buf, err
	}
	out, err := execLocalCommand(cmd)
	if err != nil {
		return out, err
	}
	return out, nil

}

/*
 * Restart IPSec on the local host
 */
func (vs *VpnServer) restartVpnLocal() (bool, error) {
	var vpnUp bool = false

	_, err := execLocalCommand(vs.Konf.String("router.stop_command"))
	if err != nil {
		return vpnUp, err
	}
	_, err = execLocalCommand(vs.Konf.String("router.start_command"))
	if err != nil {
		return vpnUp, err
	}
	// wait for VPN to come up
	for i := 0; i < vs.WaitSeconds; i++ {
		_, err = execLocalCommand(vs.Konf.String("router.check_command"))
		if err != nil {
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
			"%s VPN to %s did not come up after %d seconds",
			vs.Exit, vs.Vendor, vs.WaitSeconds)
	}
	return vpnUp, nil
}
