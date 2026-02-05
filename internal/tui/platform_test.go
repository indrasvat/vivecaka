package tui

import (
	"reflect"
	"runtime"
	"testing"
)

func TestOpenBrowserSpecEmpty(t *testing.T) {
	if _, err := openBrowserSpec("   "); err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestCopyToClipboardSpecEmpty(t *testing.T) {
	if _, err := copyToClipboardSpec("\t\n"); err == nil {
		t.Fatal("expected error for empty clipboard text")
	}
}

func TestOpenBrowserSpec(t *testing.T) {
	url := "https://example.com"
	spec, err := openBrowserSpec("  " + url + "  ")
	if err != nil {
		t.Fatalf("openBrowserSpec: %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		if spec.name != "open" {
			t.Errorf("name = %q, want open", spec.name)
		}
		if !reflect.DeepEqual(spec.args, []string{url}) {
			t.Errorf("args = %v, want %v", spec.args, []string{url})
		}
	case "linux":
		if spec.name == "" {
			t.Error("expected command name for linux")
		}
		if !reflect.DeepEqual(spec.args, []string{url}) {
			t.Errorf("args = %v, want %v", spec.args, []string{url})
		}
	case "windows":
		if spec.name != "cmd" {
			t.Errorf("name = %q, want cmd", spec.name)
		}
		want := []string{"/c", "start", "", url}
		if !reflect.DeepEqual(spec.args, want) {
			t.Errorf("args = %v, want %v", spec.args, want)
		}
	default:
		// On other platforms, openBrowserSpec should have errored.
		if spec.name != "" {
			t.Errorf("unexpected spec on unsupported platform: %+v", spec)
		}
	}
}

func TestCopyToClipboardSpec(t *testing.T) {
	text := "https://example.com/pr/123"
	spec, err := copyToClipboardSpec(text)
	if err != nil {
		t.Fatalf("copyToClipboardSpec: %v", err)
	}
	if spec.stdin != text {
		t.Errorf("stdin = %q, want %q", spec.stdin, text)
	}

	switch runtime.GOOS {
	case "darwin":
		if spec.name != "pbcopy" {
			t.Errorf("name = %q, want pbcopy", spec.name)
		}
	case "linux":
		if spec.name == "" {
			t.Error("expected command name for linux")
		}
	case "windows":
		if spec.name != "cmd" {
			t.Errorf("name = %q, want cmd", spec.name)
		}
	default:
		if spec.name != "" {
			t.Errorf("unexpected spec on unsupported platform: %+v", spec)
		}
	}
}
