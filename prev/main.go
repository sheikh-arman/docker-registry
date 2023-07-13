package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	_, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	listContainers(cli)
	//Build(ctx, cli)
	time.Sleep(time.Second * 3)
}

func listContainers(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("as\n\n\n\n")
	fmt.Println(containers, "\n\n")
	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID, container.Image)
	}
}
