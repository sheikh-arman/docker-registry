package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"io"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DockerFileName = "Dockerfile"
	DockerFilePath = "docker/"
)

var (
	kubeDB = map[string]int{
		"elasticsearch": 1,
		"mariadb":       1,
		"memcached":     1,
		"mongo":         1,
		"mysql":         1,
		"percona":       1,
		"postgres":      1,
		"redis":         1,

		"kafka":     1,
		"xtradb":    1,
		"pgbouncer": 1,
		"proxysql":  1,
	}
)

func initBuild(app App, imagename string) {
	fmt.Println(app)

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
			imageName := imagename
			imageName = "ghcr.io/sheikh-arman/" + imageName[8:]
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

		/// End Docker Image Build
	}
}

func BuildImage(DockerFileURL string, tag []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	svr, err := cli.ServerVersion(context.TODO())
	if err != nil {
		panic(err)
	}
	klog.Infof("%s +++++++++++  %+v \n", cli.ClientVersion(), svr)
	Build(ctx, cli, DockerFileURL, tag)
	time.Sleep(time.Second * 3)
}

func Build(ctx context.Context, cli *client.Client, DockerFileURL string, tag []string) {

	for _, dockerTag := range tag {
		if !checkTag(dockerTag) {
			continue
		}
		err := downloadDocker(DockerFileURL, DockerFileName)
		if err != nil {
			return
		}
		Files, err := createAssociatedFile(DockerFileURL)
		Files = append(Files, DockerFileName)
		if err != nil {
			return
		}
		buildContext, err := os.Open(DockerFilePath) // Set the correct working directory here
		if err != nil {
			return
		}
		defer buildContext.Close()
		buildCtx, err := archive.TarWithOptions(DockerFilePath, &archive.TarOptions{
			IncludeFiles: Files,
		})
		if err != nil {
			fmt.Println("error on TarWithOptions func ", err)
		}
		buildOpts := types.ImageBuildOptions{
			//Context:    buildCtx,
			Dockerfile: DockerFileName,
			Tags:       []string{dockerTag},
			CacheFrom:  nil,
		}

		resp, err := cli.ImageBuild(ctx, buildCtx, buildOpts)
		if err != nil {
			log.Fatalf("build error huuu- %s", err)
		}

		termFd, isTerm := term.GetFdInfo(os.Stderr)

		//_, err = io.Copy(os.Stdout, resp.Body)
		jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)

		pushImage(cli, ctx, dockerTag)
		deleteImage(ctx, cli, dockerTag)

	}
}

func downloadDocker(url, fileName string) error {
	//fileURL := "https://raw.githubusercontent.com/TimWolla/docker-adminer/c9c54b18f79a66409a3153a94f629ea68f08647c/4/Dockerfile"
	localFilePath := DockerFilePath
	localFilePath += fileName
	err := downloadFile(url, localFilePath)
	if err != nil {
		fmt.Println("Error downloading file")
		return err
	}
	fmt.Println(url, localFilePath)
	fmt.Println("File Downloaded successfully")

	return nil
}

func downloadFile(url, filePath string) error {
	//filePath += "Dockerfile"
	outFile, err := os.Create(filePath)
	if err != nil {
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
func createAssociatedFile(DockerFileUrl string) ([]string, error) {
	localFilePath := DockerFilePath
	localFilePath += "Dockerfile"

	openFile, err := os.Open(localFilePath)
	if err != nil {
		return []string{}, err
	}
	scanner := bufio.NewScanner(openFile)
	var listFiles []string
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "COPY") || strings.HasPrefix(line, "ADD") {
			//fmt.Println(line, " #check")
			words := strings.Split(line, " ")
			listFiles = append(listFiles, words[1])
			//fmt.Println(words[0], " tut ", words[1])
		}
	}
	for _, listFile := range listFiles {
		err = downloadDocker(DockerFileUrl, listFile)
		if err != nil {
			return []string{}, err
		}
	}
	return listFiles, nil
}
func pushImage(cli *client.Client, ctx context.Context, imageName string) {

	//imageName := "ghcr.io/sheikh-arman/hello:lastest"
	username := "sheikh-arman"
	//token := "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"
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
	fmt.Println("Pushing Image...")
	_, err = io.Copy(os.Stdout, resp)

	if err != nil {
		panic(err)
	}
	fmt.Println("Image Push complete")
}

func deleteImage(ctx context.Context, cli *client.Client, imageName string) {
	resp, err := cli.ImageRemove(ctx, imageName, types.ImageRemoveOptions{PruneChildren: true})
	if err != nil {
		fmt.Errorf("%v", err.Error())
		return
	}
	for _, re := range resp {
		fmt.Println(re)
	}
}

func checkTag(tag string) bool {

	return true
}
