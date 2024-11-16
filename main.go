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
	version       = "0.16.1"
	years         = "2023-2024"
	prefix        = "chargelimit-"
	services      = "/etc/systemd/system/"
	sleepfilename = "/usr/lib/systemd/system-sleep/chargelimit"
	syspath       = "/sys/class/power_supply/"
	threshold     = "charge_control_end_threshold"
)

var (
	events = [...]string{
		"hibernate",
		"hybrid-sleep",
		"multi-user",
		"suspend",
		"suspend-then-hibernate",
	}
	//go:embed unit.tmpl
	unitfile string
	//go:embed system-sleep.tmpl
	sleepfile string
	//go:embed help.tmpl
	helpmsg string
	//go:embed version.tmpl
	versionmsg string
	batpath    string
	bat        string
)

func usage() {
	fmt.Printf(helpmsg, version)
}

func errexit(msg string) { // I:bat
	fmt.Fprintf(os.Stderr, "[%s] Fatal: %s\n", bat, msg)
	os.Exit(1)
}

func mustRead(variable string) string { // I:batpath
	f, err := os.Open(filepath.Join(batpath, variable))
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
	maxArgs := 1
	command := "status"
	if len(os.Args) > 1 {
		command = os.Args[1]
		maxArgs = 2
	}
	switch command {
	case "l", "limit", "-l", "--limit":
		maxArgs = 3
	}
	if len(os.Args) > maxArgs {
		errexit("too many arguments")
	}

	switch command {
	case "h", "help", "-h", "--help":
		usage()
		os.Exit(0)

	case "V", "v", "version", "-V", "-v", "--version":
		fmt.Printf(versionmsg, version, years)
		os.Exit(0)
	}
	limit := ""
	if len(command) > 0 && command[0] >= '0' && command[0] <= '9' {
		limit = command
		command = "limit"
	}

	batselect := os.Getenv("BAT_SELECT")
	batglob := batselect
	if len(batselect) != 4 || batselect[:3] != "BAT" {
		batglob = "BAT?"
		batselect = ""
	}
	batteries, err := filepath.Glob(syspath + batglob)
	if err != nil || len(batteries) == 0 {
		bat = batglob
		errexit("No battery device found")
	}

	// Ignoring any other batteries!
	batpath = batteries[0]
	bat = batpath[len(batpath)-4:]
	if len(batteries) > 1 {
		fmt.Printf("More than 1 battery device found:")
		for _, battery := range batteries {
			fmt.Printf(" %s", battery[len(battery)-4:])
		}
		fmt.Println("")
	}
	thresholdpath := filepath.Join(batpath, threshold)
	switch command {
	case "s", "status", "-s", "--status":
		fmt.Printf("[%s]\n", bat)
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
			disabled := false
			for _, event := range events {
				service := prefix + event + ".service"
				output, _ := exec.Command("systemctl", "is-enabled", service).Output()
				if string(output) != "enabled\n" {
					disabled = true
				}
			}
			_, err = os.Stat(sleepfilename)
			if errors.Is(err, os.ErrNotExist) {
fmt.Println("No sleepfile")
				disabled = true
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
			_, err = f.WriteString(fmt.Sprintf(unitfile, bat, current, event, event, shell, current, thresholdpath, event))
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
		f, err := os.Create(sleepfilename)
		if err != nil {
			errexit("could not create system-sleep file '" + sleepfilename + "'")
		}
		defer f.Close()
		_, err = f.WriteString(fmt.Sprintf(sleepfile, bat, current, current, bat))
		if err != nil {
			errexit("could not instantiate system-sleep file '" + sleepfilename + "'")
		}

		fmt.Printf("[%s] Persistence enabled for charge limit: %d\n", bat, current)
	case "r", "remove", "-r", "--remove":
		os.Remove(sleepfilename)
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
		fmt.Printf("[%s] Persistence of charge limit removed\n", bat)
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
		err = os.WriteFile(thresholdpath, l, 0o644)
		if err != nil {
			if errors.Is(err, syscall.EACCES) {
				errexit("insufficient permissions, run with root privileges")
			}

			errexit("could not set battery charge limit")
		}

		if ilimit == 100 {
			fmt.Printf("[%s] Charge limit unset\n", bat)
		} else {
			bselect := ""
			if batselect != "" {
				bselect = fmt.Sprintf("BAT_SELECT=%s ", batselect)
			}
			fmt.Printf("[%s] Charge limit set, to make it persist, run:\n%sbat persist\n", bat, bselect)
		}
	default:
		usage()
		errexit("argument '" + command + "' invalid")
	}
}
