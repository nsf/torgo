// +build !windows

package make

import "syscall"
import "unsafe"
import "os"

type winsize struct {
	rows uint16
	cols uint16
	xpixels uint16
	ypixels uint16
}

func get_terminal_width() int {
	var sz winsize
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(),
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols)
}