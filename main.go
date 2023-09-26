// bat - Manage battery charge limit
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	version   = "0.11.0"
	years     = "2023"
	prefix    = "chargelimit-"
	pathglob  = "/sys/class/power_supply/BAT?"
  threshold = "charge_control_end_threshold"
  services  = "/etc/systemd/system/"
)

var events = [...]string{
	"hibernate",
	"hybrid-sleep",
	"multi-user",
	"suspend",
	"suspend-then-hibernate",}

var (
	//go:embed unit.tmpl
	unitfile   string
	//go:embed help.tmpl
	helpmsg    string
	//go:embed version.tmpl
	versionmsg string
	path       string
)

func usage() {
	fmt.Printf(helpmsg, version)
}

func errexit(msg string) {
	fmt.Fprintf(os.Stderr, "Fatal: %s\n", msg)
	os.Exit(1)
}

func mustRead(variable string) string {
	f, err := os.Open(filepath.Join(path, variable))
	if err != nil {
		return ""
	}
	defer f.Close()
	data := make([]byte, 32)
	n, err := f.Read(data)
	if err != nil && err != io.EOF {
		return ""
	}
	return string(data[:n-1])
}

func main() {
	command := "status"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "h", "help", "-h", "--help":
		usage()
		os.Exit(0)

	case "V", "v", "version", "-V", "-v", "--version":
		fmt.Printf(versionmsg, version, years)
		os.Exit(0)
	}
	var limit string
	if len(command) > 0 && command[0] >= '0' && command[0] <= '9' {
		limit = command
		command = "limit"
	}

	batteries, err := filepath.Glob(filepath.Join(pathglob, threshold))
	if err != nil || len(batteries) == 0 {
		errexit("No compatible battery devices found")
	}

	// Ignoring other compatible batteries!
	path, _ = filepath.Split(batteries[0])
	if len(batteries) > 1 {
		fmt.Println("More than 1 battery device found, using " + path)
	}
	switch command {
	case "s", "status", "-s", "--status":
		fmt.Printf("Level: %s%%\n", mustRead("capacity"))
		limit := mustRead(threshold)
		if limit != "" {
			fmt.Printf("Limit: %s%%\n", limit)
		}
		var health, full, design string
		var ifull, idesign int
		full = mustRead("charge_full")
		if full == "" { // Try energy_full
			full = mustRead("energy_full")
			if full != "" {
				design = mustRead("energy_full_design")
			}
		} else {
			design = mustRead("charge_full_design")
		}
		if full != "" && design != "" {
			ifull, err = strconv.Atoi(full)
			if err == nil && ifull > 0 {
				idesign, err = strconv.Atoi(design)
				if err == nil && idesign > 0 {
					health = fmt.Sprintf("%d", ifull*100/idesign)
				}
			}
		}
		if health != "" {
			fmt.Printf("Health: %s%%\n", health)
		} else {
			fmt.Println("Health cannot be determined")
		}
		fmt.Printf("Status: %s\n", mustRead("status"))
		if limit != "" {
			var disabled bool
			for _, event := range events {
				service := prefix + event + ".service"
				output, _ := exec.Command("systemctl", "is-active", service).Output()
				if string(output) != "active\n" {
					disabled = true
				}
			}
			enabled := "yes"
			if disabled {
				enabled = "no"
			}
			fmt.Printf("Persist: %s\n", enabled)
		} else {
			fmt.Println("Charge limit is not supported")
		}
	case "p", "persist", "-p", "--persist":
		output, err := exec.Command("systemctl", "--version").CombinedOutput()
		if err != nil {
			errexit("cannot run 'systemctl --version'")
		}

		var version int
		_, err = fmt.Sscanf(string(output), "systemd %d", &version)
		if err != nil {
			errexit("cannot read version from 'systemctl --version'")
		}

		if version < 244 { // oneshot not implemented yet
			errexit("systemd version 244-r1 or later required")
		}

		limit := mustRead(threshold)
		if limit == "" {
			errexit("cannot read current limit from '" + threshold + "'")
		}
		current, err := strconv.Atoi(limit)
		if err != nil || current == 0 {
			errexit("cannot convert '" + limit + "' to integer")
		}

		shell, err := exec.LookPath("sh")
		if err != nil && !errors.Is(err, exec.ErrNotFound) { // Just set /bin/sh as shell
			shell = "/bin/sh"
		}
		for _, event := range events {
			service := prefix + event + ".service"
			file := services + service
			f, err := os.Create(file)
			if err != nil {
				if errors.Is(err, syscall.EACCES) {
					errexit("insufficient permissions, run with root privileges")
				}

				errexit("could not create systemd unit file '" + file + "'")
			}

			defer f.Close()
			_, err = f.WriteString(fmt.Sprintf(unitfile, event, event, shell, current, batteries[0], event))
			if err != nil {
				errexit("could not instantiate systemd unit file '" + service + "'")
			}

			exec.Command("systemctl", "stop", service).Run()
			err = exec.Command("systemctl", "start", service).Run()
			if err != nil {
				errexit("could not start systemd unit file '" + service + "'")
			}
			err = exec.Command("systemctl", "enable", service).Run()
			if err != nil {
				errexit("could not enable systemd unit file '" + service + "'")
			}
		}

		fmt.Printf("Persistence enabled for charge limit: %d\n", current)
	case "r", "remove", "-r", "--remove":
		for _, event := range events {
			service := prefix + event + ".service"
			file := services + service
			exec.Command("systemctl", "stop", service).Run()
			output, err := exec.Command("systemctl", "disable", service).CombinedOutput()
			if err != nil {
				message := string(output)
				switch true {
				case strings.Contains(message, "does not exist"):
					continue
				case strings.Contains(message, "Access denied"):
					errexit("insufficient permissions, run with root privileges")
        default:
					errexit("failure to disable unit file '" + service + "'")
				}
			}
			err = os.Remove(file)
			if err != nil && !errors.Is(err, syscall.ENOENT) {
				errexit("failure to remove unit file '" + file + "'")
			}
		}
		fmt.Println("Persistence of charge limit removed")
	case "l", "limit", "-l", "--limit":
		if limit == "" {
			limit = os.Args[2]
			if limit == "" {
				errexit("Argument to 'limit' missing")
			}
		}

		ilimit, err := strconv.Atoi(limit)
		if err != nil || ilimit < 0 || ilimit > 100 {
			errexit("argument to limit must be an integer between 0 and 100")
		}

		if ilimit == 0 {
			ilimit = 100
		}
		l := []byte(fmt.Sprintf("%d", ilimit))
		err = os.WriteFile(batteries[0], l, 0o644)
		if err != nil {
			if errors.Is(err, syscall.EACCES) {
				errexit("insufficient permissions, run with root privileges")
			}

			errexit("could not set battery charge limit")
		}

		if ilimit == 100 {
			fmt.Println("Charge limit unset")
		} else {
			fmt.Println("Charge limit set, to make it persist, run:\nbat persist")
		}
	default:
		usage()
		errexit("argument '" + command + "' invalid")
	}
}
