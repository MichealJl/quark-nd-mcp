// Package login 通过本机自带浏览器（默认 Edge）完成夸克网盘登录，并把 Cookie 写入配置。
//
// 为什么走浏览器 + DevTools 协议，而不是别的路：
//   - 夸克 API 只认 __pus 这一个鉴权 Cookie，而它是 HttpOnly 的 ——
//     页面里的 document.cookie 读不到它；
//   - 浏览器自身的 Cookie 库在浏览器运行期间被独占，且新版 Chromium 用
//     App-Bound 加密，外部进程解不开（这就是各类"读库"脚本报
//     Failed to decrypt with DPAPI 的原因）。
//   - 所以让浏览器自己解密并交出结果：启动一个**专属 profile** 的独立窗口
//     （不动用户日常浏览器，也不需要用户先关闭任何东西），用户在可见窗口里正常登录，
//     再用 DevTools 协议把已解密的 Cookie 取回（含 HttpOnly）。
package login

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/MichealJl/quark-nd-mcp/quark"
)

// LoginURL 登录入口：打开夸克网盘即会引导登录。
const LoginURL = "https://pan.quark.cn/"

// Options 是 login 的参数。
type Options struct {
	Browser    string        // 浏览器名（edge/chrome/...）或可执行文件绝对路径；空=探测后交互确认
	ProfileDir string        // 专属 profile 目录；空=按浏览器取默认值
	ConfigPath string        // 配置写入位置；空=默认 ~/.quark-nd-disk/config.json
	Timeout    time.Duration // 等待用户完成登录的上限
	DumpJSON   string        // 可选：额外把 Cookie-Editor 兼容 JSON 写到该路径
	AssumeYes  bool          // 跳过浏览器选择的交互确认（用于计划任务等无人场景）
	In         io.Reader     // 交互输入，默认 os.Stdin
	Out        io.Writer     // 交互输出，默认 os.Stdout
}

// Cookie 是 DevTools 协议返回的 Cookie 元数据（与 Cookie-Editor JSON 兼容）。
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expirationDate,omitempty"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
}

type candidate struct{ name, path string }

func browserCandidates() []candidate {
	var out []candidate
	add := func(name, path string) {
		if path != "" && !strings.Contains(path, "\x00") {
			out = append(out, candidate{name, path})
		}
	}
	switch runtime.GOOS {
	case "windows":
		pf, pfx, la := os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")
		add("edge", filepath.Join(pfx, `Microsoft\Edge\Application\msedge.exe`))
		add("edge", filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`))
		add("edge", filepath.Join(la, `Microsoft\Edge\Application\msedge.exe`))
		add("chrome", filepath.Join(pf, `Google\Chrome\Application\chrome.exe`))
		add("chrome", filepath.Join(pfx, `Google\Chrome\Application\chrome.exe`))
		add("chrome", filepath.Join(la, `Google\Chrome\Application\chrome.exe`))
	case "darwin":
		add("edge", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge")
		add("chrome", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
	default:
		add("edge", "/usr/bin/microsoft-edge")
		add("chrome", "/usr/bin/google-chrome")
		add("chromium", "/usr/bin/chromium")
		add("chromium-browser", "/usr/bin/chromium-browser")
	}
	return out
}

// FindBrowser 解析**显式指定**的浏览器：给的是路径就直接用（校验存在），
// 给的是名字（edge/chrome）就按预设路径探测。
func FindBrowser(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" && (strings.ContainsAny(name, `/\`) || strings.HasSuffix(strings.ToLower(name), ".exe")) {
		if _, err := os.Stat(name); err != nil {
			return "", fmt.Errorf("浏览器路径不可用: %s", name)
		}
		return name, nil
	}
	want := strings.ToLower(name)
	for _, c := range browserCandidates() {
		if c.name != want {
			continue
		}
		if _, err := os.Stat(c.path); err == nil {
			return c.path, nil
		}
	}
	return "", fmt.Errorf("未找到浏览器 %q 的安装路径；可用 -browser <可执行文件路径> 直接指定", name)
}

// detectBrowsers 列出本机实际存在的浏览器，按预设顺序（Edge 优先）。
func detectBrowsers() []candidate {
	var out []candidate
	seen := make(map[string]bool)
	for _, c := range browserCandidates() {
		if seen[c.path] {
			continue
		}
		if _, err := os.Stat(c.path); err == nil {
			seen[c.path] = true
			out = append(out, c)
		}
	}
	return out
}

// chooseBrowser 定下用哪个浏览器：
//   - 显式给了 -browser → 直接用，不再打扰；
//   - 没给 → 列出本机探测到的浏览器让用户确认；上次用过的那一个排在最前并标注，
//     回车即沿用它（这样任何 agent 不指定参数也能接着用本机的既有约定）；
//   - 非交互环境（计划任务、管道）或给了 -yes 时自动取第一项，避免卡住等人。
func chooseBrowser(spec, prefer string, assumeYes, interactive bool, in io.Reader, out io.Writer) (string, error) {
	if strings.TrimSpace(spec) != "" {
		return FindBrowser(spec)
	}
	found := detectBrowsers()
	note := ""
	if prefer = strings.TrimSpace(prefer); prefer != "" {
		if _, err := os.Stat(prefer); err == nil {
			found = putFirst(found, candidate{name: browserLabel(prefer), path: prefer})
			note = "← 上次用的"
		}
	}
	if len(found) == 0 {
		return "", errors.New("未找到可用浏览器（找过 Edge/Chrome 的常见安装路径）；请用 -browser <可执行文件路径> 指定")
	}
	return pickFrom(found, note, assumeYes, interactive, in, out)
}

// putFirst 把 first 放到列表最前，并去掉列表里重复的那个。
func putFirst(list []candidate, first candidate) []candidate {
	out := []candidate{first}
	seen := map[string]bool{strings.ToLower(first.path): true}
	for _, c := range list {
		if seen[strings.ToLower(c.path)] {
			continue
		}
		seen[strings.ToLower(c.path)] = true
		out = append(out, c)
	}
	return out
}

// pickFrom 在候选列表里定下浏览器。assumeYes 或非交互时直接取第一项，不读输入。
func pickFrom(found []candidate, defaultNote string, assumeYes, interactive bool, in io.Reader, out io.Writer) (string, error) {
	if !interactive || assumeYes {
		fmt.Fprintf(out, "未指定浏览器，自动使用 %s（%s）\n", prettyName(found[0]), found[0].path)
		return found[0].path, nil
	}
	if defaultNote == "" {
		defaultNote = "← 默认"
	}

	fmt.Fprintln(out, "请确认用哪个浏览器登录夸克网盘：")
	for i, c := range found {
		tail := ""
		if i == 0 {
			tail = "  " + defaultNote
		}
		fmt.Fprintf(out, "  [%d] %-9s %s%s\n", i+1, prettyName(c), c.path, tail)
	}
	reader := bufio.NewReader(in)
	for attempt := 0; attempt < 5; attempt++ {
		fmt.Fprintf(out, "选择 [1-%d]，直接回车用默认，也可粘贴浏览器 exe 的完整路径（q=取消）: ", len(found))
		line, err := reader.ReadString('\n')
		if err != nil && strings.TrimSpace(line) == "" {
			return "", fmt.Errorf("读取选择失败: %w", err)
		}
		ans := strings.TrimSpace(line)
		switch {
		case ans == "":
			return found[0].path, nil
		case ans == "q" || ans == "Q":
			return "", errors.New("已取消")
		case isDigits(ans):
			if n, _ := strconv.Atoi(ans); n >= 1 && n <= len(found) {
				return found[n-1].path, nil
			}
			fmt.Fprintln(out, "  没有这个编号，请重选。")
		default:
			if _, err := os.Stat(ans); err == nil {
				return ans, nil
			}
			fmt.Fprintf(out, "  没有这个文件：%s\n", ans)
		}
	}
	return "", errors.New("连续多次输入无效，已放弃")
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func prettyName(c candidate) string {
	switch c.name {
	case "edge":
		return "Edge"
	case "chrome":
		return "Chrome"
	case "chromium", "chromium-browser":
		return "Chromium"
	}
	return strings.TrimSuffix(filepath.Base(c.path), filepath.Ext(c.path))
}

// browserLabel 取浏览器的短标签（edge / chrome / catsxp …），用于 profile 目录名。
// 这里自己按最后一个 '/' 或 '\' 切、不借 filepath：`\` 在 Unix 上不是分隔符，
// 用 filepath 会让同一个 Windows 路径在不同平台上得出不同标签。
func browserLabel(exe string) string {
	for _, c := range browserCandidates() {
		if strings.EqualFold(c.path, exe) {
			return c.name
		}
	}
	base := exe
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i] // 去扩展名（i>0 保证 .bashrc 这类整名不被切）
	}
	base = strings.ToLower(base)
	base = strings.TrimPrefix(base, "ms") // msedge → edge
	if base == "" {
		base = "browser"
	}
	return base
}

// DefaultProfileDir 返回该浏览器专属的 profile 目录。
// 按浏览器分目录：不同 Chromium 内核的 profile 不能互开（Cookie 加密密钥不同），
// 混用会读不出已有会话，还可能把 profile 迁移坏。
func DefaultProfileDir(browserExe string) string {
	return filepath.Join(homeDir(), ".quark-nd-disk", "browser-profile-"+browserLabel(browserExe))
}

// remembered 是"这台机器上次用哪个浏览器、哪个 profile 登录过"的记录，
// 存在配置文件旁边（login.json）。有了它，任何 agent 不带参数跑 login
// 也能接着用本机既有的约定，而不是自己另开一套。
type remembered struct {
	Browser    string `json:"browser"`
	ProfileDir string `json:"profile_dir"`
}

func memoryPath(cfgPath string) string {
	return filepath.Join(filepath.Dir(cfgPath), "login.json")
}

func loadRemembered(cfgPath string) remembered {
	var m remembered
	data, err := os.ReadFile(memoryPath(cfgPath))
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

func saveRemembered(cfgPath, browser, profileDir string) error {
	data, err := json.MarshalIndent(remembered{Browser: browser, ProfileDir: profileDir}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(memoryPath(cfgPath), data, 0o600)
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return "."
}

// Run 打开浏览器等用户登录，取回 Cookie、校验通过后写入配置，返回 Cookie 头字符串。
// 校验不过不会覆盖已有配置。
func Run(ctx context.Context, opts Options) (string, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Minute
	}
	in, out := opts.In, opts.Out
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	cfgPath := opts.ConfigPath
	if cfgPath == "" {
		cfgPath = defaultConfigPath()
	}
	last := loadRemembered(cfgPath)

	exe, err := chooseBrowser(opts.Browser, last.Browser, opts.AssumeYes, isInteractive(in), in, out)
	if err != nil {
		return "", err
	}
	profileDir := opts.ProfileDir
	if profileDir == "" {
		// 上次用过同一个浏览器的话，沿用它记着的 profile 目录（可能是老路径）
		if last.ProfileDir != "" && samePath(last.Browser, exe) {
			profileDir = last.ProfileDir
		} else {
			profileDir = DefaultProfileDir(exe)
		}
	}
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return "", fmt.Errorf("创建 profile 目录失败: %w", err)
	}
	port, err := freePort()
	if err != nil {
		return "", fmt.Errorf("申请调试端口失败: %w", err)
	}

	cmd := exec.Command(exe,
		"--user-data-dir="+profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		// 这个 profile 只用来登录：关掉同步与扩展，免得新 profile 首次启动
		// 自己又开出「同步你的浏览数据」「扩展安装完成」之类一堆引导页。
		"--disable-sync",
		"--disable-extensions",
		// 上次若是非正常退出，Chromium 会弹"恢复页面"气泡并恢复旧标签；
		// 两个开关都关掉，避免标签累积。
		"--disable-session-crashed-bubble",
		"--hide-crash-restore-bubble",
		"--remote-debugging-port="+strconv.Itoa(port),
		"--remote-allow-origins=*",
		LoginURL,
	)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动浏览器失败: %w", err)
	}
	// 自己 Wait：进程若提前退出（同 profile 已有窗口时会"转发后退出"），
	// 能立刻发现并给出可操作的提示，而不是干等到超时。
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	defer stopBrowser(cmd, exited)

	fmt.Fprintf(out, "浏览器：%s\n专属配置目录：%s\n配置写入位置：%s\n", exe, profileDir, cfgPath)
	fmt.Fprintln(out, "已打开浏览器窗口（专属配置，不影响你日常使用的浏览器）。")
	fmt.Fprintln(out, "请在这个窗口里登录夸克网盘；登录成功后会检测到，自动继续。")

	wsURL, err := waitCDP(ctx, port, 40*time.Second, exited, profileDir)
	if err != nil {
		return "", err
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return "", fmt.Errorf("连接浏览器调试端口失败: %w", err)
	}
	defer conn.Close()
	c := &cdp{conn: conn}
	// 无论成功、超时还是被取消，都让浏览器自己关掉（刷盘 + 不留窗口）
	defer gracefulClose(c)
	// 只留一个标签页。要清几次：恢复旧会话的标签在启动时出现，而"首次启动引导页"
	// 是浏览器起来之后才陆续被开出来的，所以隔几秒补清一遍。
	c.tidyTabsRepeatedly()

	deadline := time.Now().Add(opts.Timeout)
	var header string
	var loginCookies []Cookie
	for {
		cookies, err := c.allCookies()
		if err != nil {
			return "", err
		}
		h, ok := quarkCookieHeader(cookies)
		if ok {
			header = h
			for _, ck := range cookies {
				d := strings.ToLower(strings.TrimPrefix(ck.Domain, "."))
				if d == "quark.cn" || strings.HasSuffix(d, ".quark.cn") {
					loginCookies = append(loginCookies, ck)
				}
			}
			break
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if time.Now().After(deadline) {
			return "", errors.New("等待登录超时：没有取到夸克登录态（__pus）")
		}
		time.Sleep(2 * time.Second)
	}

	// 先校验再落盘：取到的 Cookie 必须真能调通接口，避免写入半截登录态。
	vctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if _, err := quark.NewClient(header).ListFiles(vctx, "0"); err != nil {
		return "", fmt.Errorf("取到 Cookie 但调用夸克接口校验未通过，已放弃写入配置: %w", err)
	}

	if err := saveConfig(cfgPath, header); err != nil {
		return "", err
	}
	fmt.Fprintf(out, "✅ 登录态校验通过，Cookie 已写入 %s\n", cfgPath)

	// 记下这次用的浏览器与 profile：下次（含别的 agent）不带参数也能接着用
	if err := saveRemembered(cfgPath, exe, profileDir); err != nil {
		fmt.Fprintf(os.Stderr, "提示：未能记录本次浏览器选择（不影响使用）: %v\n", err)
	} else {
		fmt.Fprintf(out, "已记住本次选择：%s + %s（下次不带参数时默认用它）\n", exe, profileDir)
	}

	if opts.DumpJSON != "" {
		if err := dumpJSON(opts.DumpJSON, loginCookies); err != nil {
			fmt.Fprintf(os.Stderr, "提示：导出 JSON 失败（不影响配置）: %v\n", err)
		}
	}
	return header, nil
}

func defaultConfigPath() string {
	return filepath.Join(homeDir(), ".quark-nd-disk", "config.json")
}

// saveConfig 原子写入配置。
// 不留 .bak：新 Cookie 在写盘之前已经真调接口校验过，用不上回滚；
// 在磁盘上多留一份旧凭据只是多一个泄口。
func saveConfig(path, cookie string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	data, err := json.MarshalIndent(map[string]string{"cookie": cookie}, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("写配置失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("写配置失败: %w", err)
	}
	return nil
}

// quarkCookieHeader 拼出夸克域下的 Cookie 头；第二个返回值表示是否拿到登录态关键项 __pus。
func quarkCookieHeader(cookies []Cookie) (string, bool) {
	type entry struct {
		value    string
		priority int
	}
	order := make([]string, 0, len(cookies))
	picked := make(map[string]entry, len(cookies))
	for _, c := range cookies {
		if c.Name == "" {
			continue
		}
		d := strings.ToLower(strings.TrimPrefix(c.Domain, "."))
		if d != "quark.cn" && !strings.HasSuffix(d, ".quark.cn") {
			continue
		}
		// 同名 Cookie 可能同时挂在 .quark.cn 与具体子域上。都写进 Cookie 头会出现
		// 同名不同值、服务端取到哪个不确定，因此按名字去重，优先域更宽的那份。
		prio := 1
		if d == "quark.cn" {
			prio = 2
		}
		if old, ok := picked[c.Name]; ok {
			if old.priority > prio {
				continue
			}
		} else {
			order = append(order, c.Name)
		}
		picked[c.Name] = entry{value: c.Value, priority: prio}
	}
	parts := make([]string, 0, len(order))
	hasPus := false
	for _, n := range order {
		parts = append(parts, n+"="+picked[n].value)
		if n == "__pus" && picked[n].value != "" {
			hasPus = true
		}
	}
	return strings.Join(parts, "; "), hasPus
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitCDP 轮询 /json/version，直到浏览器把调试端口挂起来，返回浏览器级 WebSocket 地址。
// 若我们自己启动的进程提前退出（同 profile 已有窗口时会转发给那个实例然后退出），立即报错。
func waitCDP(ctx context.Context, port int, timeout time.Duration, exited <-chan struct{}, profileDir string) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		select {
		case <-exited:
			return "", fmt.Errorf("浏览器进程刚启动就退出了：多半是该 profile 已有一个窗口在运行（%s）。\n"+
				"请先关掉那个窗口再重跑；若那个窗口里已经登录好，也可以直接完成它再重跑本命令", profileDir)
		default:
		}
		resp, err := client.Get(url)
		if err == nil {
			var v struct {
				WebSocket string `json:"webSocketDebuggerUrl"`
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if resp.StatusCode == 200 && json.Unmarshal(body, &v) == nil && v.WebSocket != "" {
				return v.WebSocket, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", errors.New("浏览器调试端口未就绪")
}

// stopBrowser 等浏览器自己退出（Run 里会先请它优雅关闭、刷盘），超时再强制结束进程树。
func stopBrowser(cmd *exec.Cmd, exited <-chan struct{}) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	select {
	case <-exited:
		return
	case <-time.After(3 * time.Second):
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
		return
	}
	_ = cmd.Process.Kill()
}

// gracefulClose 请浏览器自己关闭。走正常关闭流程会把 Cookie 等状态刷到 profile 里，
// 对下一次登录的"秒过"很关键；失败就交给 stopBrowser 强杀兜底。
func gracefulClose(c *cdp) {
	if c == nil {
		return
	}
	_, _ = c.call("Browser.close", nil)
}

// cdp 是极小的 DevTools 协议客户端：发一条命令、等到对应的响应。
// 会被多个 goroutine 用（主流程 + 定时的标签清理），而 websocket 连接不是并发安全的，
// 所以整条调用串行化。
type cdp struct {
	conn *websocket.Conn
	mu   sync.Mutex
	id   int
}

func (c *cdp) call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.id++
	req := map[string]any{"id": c.id, "method": method}
	if params != nil {
		req["params"] = params
	}
	if err := c.conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("发送 %s 失败: %w", method, err)
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return nil, err
	}
	for {
		var msg struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := c.conn.ReadJSON(&msg); err != nil {
			return nil, fmt.Errorf("读取 %s 响应失败: %w", method, connErr(err))
		}
		if msg.ID != c.id {
			continue // 事件通知或过期响应
		}
		if msg.Error != nil {
			return nil, errors.New("调试协议错误: " + msg.Error.Message)
		}
		return msg.Result, nil
	}
}

// tidyTabsRepeatedly 在开窗后的头十几秒里清几遍多余标签页（后台进行）。
func (c *cdp) tidyTabsRepeatedly() {
	go func() {
		for _, wait := range []time.Duration{0, 2 * time.Second, 4 * time.Second, 6 * time.Second, 10 * time.Second} {
			if wait > 0 {
				time.Sleep(wait)
			}
			c.closeExtraTabs()
		}
	}()
}

// closeExtraTabs 只留一个标签页。
// 登录用的这个 profile 里，标签会从两处冒出来：上次非正常退出后 Chromium 恢复的旧标签，
// 以及新 profile 首次启动时浏览器/扩展自己开的引导页（而且往往都是 quark 页）。
// 所以这里只认一个"要留的"，其余全关 —— 幸存的那个无论哪一页都是夸克页，不影响登录。
func (c *cdp) closeExtraTabs() {
	res, err := c.call("Target.getTargets", nil)
	if err != nil {
		return
	}
	var t struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	if json.Unmarshal(res, &t) != nil {
		return
	}
	type page struct{ id, url string }
	var pages []page
	for _, info := range t.TargetInfos {
		if info.Type == "page" {
			pages = append(pages, page{info.TargetID, info.URL})
		}
	}
	if len(pages) <= 1 {
		return
	}
	keep := pages[0].id
	for _, p := range pages { // 优先留夸克页
		if strings.Contains(p.url, "quark.cn") {
			keep = p.id
			break
		}
	}
	for _, p := range pages {
		if p.id != keep {
			_, _ = c.call("Target.closeTarget", map[string]any{"targetId": p.id})
		}
	}
}

// connErr 把"连接被中断"这类底层错误换成能看懂的说法：
// 最常见的原因就是用户把浏览器窗口关掉了。
func connErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return errors.New("浏览器窗口已关闭")
	}
	low := strings.ToLower(err.Error())
	for _, k := range []string{"aborted", "reset by peer", "forcibly closed", "broken pipe", "eof"} {
		if strings.Contains(low, k) {
			return fmt.Errorf("浏览器窗口已关闭（%s）", err)
		}
	}
	return err
}

// allCookies 取回浏览器全部 Cookie（含 HttpOnly）。Storage.getCookies 为主，
// 旧版内核没有该方法时回退 Network.getAllCookies。
func (c *cdp) allCookies() ([]Cookie, error) {
	var out struct {
		Cookies []Cookie `json:"cookies"`
	}
	firstErr := error(nil)
	for _, m := range []string{"Storage.getCookies", "Network.getAllCookies"} {
		res, err := c.call(m, map[string]any{})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := json.Unmarshal(res, &out); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		return out.Cookies, nil
	}
	return nil, fmt.Errorf("无法读取浏览器 Cookie: %w", firstErr)
}

func dumpJSON(path string, cookies []Cookie) error {
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
