package login

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testCands = []candidate{
	{"edge", `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`},
	{"chrome", `C:\Program Files\Google\Chrome\Application\chrome.exe`},
}

func pick(t *testing.T, input string, assumeYes, interactive bool) (string, error) {
	t.Helper()
	var out strings.Builder
	return pickFrom(testCands, "", assumeYes, interactive, strings.NewReader(input), &out)
}

func TestPickFrom(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		assumeYes   bool
		interactive bool
		want        string
		wantErr     bool
	}{
		{"回车用默认(Edge)", "\n", false, true, testCands[0].path, false},
		{"选 2 得到 Chrome", "2\n", false, true, testCands[1].path, false},
		{"选 1 得到 Edge", "1\n", false, true, testCands[0].path, false},
		{"q 取消", "q\n", false, true, "", true},
		{"编号越界后重选", "9\n2\n", false, true, testCands[1].path, false},
		{"乱输入后回车", "zzz\n\n", false, true, testCands[0].path, false},
		{"-yes 直接用默认且不读输入", "", true, true, testCands[0].path, false},
		{"非交互(计划任务)直接用默认", "", false, false, testCands[0].path, false},
	}
	for _, c := range cases {
		got, err := pick(t, c.input, c.assumeYes, c.interactive)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: 期望报错，实际得到 %q", c.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 意外报错 %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: 期望 %q，实际 %q", c.name, c.want, got)
		}
	}
}

func TestPickFromAcceptsPath(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip("取不到自身路径")
	}
	got, err := pick(t, exe+"\n", false, true)
	if err != nil {
		t.Fatalf("应接受已存在的路径: %v", err)
	}
	if got != exe {
		t.Fatalf("期望 %q，实际 %q", exe, got)
	}
}

func TestChooseBrowserExplicitSkipsPrompt(t *testing.T) {
	// 显式给出路径时不该弹菜单（输入留空也不会报"读取失败"）
	exe, err := os.Executable()
	if err != nil {
		t.Skip("取不到自身路径")
	}
	got, err := chooseBrowser(exe, "", false, true, strings.NewReader(""), &strings.Builder{})
	if err != nil || got != exe {
		t.Fatalf("显式路径应直接采用：got=%q err=%v", got, err)
	}
	if _, err := chooseBrowser("没有这个浏览器", "", false, false, strings.NewReader(""), &strings.Builder{}); err == nil {
		t.Fatal("未知浏览器名应报错")
	}
}

func TestPutFirstPrefersRemembered(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip("取不到自身路径")
	}
	got := putFirst(testCands, candidate{"catsxp", exe})
	if got[0].path != exe {
		t.Fatalf("上次用过的应排最前，实际 %+v", got[0])
	}
	if len(got) != len(testCands)+1 {
		t.Fatalf("应保留其余候选，实际 %d 个", len(got))
	}
	// 已在列表里的不应重复
	got = putFirst(testCands, candidate{"edge", testCands[0].path})
	if len(got) != len(testCands) {
		t.Fatalf("重复项应被去掉，实际 %d 个", len(got))
	}
}

func TestRememberedRoundTrip(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if got := loadRemembered(cfg); got.Browser != "" || got.ProfileDir != "" {
		t.Fatalf("没有记录时应为空，实际 %+v", got)
	}
	if err := saveRemembered(cfg, `C:\Tools\Catsxp\catsxp.exe`, `C:\Users\x\.quark-nd-disk\browser-profile`); err != nil {
		t.Fatal(err)
	}
	got := loadRemembered(cfg)
	if got.Browser != `C:\Tools\Catsxp\catsxp.exe` || got.ProfileDir != `C:\Users\x\.quark-nd-disk\browser-profile` {
		t.Fatalf("往返不一致：%+v", got)
	}
	if !samePath(got.Browser, `c:\TOOLS\catsxp\CATSXP.EXE`) {
		t.Error("Windows 路径比较应忽略大小写")
	}
}

func TestDefaultProfileDirPerBrowser(t *testing.T) {
	edge := DefaultProfileDir(`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`)
	chrome := DefaultProfileDir(`C:\Program Files\Google\Chrome\Application\chrome.exe`)
	catsxp := DefaultProfileDir(`C:\Tools\Catsxp\catsxp.exe`)
	for label, p := range map[string]string{"edge": edge, "chrome": chrome, "catsxp": catsxp} {
		if !strings.Contains(p, "browser-profile-"+label) {
			t.Errorf("%s 的默认目录不对: %s", label, p)
		}
	}
	if edge == chrome || chrome == catsxp {
		t.Error("不同浏览器必须用不同目录，否则会互相踩 profile")
	}
}

func TestQuarkCookieHeaderDedup(t *testing.T) {
	got, hasPus := quarkCookieHeader([]Cookie{
		{Name: "__pus", Value: "P", Domain: ".quark.cn"},
		{Name: "b-user-id", Value: "device", Domain: ".quark.cn"},
		{Name: "b-user-id", Value: "anon", Domain: "pan.quark.cn"},
		{Name: "other", Value: "x", Domain: "example.com"},
		{Name: "__uid", Value: "U", Domain: ".quark.cn"},
	})
	if !hasPus {
		t.Error("应识别到 __pus")
	}
	if strings.Contains(got, "other=") {
		t.Errorf("非夸克域不该进入：%s", got)
	}
	if strings.Contains(got, "b-user-id=anon") {
		t.Errorf("同名应保留域更宽的那份：%s", got)
	}
	if !strings.Contains(got, "b-user-id=device") {
		t.Errorf("缺少 b-user-id：%s", got)
	}
	if strings.Count(got, "b-user-id=") != 1 {
		t.Errorf("同名应去重：%s", got)
	}
	if !strings.Contains(got, "__pus=P") || !strings.Contains(got, "__uid=U") {
		t.Errorf("缺少关键项：%s", got)
	}
}

func TestBrowserLabel(t *testing.T) {
	// 同时覆盖 Windows 与 Unix 风格路径：标签推导不能依赖平台的路径分隔符语义
	cases := map[string]string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`: "edge",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`:        "chrome",
		`C:\Tools\Catsxp\catsxp.exe`:                                   "catsxp",
		`/opt/apps/catsxp`:                                             "catsxp",
		`/opt/apps/msedge.exe`:                                         "edge",
		"librewolf":                                                    "librewolf",
	}
	for in, want := range cases {
		if got := browserLabel(in); got != want {
			t.Errorf("browserLabel(%s)=%q，期望 %q", in, got, want)
		}
	}
}
