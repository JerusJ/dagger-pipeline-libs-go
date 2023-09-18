package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
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

func BuildK8SUtils(c *dagger.Client, ctx context.Context) (err error) {
	baseContainer := c.Container().From("docker.io/alpine:3.17.3")
	gitContainer := c.Container().From("docker.io/alpine/git:2.36.3").WithEntrypoint([]string{})

	openshiftTags := []string{}
	k8sTags := []string{}
	sameTags := []string{}

	vKustomize := "v5.0.1"
	vHelm := ""

	gitOCPRepo := "https://github.com/openshift/kubernetes.git"
	gitK8SRepo := "https://github.com/kubernetes/kubernetes.git"
	gitHelmRepo := "https://github.com/helm/helm.git"

	eg, gctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		openshiftTags, err = GetGitTags(gitOCPRepo, "master", gitContainer, c, gctx)
		return err
	})
	eg.Go(func() error {
		k8sTags, err = GetGitTags(gitK8SRepo, "master", gitContainer, c, gctx)
		return err
	})
	eg.Go(func() error {
		vHelm, err = GetLatestGitTag(gitHelmRepo, "main", gitContainer, c, gctx)
		return err
	})

	err = eg.Wait()
	if err != nil {
		return
	}

	for _, k8sTag := range k8sTags {
		for _, ocpTag := range openshiftTags {
			if k8sTag == ocpTag {
				sameTags = append(sameTags, ocpTag)
			}
		}
	}

	baseContainer = baseContainer.
		WithMountedDirectory("/download", c.Directory()).
		WithExec([]string{
			"apk", "add",
			"curl",
			"tar",
			"zip",
			"unzip",
		})

	log.Println("Matching OpenShift/Kubernetes upstream tags:", sameTags)
	for _, tag := range sameTags[len(sameTags)-5:] {
		_, err = buildK8SUtil(tag, vKustomize, vHelm, baseContainer, c, ctx)
		if err != nil {
			return err
		}
	}

	return
}

type ContainerBuilder struct {
	URL          string
	CheckCommand []string
}

func buildK8SUtil(vK8S, vKustomize, vHelm string, baseContainer *dagger.Container, c *dagger.Client, ctx context.Context) (container *dagger.Container, err error) {
	containerBuilds := []ContainerBuilder{
		{
			URL:          fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/amd64/kubectl", vK8S),
			CheckCommand: []string{"kubectl", "version", "--client"},
		},
		{
			URL:          "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F" + fmt.Sprintf("%s/kustomize_%s_linux_amd64.tar.gz", vKustomize, vKustomize),
			CheckCommand: []string{"kustomize", "version"},
		},
	}

	baseContainer = baseContainer.WithEntrypoint([]string{})
	for _, build := range containerBuilds {
		baseContainer, err = ContainerWithBinary(c, baseContainer, build.URL)
		if err != nil {
			return container, err
		}

		_, err = baseContainer.WithExec(build.CheckCommand).Stderr(ctx)
		if err != nil {
			return container, err
		}
	}

	return
}

func ContainerWithDownloadFile(url, dest string, c *dagger.Container) *dagger.Container {
	return c.WithExec([]string{
		"curl",
		"--progress-bar",
		"--create-dirs",
		"--connect-timeout", "30",
		"--retry", "300",
		"-fLk",
		url,
		"-o", dest,
	})
}

func ContainerWithBinary(c *dagger.Client, container *dagger.Container, downloadURL string) (*dagger.Container, error) {
	fName := filepath.Base(downloadURL)
	tmpDest := filepath.Join("/download", fName)
	binPath := "/usr/local/bin"

	switch filepath.Ext(downloadURL) {
	case ".gz":
		container = ContainerWithDownloadFile(downloadURL, tmpDest, container)
		container = container.WithExec([]string{"tar", "-xzf", tmpDest, "-C", "/usr/local/bin"})
	case "":
		container = container.WithFile(filepath.Join(binPath, fName), c.HTTP(downloadURL), dagger.ContainerWithFileOpts{Permissions: 0750})
	default:
		return container, fmt.Errorf("ERROR: unsupported container extension for file: %s", fName)
	}

	return container, nil
}
