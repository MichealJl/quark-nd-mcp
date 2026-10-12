//go:build !windows

package login

import (
	"io"
	"os"
)

// isInteractive 判断是否在终端里运行（非 Windows：字符设备即终端）。
func isInteractive(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
