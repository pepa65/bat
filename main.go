// bat - Manage battery charge limit
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	version                   = "0.9.0"
	years                     = "2023"
	msgTrue                   = "yes"
	msgFalse                  = "no"
	msgArgNotInt              = "Argument must be an integer"
	msgExpectedSingleArg      = "Single argument needed"
	msgIncompatibleKernel     = "Linux kernel version 5.4 or later required"
	msgIncompatibleSystemd    = "Package systemd version 243-rc1 or later required"
	msgNoOption               = "Option %s not implemented. Run `bat help` to see the available options.\n"
	msgOutOfRangeThresholdVal = "Percentage must be between 1 and 100"
	msgPermissionDenied       = "Permission denied. Try running this command using 'sudo'"
	msgPersistenceEnabled     = "Persist systemd units present and enabled"
	msgPersistenceRemoved     = "Persist systemd units not present"
	msgLimitSet               = "Charge limit set.\nRun 'sudo bat persist' to keep it after restart/hibernation/sleep"
	msgIncompatible           = `This program is most likely not compatible with your system. See
https://github.com/pepa65/bat#disclaimer for details`
)

const (
	success = iota
	failure
)

var (
	versionmsg = "bat v"+version+`
Copyright `+years+" Tshaka Eric Lekholoane, github.com/pepa65 (MIT License)"
	helpmsg = "bat v"+version+` - Manage battery charge limit
Repo:  github.com/pepa65/bat
Usage: bat <option>
  Options (every option except 's[tatus]' needs root privileges):
    [s[tatus]]       Display charge level, limit, health & persist status.
    l[imit] <int>    Set the charge limit to <int> percent.
    p[ersist]        Install and enable the persist systemd unit files.
    r[emove]         Remove the persist systemd unit files.
    h[elp]           Just display this help text.
    v[ersion]        Just display version information.`
)

// resetwriter is the interface that groups the methods to access systemd services.
type resetwriter interface {
	Remove() error
	Write() error
	Enabled() error
}

// console represents a text terminal user interface.
type console struct {
	// err represents standard error.
	err io.Writer
	// out represents standard output.
	out io.Writer
	// quit is the function that sets the exit code.
	quit func(code int)
}

// app represents this application and its dependencies.
type app struct {
	console *console
	// pager is the path of the pager.
	pager string
	// cat is the path of cat for fallback.
	cat string
	// get is the function used to read the value of the battery variable.
	get func(Variable) (string, error)
	// set is the function used to write the battery charge limit value.
	set func(Variable, string) error
	// systemder is used to write and delete systemd services that persist
	// the charge limit after restart/hibernate/sleep.
	systemder resetwriter
}

// Print to stderr using format, exit with error code 1
func (a *app) errorf(format string, v ...any) {
	fmt.Fprintf(a.console.err, format, v...)
	a.console.quit(failure)
}

// Print to stderr with neline using format, exit with error code 1
func (a *app) errorln(v ...any) {
	a.errorf("%v\n", v...)
}

// Print to stdout according to format specifier
func (a *app) writef(format string, v ...any) {
	fmt.Fprintf(a.console.out, format, v...)
}

// Print to stdout with newline using format for its operands
func (a *app) writeln(v ...any) {
	a.writef("%v\n", v...)
}

// Filter the string doc through the less pager
func (a *app) page(doc string) {
	cmd := exec.Command(
		a.pager,
		"--no-init",
		"--chop-long-lines",
		"--quit-if-one-screen",
		"--IGNORE-CASE",
		"--RAW-CONTROL-CHARS",
	)
	cmd.Stdin = strings.NewReader(doc)
	cmd.Stdout = a.console.out
	if err := cmd.Run(); err != nil {
		cmd := exec.Command(a.cat)
		cmd.Stdin = strings.NewReader(doc)
		cmd.Stdout = a.console.out
		if err := cmd.Run(); err != nil {
			log.Fatalln(err)
		}
	}
	a.console.quit(success)
}

// Return the value of the given /sys/class/power_supply/BAT?/ variable
func (a *app) show(v Variable) string {
	val, err := a.get(v)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			a.errorln(msgIncompatible)
		}
		log.Fatalln(err)
	}
	return val
}

// Print the battery health
func (a *app) health() string {
	energy := false
	var chargedesign string
	var icharge, ichargedesign int
	charge, err := a.get(ChargeFull)
	if err != nil {
		if errors.Is(err, ErrNotFound) { // Try EnergyFull
			charge, err = a.get(EnergyFull)
			if errors.Is(err, ErrNotFound) {
				a.errorln(msgIncompatible)
			} else {
				energy = true
			}
		} else {
			log.Fatalln(err)
		}
	}
	if energy {
		chargedesign, err = a.get(EnergyFullDesign)
	} else {
		chargedesign, err = a.get(ChargeFullDesign)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			a.errorln(msgIncompatible)
		}
		log.Fatalln(err)
	}
	icharge, err = strconv.Atoi(charge)
	if err != nil {
		log.Fatalln(err)
	}
	ichargedesign, err = strconv.Atoi(chargedesign)
	if err != nil {
		log.Fatalln(err)
	}
	return (fmt.Sprintf("%d", icharge*100/ichargedesign))
}

func (a *app) help() {
	buf := new(bytes.Buffer)
	buf.Grow(1024)
	fmt.Fprint(buf, helpmsg)
	a.page(buf.String())
}

func (a *app) version() {
	buf := new(bytes.Buffer)
	buf.Grow(128)
	fmt.Fprint(buf, versionmsg)
	a.page(buf.String())
}

func (a *app) persist() {
	if err := a.systemder.Write(); err != nil {
		switch {
		case errors.Is(err, ErrIncompatSystemd):
			a.errorln(msgIncompatibleSystemd)
			return
		case errors.Is(err, ErrNotFound):
			a.errorln(msgIncompatible)
			return
		case errors.Is(err, syscall.EACCES):
			a.errorln(msgPermissionDenied)
			return
		default:
			log.Fatalln(err)
		}
	}
	a.writef("%s: %s%%\n", msgPersistenceEnabled, a.show(Threshold))
}

func (a *app) remove() {
	if err := a.systemder.Remove(); err != nil {
		if errors.Is(err, syscall.EACCES) {
			a.errorln(msgPermissionDenied)
			return
		}
		log.Fatal(err)
	}
	a.writeln(msgPersistenceRemoved)
}

func (a *app) enabled() string {
	if err := a.systemder.Enabled(); err != nil {
		return msgFalse
	} else {
		return msgTrue
	}
}

func (a *app) status() {
	a.writef("Level: %s%%\n", a.show(Capacity))
	limit, _ := a.get(Threshold)
	if limit != "" {
		a.writef("Limit: %s%%\n", a.show(Threshold))
	}
	a.writef("Health: %s%%\n", a.health())
	a.writef("Status: %s\n", a.show(Status))
	if limit != "" {
		a.writef("Persist: %s\n", a.enabled())
	} else {
		a.writeln("Charge limit is not supported")
	}
}

// Return true if limit is in the range 1-100
func valid(limit int) bool {
	return limit >= 1 && limit <= 100
}

// Return the Linux kernel version as a string and an error otherwise
func kernel() (string, error) {
	var name unix.Utsname
	if err := unix.Uname(&name); err != nil {
		return "", err
	}
	return string(name.Release[:]), nil
}

// Return true if ver represents a semantic version later than 5.4
// and false otherwise (5.4 is the earliest version of the Linux kernel
// to expose the battery charge limit variable)
// Returns error if parsing the string fails
func requiredKernel(ver string) (bool, error) {
	var maj, min int
	_, err := fmt.Sscanf(ver, "%d.%d", &maj, &min)
	if err != nil {
		return false, err
	}
	if maj > 5 || (maj == 5 && min >= 4) {
		return true, nil
	}
	return false, nil
}

func (a *app) limit(args []string) {
	if len(args) != 3 {
		a.errorln(msgExpectedSingleArg)
		return
	}
	val := args[2]
	t, err := strconv.Atoi(val)
	if err != nil {
		if errors.Is(err, strconv.ErrSyntax) {
			a.errorln(msgArgNotInt)
			return
		}
		log.Fatal(err)
	}
	if !valid(t) {
		a.errorln(msgOutOfRangeThresholdVal)
		return
	}
	ver, err := kernel()
	if err != nil {
		log.Fatal(err)
		return
	}
	ok, err := requiredKernel(ver)
	if err != nil {
		log.Fatal(err)
		return
	}
	if !ok {
		a.errorln(msgIncompatibleKernel)
		return
	}
	if err := a.set(Threshold, strings.TrimSpace(val)); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			a.errorln(msgIncompatible)
			return
		case errors.Is(err, syscall.EACCES):
			a.errorln(msgPermissionDenied)
			return
		default:
			log.Fatal(err)
		}
	}
	a.writeln(msgLimitSet)
}

// Execute the application
func main() {
	app := &app{
		console: &console{
			err:  os.Stderr,
			out:  os.Stdout,
			quit: os.Exit,
		},
		pager:     "less",
		cat:       "cat",
		get:       Get,
		set:       Set,
		systemder: New(),
	}
	if len(os.Args) == 1 {
		app.status()
		return
	}
	switch command := os.Args[1]; command {
	case "h", "help", "-h", "--help":
		app.help()
	case "V", "v", "version", "-V", "-v", "--version":
		app.version()
	case "s", "status", "-s", "--status":
		app.status()
	case "p", "persist", "-p", "--persist":
		app.persist()
	case "r", "remove", "-r", "--remove":
		app.remove()
	case "l", "limit", "-l", "--limit":
		app.limit(os.Args)
	default:
		app.errorf(msgNoOption, command)
	}
}
