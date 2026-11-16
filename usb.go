package main

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	ifaceRe = regexp.MustCompile(`^MTP@\S+\s+<class IOUSBHostInterface, id (0x[0-9a-f]+)`)
	ownerRe = regexp.MustCompile(`"UsbExclusiveOwner" = "pid (\d+), ([^"]+)"`)
)

type usbStatus struct {
	iface    string // identifiant IOKit, nouveau à chaque branchement, vide sans téléphone
	ownerPID int
	owner    string
}

func readUSB() usbStatus {
	out, err := exec.Command("ioreg", "-r", "-c", "IOUSBHostInterface", "-l", "-w0").Output()
	if err != nil {
		return usbStatus{}
	}
	return parseIoreg(string(out))
}

func parseIoreg(out string) usbStatus {
	for _, block := range strings.Split("\n"+out, "\n+-o ") {
		m := ifaceRe.FindStringSubmatch(block)
		if m == nil {
			continue
		}
		s := usbStatus{iface: m[1]}
		if o := ownerRe.FindStringSubmatch(block); o != nil {
			s.ownerPID, _ = strconv.Atoi(o[1])
			s.owner = o[2]
		}
		return s
	}
	return usbStatus{}
}
