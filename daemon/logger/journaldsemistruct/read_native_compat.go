// +build linux,cgo,!static_build,journald,journald_compat

package journaldsemistruct

// #cgo pkg-config: libsystemd-journal
import "C"
