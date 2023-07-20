package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/client"
)

const (
	DockerFileName = "Dockerfile"
	DockerFilePath = "docker/"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	//fmt.Println(cli.ClientVersion())
	cli.NegotiateAPIVersion(ctx)
	//fmt.Println(cli.ClientVersion())
	//listContainers(cli)
	//listImages(cli)
	pushImage(cli, ctx)
	//cli.ImagePush(ctx, "231", types.ImagePushOptions{})
	//Build(ctx, cli)
	time.Sleep(time.Second * 3)
}

func pushImage(cli *client.Client, ctx context.Context) {

	imageName := "ghcr.io/sheikh-arman/hello:lastest"
	username := "sheikh-arman"
	token := "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"
	//password := "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"

	authConfig := types.AuthConfig{
		Username:      username,
		Password:      token,
		ServerAddress: "ghcr.io",
	}

	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}

	authStr := base64.URLEncoding.EncodeToString(authJSON)

	// Combine username and password with a colon separator and convert to base64
	//authStr := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	options := types.ImagePushOptions{
		RegistryAuth: authStr,
	}

	resp, err := cli.ImagePush(ctx, imageName, options)
	if err != nil {
		fmt.Println("culprit")
		panic(err)
	}

	defer resp.Close()
	fmt.Println("pushing Image...")
	_, err = io.Copy(os.Stdout, resp)

	if err != nil {
		panic(err)
	}
	fmt.Println("Image Push complete")
}

func listContainers(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println("\n")
	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID, container.Image)
	}
}

func listImages(cli *client.Client) {
	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Images\n\n")
	for _, image := range images {
		//fmt.Println(image)
		fmt.Printf("\n%s %s\n", image.RepoTags, image.RepoDigests)
	}
}

func BuildImage(DockerFileURL string, tag []string) {
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
		Tags:       []string{"skaliarman"},
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

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	//fmt.Println(resp, " docker ", termFd, " ", isTerm)
	jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
}
