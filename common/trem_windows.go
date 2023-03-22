//go:build windows
// +build windows

package common

import (
	"github.com/creack/pty"
	"syscall"
	"unsafe"
)

func SetConsoleVT() {
	if kernel32, err := syscall.LoadDLL("kernel32.dll"); err == nil {
		if GetStdHandle, err := kernel32.FindProc("GetStdHandle"); err == nil {
			if GetConsoleMode, err := kernel32.FindProc("GetConsoleMode"); err == nil {
				if SetConsoleMode, err := kernel32.FindProc("SetConsoleMode"); err == nil {
					//仅限win10
					v := int32(-11)
					hand, _, _ := GetStdHandle.Call(uintptr(v))
					if hand == ^uintptr(0) {
						return
					}

					var dwMode int32
					if res, _, _ := GetConsoleMode.Call(hand, uintptr(unsafe.Pointer(&dwMode))); res == 1 {
						res, _, _ = SetConsoleMode.Call(hand, uintptr(dwMode|0x0004))
						EnableTermVt = res == 1
					}
				}
			}
		}
	}
}

type COORD struct {
	X uint16
	Y uint16
}
type SMALL_RECT struct {
	Left   uint16
	Top    uint16
	Right  uint16
	Bottom uint16
}
type CONSOLE_SCREEN_BUFFER_INFO struct {
	Size              COORD
	CursorPosition    COORD
	Attributes        uint16
	Window            SMALL_RECT
	MaximumWindowSize COORD
}

func GetSize() (size *pty.Winsize) {
	var csbi CONSOLE_SCREEN_BUFFER_INFO

	if kernel32, err := syscall.LoadDLL("kernel32.dll"); err == nil {
		if GetStdHandle, err := kernel32.FindProc("GetStdHandle"); err == nil {
			if GetConsoleScreenBufferInfo, err := kernel32.FindProc("GetConsoleScreenBufferInfo"); err == nil {

				//仅限win10
				v := int32(-11)
				hand, _, _ := GetStdHandle.Call(uintptr(v))
				if hand == ^uintptr(0) {
					return nil
				}
				if res, _, _ := GetConsoleScreenBufferInfo.Call(hand, uintptr(unsafe.Pointer(&csbi))); res == 1 {
					size = &pty.Winsize{}
					size.Cols = csbi.Window.Right - csbi.Window.Left + 1
					size.Rows = csbi.Window.Bottom - csbi.Window.Top + 1
				}
			}
		}
	}
	return
}
