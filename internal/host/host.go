package host

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alibilge/dirstral-cli/internal/config"
)

type State struct {
	PID       int      `json:"pid"`
	StartedAt string   `json:"started_at"`
	Command   []string `json:"command"`
	WorkDir   string   `json:"workdir"`
	MCPURL    string   `json:"mcp_url"`
}

type UpOptions struct {
	Dir     string
	Listen  string
	MCPPath string
	Port    int
	JSON    bool
}

var urlRe = regexp.MustCompile(`URL:\s*(https?://[^\s]+)`)

func Up(ctx context.Context, opts UpOptions) error {
	baseCommand, baseArgs, workDir, err := resolveDir2MCPCommand()
	if err != nil {
		return err
	}

	args := append([]string{}, baseArgs...)
	args = append(args, "up")
	if opts.Dir != "" {
		args = append(args, "--dir", opts.Dir)
	}
	listen := strings.TrimSpace(opts.Listen)
	if listen == "" && opts.Port > 0 {
		listen = fmt.Sprintf("127.0.0.1:%d", opts.Port)
	}
	if listen != "" {
		args = append(args, "--listen", listen)
	}
	if opts.MCPPath != "" {
		args = append(args, "--mcp-path", opts.MCPPath)
	}
	if opts.JSON {
		args = append(args, "--json")
	}

	cmd := exec.CommandContext(ctx, baseCommand, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	state := State{
		PID:       cmd.Process.Pid,
		StartedAt: time.Now().Format(time.RFC3339),
		Command:   append([]string{baseCommand}, args...),
		WorkDir:   workDir,
	}
	if err := SaveState(state); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	go streamLogs(stdout, "", func(found string) {
		if found != "" {
			state.MCPURL = found
			_ = SaveState(state)
		}
	})
	go streamLogs(stderr, "[dir2mcp] ", nil)

	fmt.Printf("lighthouse: started dir2mcp (pid=%d). Press Ctrl+C to stop.\n", cmd.Process.Pid)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-done:
		_ = ClearState()
		if err != nil {
			return fmt.Errorf("dir2mcp exited: %w", err)
		}
		fmt.Println("lighthouse: dir2mcp stopped")
		return nil
	case <-sigCh:
		fmt.Println("\nlighthouse: shutting down dir2mcp...")
		if err := terminateProcess(cmd.Process.Pid); err != nil {
			return err
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
		}
		_ = ClearState()
		fmt.Println("lighthouse: stopped")
		return nil
	}
}

func Status() error {
	state, err := LoadState()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("lighthouse: no managed dir2mcp process")
			return nil
		}
		return err
	}
	alive := processAlive(state.PID)
	fmt.Printf("lighthouse: pid=%d alive=%t\n", state.PID, alive)
	if state.MCPURL != "" {
		reachable := endpointReachable(state.MCPURL)
		fmt.Printf("lighthouse: mcp=%s reachable=%t\n", state.MCPURL, reachable)
	}
	return nil
}

func Down() error {
	state, err := LoadState()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("lighthouse: nothing to stop")
			return nil
		}
		return err
	}
	if !processAlive(state.PID) {
		_ = ClearState()
		fmt.Println("lighthouse: process already stopped")
		return nil
	}
	if err := terminateProcess(state.PID); err != nil {
		return err
	}
	_ = ClearState()
	fmt.Printf("lighthouse: stopped pid=%d\n", state.PID)
	return nil
}

func SaveState(state State) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func LoadState() (State, error) {
	path, err := statePath()
	if err != nil {
		return State{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

func ClearState() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func statePath() (string, error) {
	dir, err := config.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "host_state.json"), nil
}

func streamLogs(r io.ReadCloser, prefix string, onURL func(string)) {
	defer func() {
		_ = r.Close()
	}()
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		if onURL != nil {
			match := urlRe.FindStringSubmatch(line)
			if len(match) > 1 {
				onURL(match[1])
			}
		}
		fmt.Println(prefix + line)
	}
}

func resolveDir2MCPCommand() (command string, args []string, workDir string, err error) {
	if path, lookErr := exec.LookPath("dir2mcp"); lookErr == nil {
		return path, nil, "", nil
	}
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		return "", nil, "", cwdErr
	}
	tryDirs := []string{
		filepath.Join(cwd, "dir2mcp"),
		filepath.Join(cwd, "..", "dir2mcp"),
	}
	for _, d := range tryDirs {
		if st, statErr := os.Stat(d); statErr == nil && st.IsDir() {
			return "go", []string{"run", "./cmd/dir2mcp"}, d, nil
		}
	}
	return "", nil, "", fmt.Errorf("could not locate dir2mcp binary or source directory")
}

func terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return proc.Kill()
	}
	if err := proc.Signal(syscall.SIGINT); err != nil {
		return err
	}
	return nil
}

// HealthInfo describes the current state of a managed dir2mcp process.
type HealthInfo struct {
	Found     bool   // true if a state file exists
	PID       int    // process ID from state file
	Alive     bool   // true if the process is running
	MCPURL    string // MCP endpoint URL from state
	Reachable bool   // true if the MCP endpoint responds
}

// CheckHealth returns the current health of the managed dir2mcp process.
func CheckHealth() HealthInfo {
	state, err := LoadState()
	if err != nil {
		return HealthInfo{}
	}
	alive := processAlive(state.PID)
	reachable := false
	if state.MCPURL != "" {
		reachable = endpointReachable(state.MCPURL)
	}
	return HealthInfo{
		Found:     true,
		PID:       state.PID,
		Alive:     alive,
		MCPURL:    state.MCPURL,
		Reachable: reachable,
	}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return proc.Signal(syscall.Signal(0)) == nil
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func endpointReachable(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
