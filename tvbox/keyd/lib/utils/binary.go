package utils

import (
	"encoding/binary"
	"unsafe"
)

var LocalEndian binary.ByteOrder

func isLittleEndian() bool {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb

	return b == 0x04
}

func init() {
	if isLittleEndian() {
		LocalEndian = binary.LittleEndian
	} else {
		LocalEndian = binary.BigEndian
	}
}
