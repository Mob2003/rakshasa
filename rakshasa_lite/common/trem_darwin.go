package common

import (
	"github.com/creack/pty"
	"os"
)
func SetConsoleVT() {}
func GetSize() *pty.Winsize {
	size, _ := pty.GetsizeFull(os.Stdin)
	return size
}