package pipeline

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"dagger.io/dagger"
)

// Mimic actions/setup-<LANGUAGE> scripts

// See: https://github.com/actions/setup-python/blob/main/src/install-python.ts
// and, for building: https://devguide.python.org/getting-started/setup-building/index.html#linux
func WithPythonFromSource(version string, platform string, container *dagger.Container, c *dagger.Client, ctx context.Context) *dagger.Container {
	c = c.Pipeline("With").Pipeline("Python").Pipeline(version)

	gitPythonUrl := "https://github.com/python/cpython"
	gitPython := c.Git(gitPythonUrl, dagger.GitOpts{KeepGitDir: false}).Branch(version).Tree()

	var cBuilder *dagger.Container
	switch platform {
	case "ubuntu":
		cBuilder = container.
			WithEnvVariable("DEBIAN_FRONTEND", "noninteractive").
			WithEnvVariable("TZ", "Etc/UTC").
			WithMountedCache("/var/cache/apt", c.CacheVolume("apt_cache")).
			WithExec([]string{"apt-get", "update"}).
			WithExec([]string{"apt-get", "install", "-y",
				"git",
				"pkg-config",
				"build-essential",
				"gdb",
				"lcov",
				"pkg-config",
				"libbz2-dev",
				"libffi-dev",
				"libgdbm-dev",
				"libgdbm-compat-dev",
				"liblzma-dev",
				"libncurses5-dev",
				"libreadline6-dev",
				"libsqlite3-dev",
				"libssl-dev",
				"lzma",
				"lzma-dev",
				"tk-dev",
				"uuid-dev",
				"zlib1g-dev",
			})
	case "alpine":
		cBuilder = container.
			WithMountedCache("/var/cache/apk", c.CacheVolume("apk_cache")).
			WithExec([]string{"apk", "update"}).
			WithExec([]string{"apk", "add",
				"build-base",
				"zlib-dev",
				"openssl-dev",
				"libffi-dev",
				"bzip2-dev",
				"xz-dev",
				"sqlite-dev",
				"ncurses-dev",
				"readline-dev",
				"tk-dev",
				"gdbm-dev",
				"db-dev",
				"libxml2-dev",
				"libxslt-dev",
			})
	default:
		log.Fatalf("ERROR: unknown container OS platform: %s", platform)
	}

	cBuilder = cBuilder.
		WithMountedDirectory("/REPO", gitPython).
		WithWorkdir("/REPO").
		WithExec([]string{"./configure"}).
		WithExec([]string{"make"}).
		WithExec([]string{"./python", "--version"}).
		WithExec([]string{"/bin/sh", "-c", `./python -c "import sys; print(sys.version)"`})

	pythonExecutable := cBuilder.File("python")
	cPython := container.WithFile("/usr/local/bin/python", pythonExecutable, dagger.ContainerWithFileOpts{Permissions: 0750})
	return cPython
}

func WithGo(version string, container *dagger.Container, c *dagger.Client, ctx context.Context) *dagger.Container {
	c = c.Pipeline("With").Pipeline("Go").Pipeline(version)

	goURL := fmt.Sprintf("https://go.dev/dl/go%s.linux-amd64.tar.gz", version)
	cGo, err := ContainerWithBinaryAtPath(c, container, goURL, "/usr/local")
	if err != nil {
		log.Fatal(err)
	}

	cGo = cGo.WithEnvVariable(
		"PATH",
		"/usr/local/go/bin:$PATH",
		dagger.ContainerWithEnvVariableOpts{
			Expand: true,
		},
	)

	return cGo
}

func WithTerraform(version string, container *dagger.Container, c *dagger.Client, ctx context.Context) *dagger.Container {
	c = c.Pipeline("With").Pipeline("Terraform").Pipeline(version)

	tfURL := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_amd64.zip", version, version)
	cTf, err := ContainerWithBinary(c, container, tfURL)
	if err != nil {
		log.Fatal(err)
	}

	cTf = cTf.WithEnvVariable("TF_IN_AUTOMATION", "1")

	return cTf
}

type BinaryBuilder struct {
	URL          string
	CheckCommand []string
}

func WithBinaries(binaries []BinaryBuilder, container *dagger.Container, c *dagger.Client) (newContainer *dagger.Container, err error) {
	for _, binary := range binaries {
		newContainer, err = ContainerWithBinary(c, container, binary.URL)
		if err != nil {
			err = fmt.Errorf("ERROR: cannot add binary, reason: %s", err)
			return newContainer, err
		}
		newContainer.WithExec(binary.CheckCommand)
	}

	return
}

func WithKustomize(version string, container *dagger.Container, c *dagger.Client) *dagger.Container {
	c = c.Pipeline("With").Pipeline("Kustomize").Pipeline(version)

	kubectlURL := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/amd64/kubectl", version)
	cKubectl, err := ContainerWithBinary(c, container, kubectlURL)
	if err != nil {
		log.Fatal(err)
	}
	return cKubectl
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
	binPath := "/usr/local/bin"
	return ContainerWithBinaryAtPath(c, container, downloadURL, binPath)
}

func ContainerWithBinaryAtPath(c *dagger.Client, container *dagger.Container, downloadURL string, path string) (*dagger.Container, error) {
	fName := filepath.Base(downloadURL)
	tmpDest := filepath.Join("/download", fName)

	switch filepath.Ext(downloadURL) {
	case ".gz":
		container = ContainerWithDownloadFile(downloadURL, tmpDest, container)
		container = container.WithExec([]string{"tar", "-xzf", tmpDest, "-C", path})
	case ".zip":
		container = ContainerWithDownloadFile(downloadURL, tmpDest, container)
		container = container.WithExec([]string{"unzip", tmpDest, "-d", path})
	case "":
		container = container.WithFile(filepath.Join(path, fName), c.HTTP(downloadURL), dagger.ContainerWithFileOpts{Permissions: 0750})
	default:
		return container, fmt.Errorf("ERROR: unsupported container extension for file: %s", fName)
	}

	return container, nil
}
