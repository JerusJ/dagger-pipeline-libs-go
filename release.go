package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
)

// For config, see: https://github.com/semantic-release/semantic-release/blob/master/docs/usage/ci-configuration.md
func RunSemanticRelease(repoDir *dagger.Directory, platform string, c *dagger.Client, ctx context.Context) (err error) {
	imageNode := "docker.io/node:20.5.1-alpine3.18"
	npmPkgs := []string{
		"semantic-release@latest",
		"semantic-release/release-notes-generator@latest",
		"semantic-release/npm@latest",
		"semantic-release/exec@latest",
		"semantic-release/changelog@latest",
		"semantic-release/git@latest",
	}

	secretEnv := ""
	switch platform {
	case "github":
		secretEnv = "GITHUB_TOKEN"
		npmPkgs = append(npmPkgs, "semantic-release/github@latest")
	case "gitlab":
		secretEnv = "GITLAB_TOKEN"
		npmPkgs = append(npmPkgs, "semantic-release/gitlab@latest")
	case "bitbucket":
		secretEnv = "BITBUCKET_TOKEN"
		npmPkgs = append(npmPkgs, "semantic-release/bitbucket@latest")
	default:
		err = fmt.Errorf("ERROR: unsupporteed platform, can't run semantic release. Supplieed platform: '%s' ", platform)
		return
	}

	// TODO: use *dagger.Secret for token
	token := os.Getenv(secretEnv)
	if token == "" {
		err = fmt.Errorf("ERROR: empty token for env var: %s", secretEnv)
		return
	}

	// Always release under CI
	isDryRun := true
	ciEnv := os.Getenv("CI")
	if ciEnv != "" {
		isDryRun = false
	}

	cSemantic := c.Container().From(imageNode).
		WithEntrypoint([]string{"sh", "-c"}).
		WithMountedCache("/var/cache/apk", c.CacheVolume("apk_cache")).
		WithExec([]string{"apk update"}).
		WithExec([]string{"apk add git git-lfs"})
	for _, pkg := range npmPkgs {
		cSemantic = cSemantic.WithExec([]string{fmt.Sprintf("npm install -g @%s", pkg)})
	}
	cSemantic = cSemantic.
		WithMountedDirectory("/WORK/repo", repoDir).
		WithEnvVariable(secretEnv, token).
		WithEnvVariable("CI", ciEnv).
		WithWorkdir("/WORK/repo")

	// Run Release
	if isDryRun {
		_, err = cSemantic.WithExec([]string{"npx semantic-release"}).Stderr(ctx)
	} else {
		_, err = cSemantic.WithExec([]string{"npx semantic-release --dry-run=false --debug"}).Stderr(ctx)
	}

	return
}
