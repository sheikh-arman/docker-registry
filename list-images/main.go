package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"time"

	"github.com/docker/docker/client"
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
	listContainers(cli)
	listImages(cli)
	//Build(ctx, cli)
	time.Sleep(time.Second * 3)
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
