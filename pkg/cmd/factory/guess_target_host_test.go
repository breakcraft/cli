package factory

import (
	"testing"

	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/gh"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestGuessTargetHost(t *testing.T) {
	tests := []struct {
		name         string
		repoFlag     string
		hostnameFlag bool
		hostname     string
		ghRepoEnv    string
		baseRepo     ghrepo.Interface
		baseRepoErr  error
		wantHost     string
	}{
		{
			// Given a --repo flag with an explicit GHES host
			// When I guess the target host
			// Then it should return the GHES host from the flag
			name:     "repo flag with host takes priority",
			repoFlag: "ghes.example.com/org/repo",
			wantHost: "ghes.example.com",
		},
		{
			// Given a --repo flag without an explicit host
			// When I guess the target host
			// Then it should fall back to the default host
			name:     "repo flag without host uses default",
			repoFlag: "org/repo",
			wantHost: "github.com",
		},
		{
			// Given GH_REPO env var with an explicit GHES host
			// When I guess the target host
			// Then it should return the GHES host from the env var
			name:      "GH_REPO env with host",
			ghRepoEnv: "ghes.example.com/org/repo",
			wantHost:  "ghes.example.com",
		},
		{
			// Given a --hostname flag set to a GHES host
			// And a BaseRepo pointing to github.com
			// When I guess the target host
			// Then it should return the hostname flag value (higher priority)
			name:         "hostname flag takes priority over BaseRepo",
			hostnameFlag: true,
			hostname:     "ghes.example.com",
			baseRepo:     ghrepo.NewWithHost("org", "repo", "github.com"),
			wantHost:     "ghes.example.com",
		},
		{
			// Given a git remote pointing to a GHES host
			// When I guess the target host
			// Then it should return the host from the git remote
			name:     "BaseRepo host from git remote",
			baseRepo: ghrepo.NewWithHost("org", "repo", "ghes.example.com"),
			wantHost: "ghes.example.com",
		},
		{
			// Given no flags, env vars, or git remotes
			// When I guess the target host
			// Then it should fall back to the default host
			name:        "falls back to default host",
			baseRepoErr: assert.AnError,
			wantHost:    "github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ghRepoEnv != "" {
				t.Setenv("GH_REPO", tt.ghRepoEnv)
			}

			cmd := &cobra.Command{Use: "test"}
			if tt.repoFlag != "" || tt.ghRepoEnv != "" {
				cmd.Flags().StringP("repo", "R", "", "")
				if tt.repoFlag != "" {
					cmd.Flags().Set("repo", tt.repoFlag)
				}
			}
			if tt.hostnameFlag {
				cmd.Flags().StringP("hostname", "h", "", "")
				if tt.hostname != "" {
					cmd.Flags().Set("hostname", tt.hostname)
				}
			}

			f := &cmdutil.Factory{
				BaseRepo: func() (ghrepo.Interface, error) {
					if tt.baseRepoErr != nil {
						return nil, tt.baseRepoErr
					}
					if tt.baseRepo != nil {
						return tt.baseRepo, nil
					}
					return nil, assert.AnError
				},
				Config: func() (gh.Config, error) {
					return config.NewBlankConfig(), nil
				},
			}

			got := guessTargetHost(cmd, f)
			assert.Equal(t, tt.wantHost, got)
		})
	}
}
