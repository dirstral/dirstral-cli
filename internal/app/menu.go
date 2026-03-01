package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	DefaultTerminalWidth = 120
	logoPadding          = 2
	maxLeftPad           = 20
	compactLeftPad       = 2
)

const CompactLogoText = "DIRSTRAL"

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var fullLogoLines = []string{
	"         ▄██████████▄                         ██████╗ ██╗██████╗ ███████╗████████╗██████╗  █████╗ ██╗",
	"      ▄████████████████▄                      ██╔══██╗██║██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║",
	"    ▄████████████████████▄                    ██║  ██║██║██████╔╝███████╗   ██║   ██████╔╝███████║██║",
	"    ██████████████████████                    ██║  ██║██║██╔══██╗╚════██║   ██║   ██╔══██╗██╔══██║██║",
	"    ██████████████████████                    ██████╔╝██║██║  ██║███████║   ██║   ██║  ██║██║  ██║███████╗",
	"    ██████████████████████                    ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝",
}

var mediumLogoLines = []string{
	" ✦ ▄████▄      ██████╗ ██╗██████╗ ███████╗████████╗██████╗  █████╗ ██╗",
	"   █  ████████  ██╔══██╗██║██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║",
	"   ████████████ ██║  ██║██║██████╔╝███████╗   ██║   ██████╔╝███████║██║",
	"   ████████████ ██║  ██║██║██╔══██╗╚════██║   ██║   ██╔══██╗██╔══██║██║",
	"   ████████████ ██████╔╝██║██║  ██║███████║   ██║   ██║  ██║██║  ██║███████╗",
	"   ▀▀▀▀▀▀▀▀▀▀▀ ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝",
}

type StartChoice string

const (
	ChoiceBreeze     StartChoice = "Breeze"
	ChoiceTempest    StartChoice = "Tempest"
	ChoiceLighthouse StartChoice = "Lighthouse"
	ChoiceQuit       StartChoice = "Quit"
)

type menuOption struct {
	Label string
	Value string
}

type menuSpec struct {
	Title        string
	Intro        []string
	Options      []menuOption
	QuitValue    string
	ShowLogo     bool
	ControlsText string
}

type menuKey int

const (
	keyNone menuKey = iota
	keyUp
	keyDown
	keySelect
	keyQuit
)

func ShowStartScreen() (StartChoice, error) {
	items := StartMenuItems()
	options := make([]menuOption, 0, len(items))
	for _, item := range items {
		options = append(options, menuOption{Label: item, Value: item})
	}

	width := TerminalWidth()
	intro := startupIntro(width)
	if shouldAnimateStartup() {
		animateStartup(width)
	}

	choice, err := runInteractiveMenu(menuSpec{
		Title:        "Welcome to dirstral",
		Intro:        intro,
		Options:      options,
		QuitValue:    string(ChoiceQuit),
		ShowLogo:     true,
		ControlsText: "Controls: arrows navigate, Enter select, q/esc back",
	})
	if err != nil {
		return ChoiceQuit, err
	}
	return StartChoice(choice), nil
}

func shouldAnimateStartup() bool {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("DIRSTRAL_ANIMATION")))
	switch mode {
	case "1", "on", "true":
		return true
	default:
		return false
	}
}

func animateStartup(width int) {
	clearScreen()
	logo := RenderLogo(width)
	for _, line := range strings.Split(logo, "\n") {
		fmt.Println(line)
		time.Sleep(18 * time.Millisecond)
	}
	time.Sleep(120 * time.Millisecond)
	clearScreen()
}

func StartMenuItems() []string {
	return []string{string(ChoiceBreeze), string(ChoiceTempest), string(ChoiceLighthouse), string(ChoiceQuit)}
}

func runInteractiveMenu(spec menuSpec) (string, error) {
	if len(spec.Options) == 0 {
		return spec.QuitValue, nil
	}
	if !isInteractiveTerminal() {
		return runLineMenu(spec)
	}

	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return spec.QuitValue, err
	}
	defer func() {
		_ = term.Restore(fd, state)
	}()

	reader := bufio.NewReader(os.Stdin)
	index := 0
	for {
		renderMenu(spec, index)
		key, err := readMenuKey(reader, fd)
		if err != nil {
			if err == io.EOF {
				return spec.QuitValue, nil
			}
			return spec.QuitValue, err
		}
		switch key {
		case keyUp:
			index = (index - 1 + len(spec.Options)) % len(spec.Options)
		case keyDown:
			index = (index + 1) % len(spec.Options)
		case keySelect:
			return spec.Options[index].Value, nil
		case keyQuit:
			return spec.QuitValue, nil
		}
	}
}

func renderMenu(spec menuSpec, selected int) {
	width := TerminalWidth()
	tier := ChooseTier(width)

	// Build the entire frame in a buffer. In raw terminal mode \n is not
	// translated to \r\n (OPOST is off), so we do the replacement ourselves
	// before writing to stdout in one shot.
	var buf strings.Builder

	buf.WriteString(ansiClear) // clear screen + home cursor

	if spec.ShowLogo {
		buf.WriteString(RenderLogo(width))
		buf.WriteByte('\n')
	}

	// Collect all content lines below the logo into a single list so they
	// share one uniform left-margin when centered as a block.
	var body []string

	if strings.TrimSpace(spec.Title) != "" {
		body = append(body, paint(spec.Title, colorBrandStrong, colorBold))
	}
	for _, line := range spec.Intro {
		body = append(body, paint(line, colorMuted))
	}
	body = append(body, "") // blank separator before options

	for i, item := range spec.Options {
		if i == selected {
			body = append(body, paint("> "+item.Label, colorBrandStrong, colorBold))
		} else {
			body = append(body, paint("  "+item.Label, colorMuted))
		}
	}

	body = append(body, "") // blank separator before controls
	controls := spec.ControlsText
	if controls == "" {
		controls = "Controls: arrows navigate, Enter select, q/esc back"
	}
	body = append(body, paint(controls, colorSubtle))

	if tier == LogoCompact {
		for _, line := range body {
			buf.WriteString(padLine(line, compactLeftPad))
			buf.WriteByte('\n')
		}
	} else {
		for _, line := range centerBlockLines(body, width) {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}

	// Replace \n with \r\n so the output renders correctly in raw mode
	// where the terminal does not perform output post-processing.
	fmt.Print(strings.ReplaceAll(buf.String(), "\n", "\r\n"))
}

func runLineMenu(spec menuSpec) (string, error) {
	clearScreen()
	if spec.ShowLogo {
		fmt.Println(RenderLogo(TerminalWidth()))
	}
	if strings.TrimSpace(spec.Title) != "" {
		fmt.Println(spec.Title)
	}
	for _, line := range spec.Intro {
		fmt.Println(line)
	}
	fmt.Println()
	for i, item := range spec.Options {
		fmt.Printf("  %d) %s\n", i+1, item.Label)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Select [1-%d] or q: ", len(spec.Options))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return spec.QuitValue, nil
			}
			return spec.QuitValue, err
		}
		input := strings.TrimSpace(strings.ToLower(line))
		if input == "q" || input == "quit" || input == "esc" {
			return spec.QuitValue, nil
		}
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(spec.Options) {
			continue
		}
		return spec.Options[n-1].Value, nil
	}
}

func readMenuKey(reader *bufio.Reader, fd int) (menuKey, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return keyNone, err
	}

	switch b {
	case 'q', 'Q', 3:
		return keyQuit, nil
	case '\r', '\n':
		return keySelect, nil
	case 27:
		ready, err := inputReady(fd, 20)
		if err != nil || !ready {
			return keyQuit, nil
		}
		b2, err := reader.ReadByte()
		if err != nil {
			return keyQuit, nil
		}
		if b2 != '[' {
			return keyQuit, nil
		}
		ready, err = inputReady(fd, 20)
		if err != nil || !ready {
			return keyQuit, nil
		}
		b3, err := reader.ReadByte()
		if err != nil {
			return keyQuit, nil
		}
		switch b3 {
		case 'A':
			return keyUp, nil
		case 'B':
			return keyDown, nil
		default:
			return keyQuit, nil
		}
	default:
		return keyNone, nil
	}
}

func inputReady(fd, timeoutMillis int) (bool, error) {
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, timeoutMillis)
	if err != nil {
		return false, err
	}
	return n > 0 && (fds[0].Revents&unix.POLLIN) != 0, nil
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

type LogoTier int

const (
	LogoFull    LogoTier = iota // wide terminals: full logo with large folder
	LogoMedium                  // medium terminals: compact folder + block text
	LogoCompact                 // narrow terminals: plain styled text
)

func ChooseTier(width int) LogoTier {
	fullWidth := maxVisibleWidth(NormalizeLeftSpacing(fullLogoLines))
	if width >= fullWidth+logoPadding {
		return LogoFull
	}
	medWidth := maxVisibleWidth(mediumLogoLines)
	if width >= medWidth+logoPadding {
		return LogoMedium
	}
	return LogoCompact
}

func maxVisibleWidth(lines []string) int {
	max := 0
	for _, line := range lines {
		if w := visibleWidth(line); w > max {
			max = w
		}
	}
	return max
}

func RenderLogo(width int) string {
	tier := ChooseTier(width)
	switch tier {
	case LogoCompact:
		return padLine(paint(CompactLogoText, colorBrandStrong, colorBold), compactLeftPad)
	case LogoMedium:
		styled := make([]string, 0, len(mediumLogoLines))
		for i, line := range mediumLogoLines {
			style := colorBrandStrong
			if i >= len(mediumLogoLines)-2 {
				style = colorBrand
			}
			styled = append(styled, paint(line, style))
		}
		return strings.Join(centerBlockLines(styled, width), "\n")
	default:
		lines := NormalizeLeftSpacing(fullLogoLines)
		styled := make([]string, 0, len(lines))
		for i, line := range lines {
			style := colorBrandStrong
			if i >= len(lines)-2 {
				style = colorBrand
			}
			styled = append(styled, paint(line, style))
		}
		return strings.Join(centerBlockLines(styled, width), "\n")
	}
}

func padLine(line string, pad int) string {
	if pad <= 0 {
		return line
	}
	return strings.Repeat(" ", pad) + line
}

func startupIntro(width int) []string {
	if ChooseTier(width) == LogoCompact {
		return []string{
			"Modes: Breeze (chat), Tempest (voice), Lighthouse (host MCP)",
			"Tip: Lighthouse first is the fastest demo path.",
		}
	}
	return []string{
		"Launch mode: Breeze (chat), Tempest (voice), Lighthouse (host MCP)",
		"Tip: Lighthouse first is the fastest demo path.",
	}
}

func centerBlockLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return nil
	}
	maxWidth := 0
	for _, line := range lines {
		if w := visibleWidth(line); w > maxWidth {
			maxWidth = w
		}
	}
	pad := 0
	if width > maxWidth {
		pad = clampLeftPad((width - maxWidth) / 2)
	}
	out := make([]string, len(lines))
	left := strings.Repeat(" ", pad)
	for i, line := range lines {
		out[i] = left + line
	}
	return out
}

func clampLeftPad(n int) int {
	if n < 0 {
		return 0
	}
	if n > maxLeftPad {
		return maxLeftPad
	}
	return n
}

func visibleWidth(s string) int {
	plain := ansiPattern.ReplaceAllString(s, "")
	return len([]rune(plain))
}

func NormalizeLeftSpacing(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		dup := make([]string, len(lines))
		copy(dup, lines)
		return dup
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= minIndent {
			out[i] = line[minIndent:]
		} else {
			out[i] = line
		}
	}
	return out
}

func clearScreen() {
	fmt.Print(ansiClear)
}

func TerminalWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	if raw := strings.TrimSpace(os.Getenv("COLUMNS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return DefaultTerminalWidth
}
