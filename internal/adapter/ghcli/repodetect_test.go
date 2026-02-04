package ghcli

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
		ok    bool
	}{
		{
			name:  "SSH standard",
			url:   "git@github.com:indrasvat/vivecaka.git",
			owner: "indrasvat", repo: "vivecaka", ok: true,
		},
		{
			name:  "SSH no .git suffix",
			url:   "git@github.com:indrasvat/vivecaka",
			owner: "indrasvat", repo: "vivecaka", ok: true,
		},
		{
			name:  "HTTPS standard",
			url:   "https://github.com/indrasvat/vivecaka.git",
			owner: "indrasvat", repo: "vivecaka", ok: true,
		},
		{
			name:  "HTTPS no .git suffix",
			url:   "https://github.com/indrasvat/vivecaka",
			owner: "indrasvat", repo: "vivecaka", ok: true,
		},
		{
			name:  "HTTP (not HTTPS)",
			url:   "http://github.com/indrasvat/vivecaka.git",
			owner: "indrasvat", repo: "vivecaka", ok: true,
		},
		{
			name:  "with trailing whitespace",
			url:   "  git@github.com:indrasvat/vivecaka.git  \n",
			owner: "indrasvat", repo: "vivecaka", ok: true,
		},
		{
			name: "non-GitHub SSH",
			url:  "git@gitlab.com:indrasvat/vivecaka.git",
			ok:   false,
		},
		{
			name: "non-GitHub HTTPS",
			url:  "https://gitlab.com/indrasvat/vivecaka.git",
			ok:   false,
		},
		{
			name: "empty",
			url:  "",
			ok:   false,
		},
		{
			name: "garbage",
			url:  "not-a-url",
			ok:   false,
		},
		{
			name:  "org with hyphens",
			url:   "git@github.com:my-org/my-repo.git",
			owner: "my-org", repo: "my-repo", ok: true,
		},
		{
			name:  "repo with dots in name",
			url:   "https://github.com/owner/repo.name.git",
			owner: "owner", repo: "repo.name", ok: true,
		},
		{
			name:  "repo with hyphens and underscores",
			url:   "https://github.com/owner/my_repo-name.git",
			owner: "owner", repo: "my_repo-name", ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, ok := ParseRemoteURL(tt.url)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
				return
			}
			if !ok {
				return
			}
			if ref.Owner != tt.owner {
				t.Errorf("owner = %q, want %q", ref.Owner, tt.owner)
			}
			if ref.Name != tt.repo {
				t.Errorf("name = %q, want %q", ref.Name, tt.repo)
			}
		})
	}
}
