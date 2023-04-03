package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
	"github.com/Masterminds/semver"
	"golang.org/x/sync/errgroup"
)


func GetGitTags(repoURL, destDir string, container *dagger.Container, c *dagger.Client, ctx context.Context) (tags []string, err error) {
	cloneRoot := "/REPOS"
	repoRoot := filepath.Join(cloneRoot, destDir)
	container = container.
		WithMountedCache(filepath.Join(cloneRoot, repoRoot), c.CacheVolume("repo_cache_"+destDir)).
		WithWorkdir(cloneRoot).
		WithEntrypoint([]string{})

	output, err := container.
		WithExec([]string{
			"git", "clone", "--progress", repoURL, repoRoot,
		}).
		WithWorkdir(repoRoot).
		WithExec([]string{
			"git", "tag", "-l" ,"v*",
			"--sort", "v:refname",
		}).
		Stdout(ctx)
	if err != nil {
		return
	}

	for _, tag := range strings.Split(output, "\n") {
		tag = strings.TrimSpace(tag)
		if !strings.Contains(tag, "-") && tag != "" {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	return
}


func BuildK8SUtils(c *dagger.Client, ctx context.Context) (err error) {
	baseContainer := c.Container().From("docker.io/alpine:3.17.3")
	gitContainer := c.Container().From("docker.io/alpine/git:2.36.3")
	openshiftTags := []string{}
	k8sTags := []string{}
	sameTags := []string{}

	gitOCPRepo := "https://github.com/openshift/kubernetes.git"
	gitK8SRepo := "https://github.com/kubernetes/kubernetes.git"

	eg, gctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		openshiftTags, err = GetGitTags(gitOCPRepo, "openshift-kubernetes", gitContainer, c, gctx)
		return err
	})
	eg.Go(func() error {
		k8sTags, err = GetGitTags(gitK8SRepo, "kubernetes", gitContainer, c, gctx)
		return err
	})

	err = eg.Wait()
	if err != nil {
		return
	}

	for _, k8sTag := range k8sTags {
		for _, ocpTag := range openshiftTags {
			if k8sTag == ocpTag {
				k8sSemver := semver.MustParse(ocpTag)
				fmt.Printf("%+v\n", k8sSemver)
				sameTags = append(sameTags, ocpTag)
			}
		}
	}

	fmt.Println("Openshift Tags:", openshiftTags)
	fmt.Println("K8S Tags:", k8sTags)
	fmt.Println("Same Tags:", sameTags)

	baseContainer = baseContainer.
		WithMountedDirectory("/download", c.Directory()).
		WithExec([]string{
		"apk", "add",
		"curl",
		"tar",
		"zip",
		"unzip",
	})

	for _, tag := range sameTags[len(sameTags)-5:] {
		_, err = buildK8SUtil(tag, baseContainer, c, ctx)
		if err != nil {
			return err
		}
	}

  return
}


type ContainerBuilder struct {
	URL string
	CheckCommand []string
}
func buildK8SUtil(k8sVersion string, baseContainer *dagger.Container, c *dagger.Client, ctx context.Context) (container *dagger.Container, err error) {
	versionKustomize := "v5.0.1"

	binPath := "/usr/local/bin"
	containerBuilds := []ContainerBuilder{
		{
			URL: fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/amd64/kubectl", k8sVersion),
			CheckCommand: []string{"kubectl", "version", "--client"},
		},
		{
			URL: "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F" + fmt.Sprintf("%s/kustomize_%s_linux_amd64.tar.gz", versionKustomize, versionKustomize),
			CheckCommand: []string{"kustomize", "version"},
		},
	}


	container = baseContainer.WithEntrypoint([]string{})
	for _, build := range containerBuilds {
		fName := filepath.Base(build.URL)
		ext := filepath.Ext(build.URL)
		dest := filepath.Join("/download", fName)

		switch {
		case ext == ".gz":
			container = withDownloadFile(build.URL, dest, container)
			container = container.WithExec([]string{"tar", "-xzf", dest, "-C", "/usr/local/bin"})
		case ext == "":
			container = container.WithFile(filepath.Join(binPath, filepath.Base(build.URL)), c.HTTP(build.URL), dagger.ContainerWithFileOpts{Permissions: 0750})
		default:
			err = fmt.Errorf("Unsupported container extension: %s", ext)
		}

		_, err = container.WithExec(build.CheckCommand).ExitCode(ctx)
		if err != nil {
			return container, err
		}
	}


	return
}

func withDownloadFile(url, dest string, c *dagger.Container) *dagger.Container {
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
