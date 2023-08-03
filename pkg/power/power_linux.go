// Package power - I/O for /sys/class/power_supply
package power

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
)

// Variable represents a /sys/class/power_supply/ device variable.
type Variable uint8

// Entries of /sys/class/power_supply/
const (
	Capacity Variable = iota + 1
	Status
	Threshold
	ChargeFull
	ChargeFullDesign
	EnergyFull
	EnergyFullDesign
)

func (v Variable) String() string {
	switch v {
	case Capacity:
		return "capacity"
	case Status:
		return "status"
	case Threshold:
		return "charge_control_end_threshold"
	case ChargeFull:
		return "charge_full"
	case ChargeFullDesign:
		return "charge_full_design"
	case EnergyFull:
		return "energy_full"
	case EnergyFullDesign:
		return "energy_full_design"
	default:
		return "unrecognised"
	}
}

// dir is the location of the (symlink) of the device in the sysfs
// virtual file system. A glob pattern is used to try to make compatible
// with multiple device manufacturers.
const dir = "/sys/class/power_supply/BAT?/"

var (
	// ErrNotFound indicates virtual file not found
	ErrNotFound = errors.New("power: no battery virtual file found")
	// ErrMultipleFound indicates more than one virtual file found
	ErrMultipleFound = errors.New("power: multiple battery virtual files found")
)

func find(v Variable) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, v.String()))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", ErrNotFound
	}
	if len(matches) > 1 {
		return "", ErrMultipleFound
	}
	return matches[0], nil
}

// Get returns the contents of a virtual file usually located in
// /sys/class/power_supply/BAT?/ and an error otherwise.
func Get(v Variable) (string, error) {
	p, err := find(v)
	if err != nil {
		return "", nil
	}
	contents, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(contents)), nil
}

// Set writes the virtual file usually located in
// /sys/class/power_supply/BAT?/ and returns an error otherwise.
func Set(v Variable, val string) error {
	p, err := find(v)
	if err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(val)
	return err
}
