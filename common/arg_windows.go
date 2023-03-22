//go:build windows
// +build windows

package common

import (
	"syscall"
	"unsafe"
)

func ChangeArg(param string) {

	if kernel32, err := syscall.LoadDLL("Kernel32.dll"); err == nil {
		if GetCommandLineA, err := kernel32.FindProc("GetCommandLineW"); err == nil {
			u, _, _ := GetCommandLineA.Call()

			u16, _ := syscall.UTF16FromString(param)

			for k, v := range u16 {
				*(*byte)(unsafe.Pointer(u + uintptr(k*2+0))) = byte(v)
				*(*byte)(unsafe.Pointer(u + uintptr(k*2+1))) = byte(v >> 8)
			}

			*(*uint16)(unsafe.Pointer(u + uintptr(len(u16)*2+1))) = 0

		}
	}

}
