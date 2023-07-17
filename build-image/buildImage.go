package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	DockerFileName = "Dockerfile"
	DockerFilePath = "/home/user/go/src/github.com/sheikh-arman/docker-registry/build-image/"
)

func buildImage(DockerFileURL string, tag string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)
	Build(ctx, cli, DockerFileURL, tag)
	time.Sleep(time.Second * 3)
}

func Build(ctx context.Context, cli *client.Client, DockerFileURL string, tag string) {

	downloadDocker(DockerFileURL)

	buildOpts := types.ImageBuildOptions{
		Dockerfile: DockerFileName,
		Tags:       []string{tag},
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
	//fmt.Println(resp, " arman ", termFd, " ", isTerm)
	jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
}

func downloadDocker(fileURL string) {
	//fileURL := "https://raw.githubusercontent.com/TimWolla/docker-adminer/c9c54b18f79a66409a3153a94f629ea68f08647c/4/Dockerfile"
	localFilePath := DockerFilePath

	err := downloadFile(fileURL, localFilePath)
	if err != nil {
		fmt.Println("Error downloading file")
		return
	}
	fmt.Println("File Downloaded successfully")

	fmt.Println(fileURL, localFilePath)

}

func downloadFile(url, filePath string) error {
	filePath += "Dockerfile"
	outFile, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err, "culprit ? ")
		return err
	}
	defer outFile.Close()
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", response.StatusCode)
	}
	_, err = io.Copy(outFile, response.Body)
	if err != nil {
		return err
	}
	return nil
}
