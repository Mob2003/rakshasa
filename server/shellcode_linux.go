//go:build linux || darwin
// +build linux darwin

package server

import "errors"

func shellcodeRun(b []byte) error {
	return errors.New("linux暂不支持")
}
