package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/moby/term"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	DockerFileName = "Dockerfile"
	DockerFilePath = "~/arman/"
)

func initBuild(app *App, file *object.File) {
	repoUrl := app.GitRepo
	gitHub := "https://github.com/"
	repoUrl = repoUrl[len(gitHub) : len(repoUrl)-4]

	url := "https://raw.githubusercontent.com/"
	url += repoUrl
	url += "/"

	for _, b := range app.Blocks {

		tags := b.Tags
		var Tags []string
		for _, tag := range tags {
			imageName := file.Name
			imageName = imageName[8:]
			conTag := imageName
			conTag += ":"
			conTag += tag
			Tags = append(Tags, conTag)
		}

		urlBlock := url
		if len(b.Directory) == 0 {
			if len(b.GitCommit) == 0 {
				urlMain := urlBlock
				urlMain += "main"
				urlMain += "/Dockerfile"
				BuildImage(urlMain, Tags)

				urlMaster := urlBlock
				urlMaster += "master"
				urlMaster += "/Dockerfile"
				BuildImage(urlMaster, Tags)
			} else {
				urlCommit := urlBlock
				urlCommit += b.GitCommit
				urlCommit += "/Dockerfile"
				BuildImage(urlCommit, Tags)
			}
		} else {
			if len(b.GitCommit) == 0 {
				urlMain := urlBlock
				urlMain += "main/"
				urlMain += b.Directory
				urlMain += "/Dockerfile"
				BuildImage(urlMain, Tags)

				urlMaster := urlBlock
				urlMaster += "master/"
				urlMaster += b.Directory
				urlMaster += "/Dockerfile"
				BuildImage(urlMaster, Tags)
			} else {
				urlCommit := urlBlock
				urlCommit += b.GitCommit
				urlCommit += "/"
				urlCommit += b.Directory
				urlCommit += "/Dockerfile"
				BuildImage(urlCommit, Tags)
			}
		}
		fmt.Println(Tags, " Done")
		/// End Docker Image Build
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
	Build(ctx, cli, DockerFileURL, tag)
	time.Sleep(time.Second * 3)
}

func Build(ctx context.Context, cli *client.Client, DockerFileURL string, tag []string) {

	err := downloadDocker(DockerFileURL)
	if err != nil {
		return
	}
	buildOpts := types.ImageBuildOptions{
		Dockerfile: DockerFileName,
		Tags:       tag,
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
	//fmt.Println(resp, " arman ", termFd, " ", isTerm)
	jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
}

func downloadDocker(fileURL string) error {
	//fileURL := "https://raw.githubusercontent.com/TimWolla/docker-adminer/c9c54b18f79a66409a3153a94f629ea68f08647c/4/Dockerfile"
	localFilePath := DockerFilePath
	fmt.Println(fileURL)
	err := downloadFile(fileURL, localFilePath)
	if err != nil {
		fmt.Println("Error downloading file")
		return err
	}
	fmt.Println("File Downloaded successfully")

	fmt.Println(fileURL, localFilePath)
	return nil
}

func downloadFile(url, filePath string) error {
	filePath += "Dockerfile"
	outFile, err := os.Create(filePath)
	if err != nil {
		//fmt.Println(err, "culprit ? ")
		return err
	}
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", response.StatusCode)
	}
	_, err = io.Copy(outFile, response.Body)
	if err != nil {
		return err
	}
	return nil
}
