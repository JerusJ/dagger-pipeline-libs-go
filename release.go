package pipeline

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

// For config, see: https://github.com/semantic-release/semantic-release/blob/master/docs/usage/ci-configuration.md
func RunSemanticRelease(repoDir *dagger.Directory, platform string, c *dagger.Client, ctx context.Context) (err error) {
	secretEnv := ""
	switch platform {
	case "github":
		secretEnv = "GITHUB_TOKEN"
	case "gitlab":
		secretEnv = "GITLAB_TOKEN"
	case "bitbucket":
		secretEnv = "BITBUCKET_TOKEN"
	default:
		err = fmt.Errorf("ERROR: unsupporteed platform, can't run semantic release. Supplieed platform: '%s' ", platform)
		return
	}

	token := os.Getenv(secretEnv)
	if token == "" {
		err = fmt.Errorf("ERROR: empty token for env var: %s", secretEnv)
		return
	}

	// TODO: use *dagger.Secret for token

	cSemantic := c.Container().From("docker.io/node:20.5.1-alpine3.18").
		WithEntrypoint([]string{"sh", "-c"}).
		WithExec([]string{"apk update"}).
		WithExec([]string{"apk add git"}).
		WithExec([]string{"npm install -g semantic-release@latest"}).
		WithExec([]string{"npm install -g @semantic-release/release-notes-generator@latest"}).
		WithExec([]string{"npm install -g @semantic-release/npm@latest"}).
		WithExec([]string{"npm install -g @semantic-release/exec@latest"}).
		WithExec([]string{"npm install -g @semantic-release/changelog@latest"}).
		WithExec([]string{"npm install -g @semantic-release/git@latest"}).
		WithExec([]string{"npm install -g @semantic-release/github@latest"}).
		WithExec([]string{"npm install -g @semantic-release/gitlab@latest"}).
		WithMountedCache("/var/cache/apk", c.CacheVolume("apk_cache")).
		WithMountedDirectory("/WORK/repo", repoDir).
		WithEnvVariable(secretEnv, token).
		WithWorkdir("/WORK/repo")

	// Run Release
	_, err = cSemantic.WithExec([]string{"npx semantic release"}).Stderr(ctx)

	return
}
