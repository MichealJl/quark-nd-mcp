package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCfg(t *testing.T, path, cookie string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(`{"cookie":"`+cookie+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherPicksUpChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeCfg(t, path, "A")
	w := NewWatcher(path)

	if got := w.Cookie(); got != "A" {
		t.Fatalf("首次读取应为 A，实际 %q", got)
	}
	if got := w.Cookie(); got != "A" {
		t.Fatalf("未改动时应命中缓存返回 A，实际 %q", got)
	}

	// 模拟 login 重写配置：必须被察觉
	time.Sleep(1100 * time.Millisecond) // 保证 mtime 变化（部分文件系统时间戳精度为秒）
	writeCfg(t, path, "BBBB")
	if got := w.Cookie(); got != "BBBB" {
		t.Fatalf("文件改动后应返回新值 BBBB，实际 %q", got)
	}

	// 文件被写坏/删掉时，沿用上一次可用的值，不返回空
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got := w.Cookie(); got != "BBBB" {
		t.Fatalf("配置不可读时应沿用上一次的值，实际 %q", got)
	}
}

func TestWatcherMissingFile(t *testing.T) {
	w := NewWatcher(filepath.Join(t.TempDir(), "nope.json"))
	if got := w.Cookie(); got != "" {
		t.Fatalf("配置不存在应返回空串，实际 %q", got)
	}
}

func TestDefaultConfigPathUsesUserHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("取不到用户目录")
	}
	if filepath.Dir(filepath.Dir(DefaultConfigPath)) != home {
		t.Fatalf("默认路径应位于用户目录下，实际 %s", DefaultConfigPath)
	}
}
