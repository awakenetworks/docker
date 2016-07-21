// +build linux,cgo,!static_build,journald,!journald_compat

package journald_semistruct

// #cgo pkg-config: libsystemd
import "C"
