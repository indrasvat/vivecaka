package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type cmdSpec struct {
	name  string
	args  []string
	stdin string
}

var lookPath = exec.LookPath

var openBrowser = func(url string) error {
	spec, err := openBrowserSpec(url)
	if err != nil {
		return err
	}
	cmd := exec.Command(spec.name, spec.args...)
	return cmd.Run()
}

var copyToClipboard = func(text string) error {
	spec, err := copyToClipboardSpec(text)
	if err != nil {
		return err
	}
	cmd := exec.Command(spec.name, spec.args...)
	cmd.Stdin = strings.NewReader(spec.stdin)
	return cmd.Run()
}

func openBrowserSpec(url string) (cmdSpec, error) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return cmdSpec{}, errors.New("empty url")
	}

	switch runtime.GOOS {
	case "darwin":
		return cmdSpec{name: "open", args: []string{trimmed}}, nil
	case "linux":
		path, err := lookPath("xdg-open")
		if err != nil {
			return cmdSpec{}, fmt.Errorf("xdg-open not found: %w", err)
		}
		return cmdSpec{name: path, args: []string{trimmed}}, nil
	case "windows":
		return cmdSpec{name: "cmd", args: []string{"/c", "start", "", trimmed}}, nil
	default:
		return cmdSpec{}, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func copyToClipboardSpec(text string) (cmdSpec, error) {
	if strings.TrimSpace(text) == "" {
		return cmdSpec{}, errors.New("empty clipboard text")
	}

	switch runtime.GOOS {
	case "darwin":
		return cmdSpec{name: "pbcopy", stdin: text}, nil
	case "linux":
		if path, err := lookPath("xclip"); err == nil {
			return cmdSpec{name: path, args: []string{"-selection", "clipboard"}, stdin: text}, nil
		}
		if path, err := lookPath("xsel"); err == nil {
			return cmdSpec{name: path, args: []string{"--clipboard", "--input"}, stdin: text}, nil
		}
		return cmdSpec{}, errors.New("clipboard helper not found: install xclip or xsel")
	case "windows":
		return cmdSpec{name: "cmd", args: []string{"/c", "clip"}, stdin: text}, nil
	default:
		return cmdSpec{}, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
