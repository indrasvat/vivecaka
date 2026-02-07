package tui

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenBrowserSpecEmpty(t *testing.T) {
	_, err := openBrowserSpec("   ")
	assert.Error(t, err, "expected error for empty url")
}

func TestCopyToClipboardSpecEmpty(t *testing.T) {
	_, err := copyToClipboardSpec("\t\n")
	assert.Error(t, err, "expected error for empty clipboard text")
}

func TestOpenBrowserSpec(t *testing.T) {
	url := "https://example.com"
	spec, err := openBrowserSpec("  " + url + "  ")
	require.NoError(t, err)

	switch runtime.GOOS {
	case "darwin":
		assert.Equal(t, "open", spec.name)
		assert.True(t, reflect.DeepEqual(spec.args, []string{url}))
	case "linux":
		assert.NotEmpty(t, spec.name, "expected command name for linux")
		assert.True(t, reflect.DeepEqual(spec.args, []string{url}))
	case "windows":
		assert.Equal(t, "cmd", spec.name)
		want := []string{"/c", "start", "", url}
		assert.True(t, reflect.DeepEqual(spec.args, want))
	default:
		// On other platforms, openBrowserSpec should have errored.
		assert.Empty(t, spec.name, "unexpected spec on unsupported platform")
	}
}

func TestCopyToClipboardSpec(t *testing.T) {
	text := "https://example.com/pr/123"
	spec, err := copyToClipboardSpec(text)

	// On Linux CI, clipboard tools (xclip/xsel) may not be installed.
	if runtime.GOOS == "linux" && err != nil {
		t.Skipf("skipping: %v", err)
	}

	require.NoError(t, err)
	assert.Equal(t, text, spec.stdin)

	switch runtime.GOOS {
	case "darwin":
		assert.Equal(t, "pbcopy", spec.name)
	case "linux":
		assert.NotEmpty(t, spec.name, "expected command name for linux")
	case "windows":
		assert.Equal(t, "cmd", spec.name)
	default:
		assert.Empty(t, spec.name, "unexpected spec on unsupported platform")
	}
}
