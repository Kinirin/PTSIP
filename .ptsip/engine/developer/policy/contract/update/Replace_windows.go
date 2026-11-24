package update

import (
	"syscall"
	"unsafe"
)

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func replaceFile(from, to string) error {
	source, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	destination, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	const replaceExisting = 0x1
	const writeThrough = 0x8
	result, _, err := moveFileEx.Call(uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(destination)), replaceExisting|writeThrough)
	if result == 0 {
		return err
	}
	return nil
}

// MOVEFILE_WRITE_THROUGH supplies durability for the Windows replacement.
func syncDirectory(string) error { return nil }
