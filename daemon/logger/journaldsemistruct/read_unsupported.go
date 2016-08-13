// +build !linux !cgo static_build !journald

package journald_semistruct

func (s *journald) Close() error {
	return nil
}
