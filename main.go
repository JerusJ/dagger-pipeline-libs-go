package main

import (
	"context"
	"flag"
	"log"
	"os"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

var (
	flagAll       = flag.Bool("all", false, "")
	flagContainer = flag.Bool("containers", false, "")
	flagRelease   = flag.Bool("release", false, "")
)

func main() {
	if err := runPipelines(context.Background()); err != nil {
		log.Fatal(err)
	}
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

	if *flagRelease || *flagAll {
		repoDir := c.Host().Directory(".", dagger.HostDirectoryOpts{Include: []string{".git", ".releaserc.json"}})
		eg.Go(func() error {
			return RunSemanticRelease(repoDir, "github", c, gctx)
		})
	}

	return eg.Wait()
}

func runContainerPipeline(c *dagger.Client, ctx context.Context) (err error) {
	eg, gctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return BuildK8SUtils(c, gctx)
	})

	return eg.Wait()
}
