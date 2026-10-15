// Package device reads the raw identifier of the machine the SDK runs on: the deviceId of
// every order.
//
// The server binds orders to a device. Nobody configures one: the SDK reads the operating
// system's own machine id and sends it as is, so the same machine always has the same deviceId
// (nothing is hashed or invented).
//
// Sources, first that works: macOS IOPlatformUUID, Linux /etc/machine-id, Windows MachineGuid;
// then the network card's MAC address (aa:bb:cc:dd:ee:ff), or the host name.
package device

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	once sync.Once
	id   string
)

// ID returns the raw machine id of this computer. It is read once per process.
func ID() string {
	once.Do(func() { id = machineSource() })
	return id
}

func machineSource() string {
	var found string
	switch runtime.GOOS {
	case "darwin":
		found = macosUUID()
	case "linux":
		found = linuxMachineID()
	case "windows":
		found = windowsMachineGUID()
	}
	if found != "" {
		return found
	}
	return macOrHost()
}

func run(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output() // #nosec G204 -- fixed commands, no user input
	if err != nil {
		return ""
	}
	return string(out)
}

func macosUUID() string {
	out := run("/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.SplitN(line, "=", 2)
			return strings.Trim(strings.TrimSpace(parts[len(parts)-1]), `"`)
		}
	}
	return ""
}

func linuxMachineID() string {
	for _, name := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		b, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	return ""
}

func windowsMachineGUID() string {
	out := run("reg", "query", `HKLM\SOFTWARE\Microsoft\Cryptography`, "/v", "MachineGuid", "/reg:64")
	return parseRegMachineGUID(out)
}

func parseRegMachineGUID(out string) string {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "MachineGuid" {
			return fields[len(fields)-1]
		}
	}
	return ""
}

func macOrHost() string {
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, i := range ifaces {
			if i.Flags&net.FlagLoopback != 0 || len(i.HardwareAddr) == 0 {
				continue
			}
			if i.HardwareAddr[0]&0x02 != 0 { // locally administered: virtual/random
				continue
			}
			return i.HardwareAddr.String()
		}
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown-host"
}
