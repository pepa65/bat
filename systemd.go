// Package systemd - Managing systemd services to persist charge limit after restart/hibernation/sleep
package main

import (
	"bytes"
	_ "embed" // Allow embedding systemd unit template
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
	"text/template"

	//"github.com/pepa65/bat/pkg/power"
)

// ErrIncompatSystemd indicates an incompatible version of systemd.
var ErrIncompatSystemd = errors.New("systemd: incompatible systemd version")

// unit is a template of a systemd unit file that encodes information
// about the services used to persist the charge limit after restart/hibernation/sleep,
//
//go:embed unit.tmpl
var unit string

// compatSystemd returns nil if the systemd version of the system in
// question is later than 244 and returns false otherwise.
// (systemd v244-rc1 is the earliest version to allow restarts for
// oneshot services).
func compatSystemd() error {
	out, err := exec.Command("systemctl", "--version").Output()
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`\d+`)
	ver, err := strconv.Atoi(string(re.Find(out)))
	if err != nil {
		return err
	}
	if ver < 244 {
		return ErrIncompatSystemd
	}
	return nil
}

// config represents a systemd unit file's configuration for a service.
type config struct {
	Event, Target string
	Threshold     int
}

func configs() ([]config, error) {
	val, err := Get(Threshold)
	if err != nil {
		return nil, err
	}
	threshold, err := strconv.Atoi(val)
	if err != nil {
		return nil, err
	}
	return []config{
		{"boot", "multi-user", threshold},
		{"hibernation", "hibernate", threshold},
		{"hybridsleep", "hybrid-sleep", threshold},
		{"sleep", "suspend", threshold},
		{"suspendthenhibernate", "suspend-then-hibernate", threshold},
	}, nil
}

// Systemd directory
type Systemd struct{ dir string }

// New creates a new Systemd with the directory set to
// /etc/systemd/system/.
func New() *Systemd {
	return &Systemd{dir: "/etc/systemd/system/"}
}

// process runs the given function on the configurations in parallel and
// returns an error if any one call resulted in a error.
func process(cfgs []config, fn func(cfg config, in chan<- error)) error {
	errs := make(chan error, len(cfgs))
	for _, cfg := range cfgs {
		go fn(cfg, errs)
	}
	for range cfgs {
		if err := <-errs; err != nil {
			return err
		}
	}
	return nil
}

func (s *Systemd) remove(cfgs []config) error {
	return process(cfgs, func(cfg config, in chan<- error) {
		name := s.dir + "bat-" + cfg.Event + ".service"
		if err := os.Remove(name); err != nil && !errors.Is(err, syscall.ENOENT) {
			in <- err
			return
		}
		in <- nil
	})
}

func (s *Systemd) write(cfgs []config) error {
	if err := compatSystemd(); err != nil {
		return err
	}
	tmpl, err := template.New("unit").Parse(unit)
	if err != nil {
		return err
	}
	return process(cfgs, func(cfg config, in chan<- error) {
		name := s.dir + "bat-" + cfg.Event + ".service"
		sf, err := os.Create(name)
		if err != nil && !errors.Is(err, syscall.ENOENT) {
			in <- err
			return
		}
		defer sf.Close()
		if err := tmpl.Execute(sf, cfg); err != nil {
			in <- err
			return
		}
		in <- nil
	})
}

func (s *Systemd) disable(cfgs []config) error {
	return process(cfgs, func(cfg config, in chan<- error) {
		name := "bat-" + cfg.Event + ".service"
		buf := new(bytes.Buffer)
		cmd := exec.Command("systemctl", "disable", name)
		cmd.Stderr = buf
		if err := cmd.Run(); err != nil &&
			!bytes.Contains(buf.Bytes(), []byte(name+" does not exist.")) {
			in <- err
			return
		}
		in <- nil
	})
}

func (s *Systemd) enable(cfgs []config) error {
	return process(cfgs, func(cfg config, in chan<- error) {
		name := "bat-" + cfg.Event + ".service"
		cmd := exec.Command("systemctl", "enable", name)
		if err := cmd.Run(); err != nil {
			in <- err
			return
		}
		in <- nil
	})
}

func (s *Systemd) present(cfgs []config) error {
	return process(cfgs, func(cfg config, in chan<- error) {
		name := "bat-" + cfg.Event + ".service"
		output, err := exec.Command("systemctl", "list-unit-files", "-q", name).Output()
		if err != nil || string(output) == "" {
			in <- err
			return
		}
		in <- nil
	})
}

func (s *Systemd) enabled(cfgs []config) error {
	return process(cfgs, func(cfg config, in chan<- error) {
		name := "bat-" + cfg.Event + ".service"
		output, err := exec.Command("systemctl", "is-enabled", name).Output()
		if err != nil || string(output) != "enabled" {
			in <- err
			return
		}
		in <- nil
	})
}

// Present checks if all systemd services are installed.
func (s *Systemd) Present() error {
	cfgs, err := configs()
	if err != nil {
		return err
	}
	if err := s.present(cfgs); err != nil {
		return err
	}
	return nil
}

// Enabled checks if all systemd services are enabled.
func (s *Systemd) Enabled() error {
	cfgs, err := configs()
	if err != nil {
		return err
	}
	if err := s.enabled(cfgs); err != nil {
		return err
	}
	return nil
}

// Remove removes and disables all systemd services created by the
// application.
func (s *Systemd) Remove() error {
	cfgs, err := configs()
	if err != nil {
		return err
	}
	if err := s.remove(cfgs); err != nil {
		return err
	}
	if err := s.disable(cfgs); err != nil {
		return err
	}
	return nil
}

// Write creates all the systemd services required to persist the
// charge limit after restart/hibernation/sleep.
func (s *Systemd) Write() error {
	cfgs, err := configs()
	if err != nil {
		return err
	}
	if err := s.write(cfgs); err != nil {
		return err
	}
	if err := s.enable(cfgs); err != nil {
		return err
	}
	return nil
}

// Disable creates all the systemd services required to persist the
// charge limit after restart/hibernation/sleep and disables them.
func (s *Systemd) Disable() error {
	cfgs, err := configs()
	if err != nil {
		return err
	}
	if err := s.write(cfgs); err != nil {
		return err
	}
	if err := s.disable(cfgs); err != nil {
		return err
	}
	return nil
}
