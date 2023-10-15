package pipeline

import (
	"context"
	"os"
	"strings"

	"dagger.io/dagger"
)

// Good resource: https://github.com/mesosphere/d2iq-daggers/blob/main/daggers/containers/customizer.go

func WithHostEnvVariablesMatchingPrefix(ctx context.Context, ctr *dagger.Container, prefix string, ignore ...string) *dagger.Container {
	// convert ignore list to a map for faster lookup
	ignoreMap := sliceToKeyMap(ignore)

	envAll := os.Environ()
	for _, env := range envAll {
		k := strings.Split(env, "=")[0]
		v := strings.Split(env, "=")[1]

		if strings.HasPrefix(k, prefix) || ignoreMap[env] {
			ctr = ctr.WithEnvVariable(k, v)
		}
	}

	return ctr
}
