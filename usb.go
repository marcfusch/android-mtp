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

// usbStatus décrit l'interface MTP du téléphone telle que la voit IOKit.
type usbStatus struct {
	iface    string // identifiant IOKit, nouveau à chaque énumération, vide sans téléphone
	ownerPID int
	owner    string // processus qui tient l'interface en exclusivité
}

func readUSB() usbStatus {
	out, err := exec.Command("ioreg", "-r", "-c", "IOUSBHostInterface", "-l", "-w0").Output()
	if err != nil {
		return usbStatus{}
	}
	return parseIoreg(string(out))
}

// parseIoreg lit la sortie de ioreg -r -c IOUSBHostInterface -l, où chaque interface commence
// par +-o en début de ligne.
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
