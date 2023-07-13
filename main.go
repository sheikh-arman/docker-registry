package main

import (
	"context"
	"fmt"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
	"time"
)

func main() {
	_, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {

		panic(err)
	}
	listContainer(cli)
	time.Sleep(time.Second * 3)
}

func listContainer(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), dockertypes.ContainerListOptions(types.ContainerListOptions{}))
	if err != nil {
		fmt.Printf("check2")
		panic(err)
	}
	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}
}
