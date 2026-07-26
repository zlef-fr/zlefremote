//go:build windows

// agent.go — locating, launching and supervising `zlefremote-agent.exe`.
//
// The tray never injects input itself: it drives the same agent binary the CLI
// and the Xfce plugin drive, in `-machine` mode, and renders the `@zr` protocol
// it prints (protocol.go). That keeps one implementation of the transport,
// crypto and input injection across every front-end.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const relayHost = "remote.zlef.fr"

// Runner owns the agent process and pumps its stdout to the UI thread.
type Runner struct {
	Path     string // resolved agent binary ("" = not found)
	Remember bool   // supported by the resolved agent?
	AgentVer string

	cmd  *exec.Cmd
	job  windows.Handle
	mu   sync.Mutex
	dead bool

	lines  chan string // agent stdout, drained on the UI thread
	exited chan struct{}
}

func NewRunner() *Runner {
	r := &Runner{lines: make(chan string, 256)}
	r.Path = findAgent()
	r.Remember, r.AgentVer = probeAgent(r.Path)
	return r
}

func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil && !r.dead
}

// Lines is drained by the window procedure after a wmAgentLine notification.
func (r *Runner) Lines() <-chan string { return r.lines }

// Start launches the agent. notify is posted (from any goroutine) whenever new
// output or an exit is pending; the UI thread then drains Lines().
func (r *Runner) Start(remote, remember bool, notify func(), onExit func()) error {
	r.mu.Lock()
	if r.cmd != nil && !r.dead {
		r.mu.Unlock()
		return nil
	}
	if r.Path == "" {
		r.mu.Unlock()
		return fmt.Errorf("no agent")
	}

	args := []string{"-machine", "-mode", "lan"}
	if remote {
		args[2] = "remote"
	}
	if remote && remember && r.Remember {
		args = append(args, "-remember")
	}

	cmd := exec.Command(r.Path, args...)
	cmd.SysProcAttr = &windows.SysProcAttr{
		// CREATE_NO_WINDOW: the agent is a console binary; without this every
		// start would flash a console window on the user's desktop.
		CreationFlags: windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP,
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.mu.Unlock()
		return err
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		r.mu.Unlock()
		return err
	}
	r.cmd = cmd
	r.dead = false
	r.exited = make(chan struct{})
	exited := r.exited
	// Kill-on-close job: if the tray dies (crash, kill, logoff) the agent must
	// not survive holding an open room.
	r.job = assignJob(cmd)
	r.mu.Unlock()

	go func() {
		var sp LineSplitter
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				sp.Feed(buf[:n], func(line string) {
					select {
					case r.lines <- line:
					default: // UI wedged: drop rather than block the agent's stdout
					}
				})
				notify()
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		cmd.Wait()
		r.mu.Lock()
		r.dead = true
		if r.job != 0 {
			windows.CloseHandle(r.job)
			r.job = 0
		}
		r.mu.Unlock()
		close(exited)
		onExit()
	}()

	return nil
}

// Stop ends the session. Windows has no SIGTERM for a detached console process,
// so we ask the process group nicely (CTRL_BREAK) and fall back to the job kill.
func (r *Runner) Stop() {
	r.mu.Lock()
	cmd, job, exited := r.cmd, r.job, r.exited
	r.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	if job != 0 {
		procTerminateJob.Call(uintptr(job), 0)
	} else {
		cmd.Process.Kill()
	}
	if exited != nil {
		select {
		case <-exited:
		case <-time.After(2 * time.Second):
			cmd.Process.Kill()
		}
	}
}

var (
	procCreateJobObject = kernel32.NewProc("CreateJobObjectW")
	procSetJobInfo      = kernel32.NewProc("SetInformationJobObject")
	procAssignJob       = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJob    = kernel32.NewProc("TerminateJobObject")
)

// assignJob puts the child in a kill-on-close job object. Best effort: an older
// Windows or a nested job just means the child outlives a tray crash.
func assignJob(cmd *exec.Cmd) windows.Handle {
	h, _, _ := procCreateJobObject.Call(0, 0)
	if h == 0 {
		return 0
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	procSetJobInfo.Call(h,
		uintptr(windows.JobObjectExtendedLimitInformation),
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info))
	ph, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false, uint32(cmd.Process.Pid))
	if err != nil {
		windows.CloseHandle(windows.Handle(h))
		return 0
	}
	defer windows.CloseHandle(ph)
	if ok, _, _ := procAssignJob.Call(h, uintptr(ph)); ok == 0 {
		windows.CloseHandle(windows.Handle(h))
		return 0
	}
	return windows.Handle(h)
}

// ── discovery ───────────────────────────────────────────────────────────────

func agentNames() []string {
	return []string{"zlefremote-agent.exe", "zlefremote-agent-windows-amd64.exe",
		"zlefremote-agent-windows-arm64.exe"}
}

// findAgent searches, in order: $ZLEFREMOTE_AGENT, next to this exe, the
// per-user install dir, Program Files, then %PATH%.
func findAgent() string {
	if env := os.Getenv("ZLEFREMOTE_AGENT"); env != "" && isFile(env) {
		return env
	}
	var dirs []string
	if self, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(self))
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		dirs = append(dirs, filepath.Join(la, "ZlefRemote"))
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if pf := os.Getenv(env); pf != "" {
			dirs = append(dirs, filepath.Join(pf, "ZlefRemote"))
		}
	}
	for _, d := range dirs {
		for _, n := range agentNames() {
			if p := filepath.Join(d, n); isFile(p) {
				return p
			}
		}
	}
	for _, n := range agentNames() {
		if p, err := exec.LookPath(n); err == nil {
			return p
		} else {
			_ = p
		}
	}
	return ""
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// probeAgent asks the resolved binary what it supports. Older agents (<1.1.0)
// abort on an unknown -remember flag, which used to look like "Start does
// nothing", so the option is only offered when the help text mentions it.
func probeAgent(path string) (remember bool, version string) {
	if path == "" {
		return false, ""
	}
	run := func(args ...string) string {
		cmd := exec.Command(path, args...)
		cmd.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
		out, _ := cmd.CombinedOutput() // Go's flag package prints usage to stderr
		return string(out)
	}
	help := run("-help")
	remember = strings.Contains(help, "-remember")
	if v := strings.TrimSpace(run("-version")); strings.HasPrefix(v, "zlefremote-agent") {
		version = strings.TrimSpace(strings.TrimPrefix(v, "zlefremote-agent"))
	}
	return remember, version
}

// Rescan re-resolves the agent (after an install or an update).
func (r *Runner) Rescan() {
	r.Path = findAgent()
	r.Remember, r.AgentVer = probeAgent(r.Path)
}

// ── agent install / update ──────────────────────────────────────────────────

type manifest struct {
	Version string `json:"version"`
	Assets  map[string]struct {
		File   string `json:"file"`
		Sha256 string `json:"sha256"`
		URL    string `json:"url"`
	} `json:"assets"`
}

func fetchManifest() (*manifest, error) {
	c := &http.Client{Timeout: 20 * time.Second}
	resp, err := c.Get("https://" + relayHost + "/api/agent/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var m manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// InstallAgent downloads the agent for this architecture into
// %LOCALAPPDATA%\ZlefRemote and verifies its SHA-256 before keeping it. Used
// when the tray is run standalone (portable exe) without the installer.
func InstallAgent() (string, error) {
	m, err := fetchManifest()
	if err != nil {
		return "", err
	}
	key := "windows-" + goArch()
	a, ok := m.Assets[key]
	if !ok {
		return "", fmt.Errorf("no build for %s", key)
	}
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "ZlefRemote")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	c := &http.Client{Timeout: 5 * time.Minute}
	resp, err := c.Get(a.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	tmp := filepath.Join(dir, "zlefremote-agent.exe.new")
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	f.Close()
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, a.Sha256) {
		os.Remove(tmp)
		return "", fmt.Errorf("checksum mismatch")
	}
	final := filepath.Join(dir, "zlefremote-agent.exe")
	os.Remove(final)
	if err := os.Rename(tmp, final); err != nil {
		return "", err
	}
	return final, nil
}

// UpdateAgent runs the agent's own in-place updater (same code path as
// `zlefremote-agent -update`), so the tray never re-implements it.
func UpdateAgent(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("no agent")
	}
	cmd := exec.Command(path, "-update")
	cmd.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
