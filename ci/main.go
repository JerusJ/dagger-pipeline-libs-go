package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"

	pipeline "github.com/jerusj/dagger-pipeline-libs-go/v2"
)

var (
	flagAll       = flag.Bool("all", false, "")
	flagContainer = flag.Bool("containers", false, "")
	flagSetup     = flag.Bool("setup", false, "")
	flagRelease   = flag.Bool("release", false, "")
)

func main() {
	if err := runPipelines(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func getParentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("ERROR: could not get working dir: %s", err)
	}
	parentDir := filepath.Dir(wd)
	return parentDir
}

func runPipelines(ctx context.Context) (err error) {
	flag.Parse()

	// initialize Dagger client
	c, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))

	if err != nil {
		return
	}
	defer c.Close()

	eg, gctx := errgroup.WithContext(ctx)

	if *flagContainer || *flagAll {
		eg.Go(func() error {
			return runContainerPipeline(c, gctx)
		})
	}

	if *flagSetup || *flagAll {
		eg.Go(func() error {
			return runSetupPipeline(c, gctx)
		})
	}

	if *flagRelease || *flagAll {
		parentDir := getParentDir()
		repoDir := c.Host().Directory(parentDir, dagger.HostDirectoryOpts{Include: []string{".git", ".releaserc.json"}})
		eg.Go(func() error {
			return pipeline.RunSemanticRelease(repoDir, "github", c, gctx)
		})
	}

	return eg.Wait()
}

func runSetupPipeline(c *dagger.Client, ctx context.Context) (err error) {
	eg, gctx := errgroup.WithContext(ctx)
	cAlpine := c.Container().From("alpine:latest").WithExec([]string{"apk", "add", "curl", "file", "unzip"})
	cUbuntu := c.Container().From("ubuntu:latest").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "curl", "unzip"})

	pythonVersions := getLinesFromFile("test/python-versions")
	for _, version := range pythonVersions {
		version := version
		eg.Go(func() error {
			cPy := pipeline.WithPythonFromSource(version, "alpine", cAlpine, c, gctx)
			_, err := cPy.WithExec([]string{"python", "--version"}).Stderr(ctx)
			return err
		})
	}

	for _, version := range pythonVersions {
		version := version
		eg.Go(func() error {
			cPy := pipeline.WithPythonFromSource(version, "ubuntu", cUbuntu, c, gctx)
			_, err := cPy.WithExec([]string{"python", "--version"}).Stderr(ctx)
			return err
		})
	}

	eg.Go(func() error {
		cTf := pipeline.WithTerraform("1.6.0", cUbuntu, c, gctx)
		_, err := cTf.WithExec([]string{"terraform", "version"}).Stderr(ctx)
		return err
	})

	eg.Go(func() error {
		cTf := pipeline.WithTerraform("1.6.0", cAlpine, c, gctx)
		_, err := cTf.WithExec([]string{"terraform", "version"}).Stderr(ctx)
		return err
	})

	// Not sure why we get 404... Rate-limiting?
	// goVersions := getLinesFromFile("test/go-versions")
	// for _, version := range goVersions {
	// 	version := version
	// 	eg.Go(func() error {
	// 		cGo := pipeline.WithGo(cAlpine, version, c, gctx)
	// 		_, err := cGo.WithExec([]string{"go", "version"}).Stderr(ctx)
	// 		return err
	// 	})
	// }

	return eg.Wait()
}

func runContainerPipeline(c *dagger.Client, ctx context.Context) (err error) {
	eg, gctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return pipeline.BuildK8SUtils(c, gctx)
	})

	return eg.Wait()
}

func getLinesFromFile(p string) []string {
	f, err := os.ReadFile("test/python-versions")
	if err != nil {
		log.Fatalf("ERROR: cannot get lines from file: %s", err)
	}
	fLines := strings.Split(string(f), "\n")
	// Truncate trailing newline
	if fLines[len(fLines)-1] == "\n" {
		fLines = fLines[:len(fLines)-1]
	}
	return fLines
}
