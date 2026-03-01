package tempest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alibilge/dirstral-cli/internal/mcp"
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	MCPURL    string
	Transport string
	Model     string
	Voice     string
	Device    string
	Mute      bool
	Verbose   bool
	BaseURL   string
}

func Run(ctx context.Context, opts Options) error {
	if err := preflight(opts); err != nil {
		return err
	}

	if strings.TrimSpace(opts.Transport) == "" {
		opts.Transport = "streamable-http"
	}
	if strings.TrimSpace(opts.Model) == "" {
		opts.Model = "mistral-small-latest"
	}

	client := mcp.NewWithTransport(opts.MCPURL, opts.Transport, opts.Verbose)
	defer func() {
		_ = client.Close()
	}()
	if err := client.Initialize(ctx); err != nil {
		return err
	}

	p := tea.NewProgram(initialModel(ctx, client, opts), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func recordAudio(ctx context.Context, device string) (string, error) {
	out := filepath.Join(os.TempDir(), fmt.Sprintf("dirstral-%d.wav", time.Now().UnixNano()))
	var attempts [][]string
	switch runtime.GOOS {
	case "darwin":
		input, err := resolveMacInputDevice(device)
		if err != nil {
			return "", err
		}
		attempts = append(attempts, []string{"-y", "-f", "avfoundation", "-i", input, "-t", "6", "-ac", "1", "-ar", "16000", out})
	case "linux":
		if strings.TrimSpace(device) != "" {
			attempts = append(attempts, []string{"-y", "-f", "alsa", "-i", strings.TrimSpace(device), "-t", "6", "-ac", "1", "-ar", "16000", out})
		} else {
			attempts = append(attempts,
				[]string{"-y", "-f", "alsa", "-i", "default", "-t", "6", "-ac", "1", "-ar", "16000", out},
				[]string{"-y", "-f", "pulse", "-i", "default", "-t", "6", "-ac", "1", "-ar", "16000", out},
			)
		}
	default:
		return "", fmt.Errorf("unsupported OS for quick mic recording: %s", runtime.GOOS)
	}

	var lastErr error
	for _, args := range attempts {
		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return out, nil
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("failed to capture audio, last error: %w", lastErr)
}

func transcribeElevenLabs(ctx context.Context, baseURL, audioPath string) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("ELEVENLABS_API_KEY is required")
	}
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	if err := writer.WriteField("model_id", "scribe_v1"); err != nil {
		return "", err
	}
	fw, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", err
	}
	f, err := os.Open(audioPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/speech-to-text", buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("stt http %d: %s", resp.StatusCode, string(body))
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	text, _ := parsed["text"].(string)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("empty transcript")
	}
	return text, nil
}

func synthesizeElevenLabs(ctx context.Context, baseURL, voice, text string) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("ELEVENLABS_API_KEY is required")
	}
	body := map[string]any{
		"text":     text,
		"model_id": "eleven_multilingual_v2",
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/text-to-speech/"+voice, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("tts http %d: %s", resp.StatusCode, string(data))
	}
	out := filepath.Join(os.TempDir(), fmt.Sprintf("dirstral-tts-%d.mp3", time.Now().UnixNano()))
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func playAudio(ctx context.Context, path string) error {
	var cmd *exec.Cmd
	if _, err := exec.LookPath("afplay"); err == nil {
		cmd = exec.CommandContext(ctx, "afplay", path)
	} else if _, err := exec.LookPath("ffplay"); err == nil {
		cmd = exec.CommandContext(ctx, "ffplay", "-nodisp", "-autoexit", path)
	} else {
		return fmt.Errorf("no playback binary found (afplay or ffplay)")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func preflight(opts Options) error {
	var missing []string

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		missing = append(missing, "ffmpeg (required for mic recording)")
	}
	if strings.TrimSpace(os.Getenv("ELEVENLABS_API_KEY")) == "" {
		missing = append(missing, "ELEVENLABS_API_KEY environment variable")
	}
	if !opts.Mute {
		hasPlayer := false
		if _, err := exec.LookPath("afplay"); err == nil {
			hasPlayer = true
		}
		if _, err := exec.LookPath("ffplay"); err == nil {
			hasPlayer = true
		}
		if !hasPlayer {
			missing = append(missing, "audio player: afplay (macOS) or ffplay (Linux)")
		}
	}

	if len(missing) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("Tempest requires the following to be available:\n")
	for _, m := range missing {
		b.WriteString("  - " + m + "\n")
	}
	b.WriteString("\nSetup hints:\n")
	b.WriteString("  ffmpeg:           brew install ffmpeg (macOS) or apt install ffmpeg (Linux)\n")
	b.WriteString("  ELEVENLABS_API_KEY: export ELEVENLABS_API_KEY=your-key (https://elevenlabs.io/app/settings/api-keys)\n")
	b.WriteString("  audio player:     ffmpeg includes ffplay; macOS has afplay built-in\n")
	b.WriteString("  mute mode:        use --mute to skip audio playback")
	return fmt.Errorf("%s", b.String())
}

func resolveMacInputDevice(device string) (string, error) {
	trimmed := strings.TrimSpace(device)
	if trimmed == "" {
		return ":0", nil
	}
	if _, err := strconv.Atoi(trimmed); err == nil {
		return ":" + trimmed, nil
	}
	devices, err := listMacInputDevices()
	if err != nil {
		return "", fmt.Errorf("unable to resolve --device %q: %w", trimmed, err)
	}
	needle := strings.ToLower(trimmed)
	for idx, name := range devices {
		if strings.Contains(strings.ToLower(name), needle) {
			return fmt.Sprintf(":%d", idx), nil
		}
	}
	return "", fmt.Errorf("input device %q not found, list devices with: ffmpeg -f avfoundation -list_devices true -i \"\"", trimmed)
}

func listMacInputDevices() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return nil, err
		}
	}
	lines := strings.Split(string(out), "\n")
	devices := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "AVFoundation input device") {
			continue
		}
		open := strings.Index(line, "] ")
		if open == -1 || open+2 >= len(line) {
			continue
		}
		name := strings.TrimSpace(line[open+2:])
		if name != "" {
			devices = append(devices, name)
		}
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no input devices found")
	}
	return devices, nil
}
