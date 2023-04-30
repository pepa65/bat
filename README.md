# `bat`
**Battery charge limit management**

```
Usage:  bat <option>
  Options:
    [s[tatus]]       Display the charge level, limit & status (default).
    l[imit] <int>    Set the charge limit to <int> percent (privileged).
    p[ersist]        Persist the currently set charge limit (privileged).
    c[lear]          Clear persist config of the charge limit (privileged).
    h[elp]           Display only this help text.
    v[ersion]        Display only version information.
```

## About
The goal is to replicate the functionality of the [ASUS Battery Health Charging](https://www.asus.com/us/support/FAQ/1032726/) utility for ASUS laptops on Windows which aims to prolong the battery's life-span <a href="https://electrek.co/2017/09/01/tesla-battery-expert-recommends-daily-battery-pack-charging/"><sup>1</sup></a> <a href="https://batteryuniversity.com/learn/article/how_to_prolong_lithium_based_batteries"><sup>2</sup></a>.
## Requirements
* **Linux kernel version later than 5.4-rc1** which is the [earliest version to expose the battery charge limit variable](https://github.com/torvalds/linux/commit/7973353e92ee1e7ca3b2eb361a4b7cb66c92abee).
* To persist the battery charge limit setting after restart/hibernation/wake-up, the application relies on **[systemd](https://systemd.io/) version 244 or later** (bundled with most Linux distributions).
* To output help or version to a too-small screen, **less** is used as a pager.

## Disclaimer
This has been reported to only work with some ASUS and [Lenovo ThinkPad](https://github.com/tshakalekholoane/bat/discussions/23) laptops. For Dell systems, see [smbios-utils](https://github.com/dell/libsmbios), particularly the `smbios-battery-ctl` command, or install it using your package manager. For other manufacturers there is also [TLP](https://linrunner.de/tlp/).

There have also been some [problems setting the charge limit inside of a virtual machine](https://github.com/tshakalekholoane/bat/issues/3#issuecomment-858581495).

## Installation
Precompiled binaries (Linux x86_64) are available from the [GitHub releases page](https://github.com/pepa65/bat/releases), download the [latest here](https://github.com/pepa65/bat/releases/latest/download/bat).

```shell
sudo wget -qO /usr/local/bin/bat github.com/pepa65/bat/releases/latest/download/bat
chmod +x /usr/local/bin/bat
```

Alternatively, the application can be build from source by running the following command in the root directory of this repository. This requires a working version of [Make](https://www.gnu.org/software/make/) and [Go](https://golang.org/).

```shell
make build
```

You can also rename the binary to something else if another program with the same name already exists.

## Examples
```shell
# Print the current battery charge level, limit and status:
bat

# Set a battery charge limit in percentage points (requires privileges):
sudo bat limit 80

# Undo the battery charge limit (requires privileges):
sudo bat limit 100

# Persist the currently set charge limit after restart/hibernation/wake-up (requires privileges):
sudo bat persist

# Clear the persist config settings (requires privileges):
sudo bat clear
```
