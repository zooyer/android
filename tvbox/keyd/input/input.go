/**
 * @Author: zzy
 * @Email: zhangzhongyuan@didiglobal.com
 * @Description:
 * @File: input.go
 * @Package: input
 * @Version: 1.0.0
 * @Date: 2022/10/8 09:47
 */

package input

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ID struct {
	Bus     uint16 `json:"bus"`
	Vendor  uint16 `json:"vendor"`
	Product uint16 `json:"product"`
	Version uint16 `json:"version"`
}

type Absinfo struct {
	Value      int32 `json:"value"`
	Minimum    int32 `json:"minimum"`
	Maximum    int32 `json:"maximum"`
	Fuzz       int32 `json:"fuzz"`
	Flat       int32 `json:"flat"`
	Resolution int32 `json:"resolution"`
}

type KeymapEntry struct {
	Flags    uint8     `json:"flags"`
	Len      uint8     `json:"len"`
	Index    uint16    `json:"index"`
	Keycode  uint32    `json:"keycode"`
	Scancode [32]uint8 `json:"scancode"`
}

type InputMask struct {
	Type      uint32 `json:"type"`
	CodesSize uint32 `json:"codes_size"`
	CodesPtr  uint64 `json:"codes_ptr"`
}

type Device struct {
	ID       ID     `json:"id"`
	Name     string `json:"name"`
	Phys     string `json:"phys"`
	Sysfs    string `json:"sysfs"`
	Uniq     string `json:"uniq"`
	Handlers string `json:"handlers"`
	Bitmasks uint64 `json:"bitmasks"`
}

func (d Device) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("I: Bus=%04x Vendor=%04x Product=%04x Version=%04x\n", d.ID.Bus, d.ID.Vendor, d.ID.Product, d.ID.Version))
	sb.WriteString(fmt.Sprintf("N: Name=\"%v\"\n", d.Name))
	sb.WriteString(fmt.Sprintf("P: Phys=%v\n", d.Phys))
	sb.WriteString(fmt.Sprintf("S: Sysfs=%v\n", d.Sysfs))
	sb.WriteString(fmt.Sprintf("U: Uniq=%v\n", d.Uniq))
	sb.WriteString(fmt.Sprintf("H: Handlers=%v\n", d.Handlers))
	// TODO bitmask

	return sb.String()
}

func parseUint16(str string, base int) (u uint16, err error) {
	num, err := strconv.ParseUint(str, base, 16)
	if err != nil {
		return
	}

	return uint16(num), nil
}

func ReadInputDevices() (devs []Device, err error) {
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		return
	}

	devices := strings.Split(string(data), "\n\n")
	for _, device := range devices {
		if device = strings.TrimSpace(device); device == "" {
			continue
		}

		lines := strings.Split(device, "\n")
		var device Device
		for _, line := range lines {
			if len(line) < 2 {
				continue
			}

			switch line[:3] {
			case "I: ":
				if _, err = fmt.Sscanf(line, "I: Bus=%04x Vendor=%04x Product=%04x Version=%04x", &device.ID.Bus, &device.ID.Vendor, &device.ID.Product, &device.ID.Version); err != nil {
					return
				}
			case "N: ":
				device.Name = strings.TrimPrefix(line, "N: Name=")
				device.Name = strings.Trim(device.Name, "\"")
			case "P: ":
				device.Phys = strings.TrimPrefix(line, "P: Phys=")
			case "S: ":
				device.Sysfs = strings.TrimPrefix(line, "S: Sysfs=")
			case "U: ":
				device.Uniq = strings.TrimPrefix(line, "U: Uniq=")
			case "H: ":
				device.Handlers = strings.TrimPrefix(line, "H: Handlers=")
			case "B: ":
				line = strings.TrimPrefix(line, "B: ")
				if fields := strings.Split(line, "="); len(fields) > 1 {
					var key, value = fields[0], fields[1]
					_ = value
					// TODO implement
					switch key {
					case "PROP":
					case "EV":
					case "KEY":
					case "REL":
					case "ABS":
					case "MSC":
					case "LED":
					}
				}
			}
		}

		devs = append(devs, device)
	}

	return
}
