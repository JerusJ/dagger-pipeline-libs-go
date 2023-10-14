package pipeline

import (
	"context"
	"fmt"
	"log"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

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

func buildK8SUtil(vK8S, vKustomize, vHelm string, baseContainer *dagger.Container, c *dagger.Client, ctx context.Context) (container *dagger.Container, err error) {
	containerBuilds := []BinaryBuilder{
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
	container, err = WithBinaries(containerBuilds, baseContainer, c)
	return
}
