//go:build windows

package login

import (
	"io"
	"os"
	"syscall"
)

// isInteractive 判断是否在真正的控制台里运行。
// Windows 上不能只看"是不是字符设备"——NUL 设备也是字符设备，
// 计划任务/脚本把 stdin 接到 NUL 时会被误判成交互式，然后弹菜单、读输入直接 EOF。
func isInteractive(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode) == nil
}
