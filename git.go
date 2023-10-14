package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/mod/semver"
)

func GetGitTags(repoURL, ref string, container *dagger.Container, c *dagger.Client, ctx context.Context) (tags []string, err error) {
	project := c.Git(repoURL, dagger.GitOpts{KeepGitDir: true}).Branch(ref).Tree()

	output, err := container.
		WithMountedDirectory("/REPO", project).
		WithWorkdir("/REPO").
		WithExec([]string{"git", "fetch", "--tags"}).
		WithExec([]string{"git", "tag", "-l", "v*", "--sort", "v:refname"}).
		Stdout(ctx)
	if err != nil {
		err = fmt.Errorf("ERROR: could not get repository: %s; reason: %s", repoURL, err)
		return
	}

	for _, tag := range strings.Split(output, "\n") {
		tag = strings.TrimSpace(tag)
		if !strings.Contains(tag, "-") && tag != "" {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}
	semver.Sort(tags)

	return
}

func GetLatestGitTag(repoURL, ref string, container *dagger.Container, c *dagger.Client, ctx context.Context) (tag string, err error) {
	tags, err := GetGitTags(repoURL, ref, container, c, ctx)
	if err != nil {
		return
	}
	log.Println(tags)
	tag = tags[len(tags)-1]
	return
}
