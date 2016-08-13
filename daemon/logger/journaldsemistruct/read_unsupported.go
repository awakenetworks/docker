// +build !linux !cgo static_build !journald

package journaldsemistruct

func (s *journald) Close() error {
	return nil
}
