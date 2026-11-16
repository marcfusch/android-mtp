package main

import "testing"

const ioregFree = `+-o MTP@0  <class IOUSBHostInterface, id 0x1000025d1, registered, matched, active, busy 0 (26 ms), retain 13>
    {
      "bInterfaceClass" = 6
      "bInterfaceSubClass" = 1
      "USB Serial Number" = "1840dd17"
    }
`

const ioregHeld = `+-o IOUSBHostInterface@0  <class IOUSBHostInterface, id 0x100000a01, registered, matched, active, busy 0 (1 ms), retain 9>
  |   "bInterfaceClass" = 3
  |   "UsbExclusiveOwner" = "pid 99, keyboardd"
+-o MTP@0  <class IOUSBHostInterface, id 0x1000025d1, registered, matched, active, busy 0 (26 ms), retain 13>
  |   "bInterfaceClass" = 6
  |   "UsbExclusiveOwner" = "pid 10775, ptpcamerad"
  +-o ptpcamerad  <class AppleUSBHostInterfaceUserClient, id 0x1000025d7, !registered, !matched, active, busy 0, retain 6>
`

func TestParseIoreg(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want usbStatus
	}{
		{"aucune interface", "", usbStatus{}},
		{"interface libre", ioregFree, usbStatus{iface: "0x1000025d1"}},
		{"interface tenue, après une autre interface", ioregHeld, usbStatus{iface: "0x1000025d1", ownerPID: 10775, owner: "ptpcamerad"}},
		{
			"PTP n'est pas MTP",
			`+-o PTP@0  <class IOUSBHostInterface, id 0x100000b02, registered, matched, active, busy 0, retain 6>`,
			usbStatus{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseIoreg(tt.out); got != tt.want {
				t.Errorf("parseIoreg() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
