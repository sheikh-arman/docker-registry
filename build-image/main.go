package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	DockerFileName = "dockerfile"
	DockerFilePath = "/home/user/go/src/github.com/sheikh-arman/docker-registry/build-image/"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)
	Build(ctx, cli)
	time.Sleep(time.Second * 3)
}

func Build(ctx context.Context, cli *client.Client) {
	buildOpts := types.ImageBuildOptions{
		Dockerfile: DockerFileName,
		Tags:       []string{"skaliarman/docker-registry:blue", "skaliarman/docker-registry2:blue"},
		CacheFrom:  nil,
	}

	buildCtx, err := archive.TarWithOptions(DockerFilePath, &archive.TarOptions{
		IncludeFiles: []string{
			DockerFileName,
		},
	})
	if err != nil {
		fmt.Println("error on TarWithOptions func ", err)
	}

	resp, err := cli.ImageBuild(ctx, buildCtx, buildOpts)
	if err != nil {
		log.Fatalf("build error huuu- %s", err)
	}
	defer resp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	fmt.Println(resp, " arman ", termFd, " ", isTerm)
	jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
}
