package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetTerminalBackgroundEmitsOSC11(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	setTerminalBackground(&buf)

	got := buf.String()
	assert.True(t, strings.HasPrefix(got, "\x1b]11;"), "expected OSC 11 prefix, got %q", got)
	assert.Contains(t, strings.ToLower(got), strings.ToLower(terminalBackgroundHex))
	assert.True(t, strings.HasSuffix(got, "\x07"), "expected BEL terminator, got %q", got)
}

func TestResetTerminalBackgroundEmitsOSC111(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	resetTerminalBackground(&buf)

	assert.Equal(t, resetBackgroundOSC, buf.String())
}
