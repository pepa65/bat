# bat v0.8.4
**Manage battery charge limit**

[![Go Report Card](https://goreportcard.com/badge/github.com/pepa65/bat)](https://goreportcard.com/report/github.com/pepa65/bat)
[![GoDoc](https://godoc.org/github.com/pepa65/bat?status.svg)](https://godoc.org/github.com/pepa65/bat)

* Repo: github.com/pepa65/bat
* After: github.com/tshakalekholoane/bat
* License: MIT
* Required: Linux-5.4+ systemd-244+
 
```
bat v0.8.4 - Manage battery charge limit
Repo:  github.com/pepa65/bat
Ref:   https://wiki.archlinux.org/title/Laptop/ASUS#Battery_charge_threshold
Usage: bat <option>
  Options (every option except 's[tatus]' needs root privileges):
    [s[tatus]]       Display charge level, limit, health & persist status.
    l[imit] <int>    Set the charge limit to <int> percent.
    p[ersist]        Install and enable the persist systemd unit files.
    r[emove]         Remove the persist systemd unit files.
    d[isable]        Disable the persist systemd unit files.
    h[elp]           Just display this help text.
    v[ersion]        Just display version information.
```

## About
The goal is to replicate the functionality of the [ASUS Battery Health Charging](https://www.asus.com/us/support/FAQ/1032726/) utility for ASUS laptops on Windows which aims to prolong the battery's life-span <a href="https://electrek.co/2017/09/01/tesla-battery-expert-recommends-daily-battery-pack-charging/"><sup>1</sup></a> <a href="https://batteryuniversity.com/learn/article/how_to_prolong_lithium_based_batteries"><sup>2</sup></a>.

* Linux kernel module: `asus_nb_wmi`
* System variables used: `/sys/class/power_supply/BAT?/`

## Requirements
* **Linux kernel version later than 5.4-rc1** which is the [earliest version to expose the battery charge limit variable](https://github.com/torvalds/linux/commit/7973353e92ee1e7ca3b2eb361a4b7cb66c92abee).
* To persist the battery charge limit setting after restart/hibernation/wake-up, the application relies on **[systemd](https://systemd.io/) version 244 or later** (bundled with most current Linux distributions).

## Disclaimer
This has been reported to only work with some ASUS and [Lenovo ThinkPad](https://github.com/tshakalekholoane/bat/discussions/23) laptops. For Dell systems, see [smbios-utils](https://github.com/dell/libsmbios), particularly the `smbios-battery-ctl` command, or install it using your package manager. For other manufacturers there is also [TLP](https://linrunner.de/tlp/).

## Installation
Precompiled binaries (Linux x86_64) are available from the [GitHub releases page](https://github.com/pepa65/bat/releases), download the [latest here](https://github.com/pepa65/bat/releases/latest/download/bat).

```shell
sudo wget -qO /usr/local/bin/bat github.com/pepa65/bat/releases/latest/download/bat
sudo chmod +x /usr/local/bin/bat
```

Alternatively, the application can be build from source by running the following command in the root directory of this repository. This requires a working version of [Go](https://golang.org/):

`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"`

To install, do: `sudo mv bat /usr/local/bin` (requires root privileges).
You can also rename the binary to something else if another program with the same name already exists.

Or install by simply: `go install github.com/pepa65/bat@latest`

## Examples
### Print the current battery charge level, limit and status
`bat`

Sample output:
```
Level: 45%
Limit: 80%
Health: 85%
Charging
Persist systemd units present: yes
Persist systemd units enabled: yes
```

### Set a battery charge limit in percentage points (requires privileges):
`sudo bat limit 80`

### Undo the battery charge limit (requires privileges):
`sudo bat limit 100`

### Persist the currently set charge limit after restart/hibernation/wake-up (requires privileges):
`sudo bat persist`

### Remove the persist config settings (requires privileges):
`sudo bat remove`

