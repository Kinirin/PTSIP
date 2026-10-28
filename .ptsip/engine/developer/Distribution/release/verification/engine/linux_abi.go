package enginewheel

import (
	"bytes"
	"debug/elf"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func VerifyLinuxABI(binary string) error {
	data, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	return linuxABI(data)
}

func linuxABI(data []byte) error {
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer f.Close()
	if f.Machine != elf.EM_X86_64 || f.Class != elf.ELFCLASS64 {
		return fmt.Errorf("manylinux wheel requires an x86_64 ELF executable")
	}
	libs, err := f.ImportedLibraries()
	if err != nil {
		return err
	}
	for _, lib := range libs {
		if !has([]string{"libc.so.6", "libgcc_s.so.1", "libm.so.6", "libpthread.so.0", "librt.so.1", "libdl.so.2", "ld-linux-x86-64.so.2"}, lib) {
			return fmt.Errorf("external library is not admitted for this wheel: %s", lib)
		}
	}
	symbols, err := f.ImportedSymbols()
	if err != nil {
		return err
	}
	for _, symbol := range symbols {
		if strings.HasPrefix(symbol.Version, "GLIBC_") {
			parts := strings.Split(strings.TrimPrefix(symbol.Version, "GLIBC_"), ".")
			if len(parts) < 2 {
				return fmt.Errorf("unregistered GLIBC symbol requirement")
			}
			major, e1 := strconv.Atoi(parts[0])
			minor, e2 := strconv.Atoi(parts[1])
			if e1 != nil || e2 != nil || major != 2 || minor > 39 {
				return fmt.Errorf("ELF requires a GLIBC ABI newer than 2.39")
			}
		}
	}
	return nil
}
