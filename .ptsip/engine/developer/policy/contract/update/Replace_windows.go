package update

import (
	"syscall"
	"unsafe"
)

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

const (
	replaceExisting = 0x1
	writeThrough    = 0x8
)

func replaceFile(from, to string) error {
	return moveFile(from, to, replaceExisting|writeThrough)
}

func moveFile(from, to string, flags uintptr) error {
	source, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	destination, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	result, _, err := moveFileEx.Call(uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(destination)), flags)
	if result == 0 {
		return err
	}
	return nil
}

// MOVEFILE_WRITE_THROUGH supplies durability for both Windows moves.
func syncDirectory(string) error { return nil }
