package ghcli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestAdapterImplementsRepoManager(t *testing.T) {
	// Compile-time check is in repomanager.go (var _ domain.RepoManager = (*Adapter)(nil))
	// but this test makes it explicit and exercised.
	adapter := New()
	var _ domain.RepoManager = adapter
	assert.NotNil(t, adapter)
}

func TestAdapterImplementsBothWriterAndRepoManager(t *testing.T) {
	adapter := New()
	var _ domain.PRWriter = adapter
	var _ domain.RepoManager = adapter
	assert.NotNil(t, adapter)
}
