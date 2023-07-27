package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	DockerFileName   = "Dockerfile"
	DockerFilePath   = "docker/"
	RemoteRepository = "ghcr.io/sheikh-arman/"
)

var (
	kubeDB = map[string]int{
		"elasticsearch": 1,
		/*"mariadb":       1,
		"memcached":     1,
		"mongo":         1,
		"mysql":         1,
		"percona":       1,
		"postgres":      1,
		"redis":         1,

		"kafka":     1,
		"xtradb":    1,
		"pgbouncer": 1,
		"proxysql":  1,*/
	}
	currentTag = map[string]int{}
)

func init() {
	configRemoteRepo()
	currentTagList()
}
func currentTagList() {
	openFile, err := os.OpenFile("taglist.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(openFile)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		currentTag[line] = 1
	}
}
func initBuild(app *App, imageName string) {
	//fmt.Println(app)
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
			imageName := imageName
			imageName = RemoteRepository + imageName[8:]
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
				BuildImage(urlMain, Tags, imageName)

				urlMaster := urlBlock
				urlMaster += "master"
				urlMaster += "/Dockerfile"
				BuildImage(urlMaster, Tags, imageName)
			} else {
				urlCommit := urlBlock
				urlCommit += b.GitCommit
				urlCommit += "/Dockerfile"
				BuildImage(urlCommit, Tags, imageName)
			}
		} else {
			if len(b.GitCommit) == 0 {
				urlMain := urlBlock
				urlMain += "main/"
				urlMain += b.Directory
				urlMain += "/Dockerfile"
				BuildImage(urlMain, Tags, imageName)

				urlMaster := urlBlock
				urlMaster += "master/"
				urlMaster += b.Directory
				urlMaster += "/Dockerfile"
				BuildImage(urlMaster, Tags, imageName)
			} else {
				urlCommit := urlBlock
				urlCommit += b.GitCommit
				urlCommit += "/"
				urlCommit += b.Directory
				urlCommit += "/Dockerfile"
				BuildImage(urlCommit, Tags, imageName)
			}
		}
		/// End Docker Image Build
	}
}

func BuildImage(DockerFileURL string, tag []string, imageName string) {
	for _, dockerTag := range tag {
		fmt.Println(dockerTag, currentTag[dockerTag], " huh")
		if currentTag[dockerTag] == 1 {
			continue
		}
		fmt.Println("not Working")
		err := downloadDocker(DockerFileURL, DockerFileName)
		if err != nil {
			return
		}
		Files, err := createAssociatedFile(DockerFileURL)
		Files = append(Files, DockerFileName)
		if err != nil {
			return
		}
		args := []interface{}{
			"build",
			"-t",
			dockerTag,
			DockerFilePath,
		}
		data, err := sh.Command("docker", args...).Output()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(data))
		reportVul(dockerTag, imageName)
		pushImage(dockerTag)
		deleteImage(dockerTag)
		addTag(dockerTag)
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
			words := strings.Split(line, " ")
			listFiles = append(listFiles, words[1])
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
func reportVul(dockerTag, imageName string) {
	imageName = imageName[8:]
	repo := RemoteRepository
	fileName := "json/"
	if !sh.Test("dir", fileName) {
		sh.Command("mkdir", fileName).Run()
	}
	fileName += imageName + "/"
	if !sh.Test("dir", fileName) {
		sh.Command("mkdir", fileName).Run()
	}
	fileName += dockerTag[len(repo):]
	fileName += ".json"
	args := []interface{}{
		"image",
		"-f",
		"json",
		"-o",
		fileName,
		dockerTag,
	}
	data, err := sh.Command("trivy", args...).Output()

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))

	/*fmt.Println("Check file name", fileName)
	sh.Command("cat", fileName).Run()*/

}
func pushImage(imageName string) {
	args2 := []interface{}{
		"push",
		imageName,
	}
	data, err := sh.Command("docker", args2...).Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
}

func deleteImage(imageName string) {
	args2 := []interface{}{
		"rmi",
		imageName,
	}
	data, err := sh.Command("docker", args2...).Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
}

func configRemoteRepo() {
	args := []interface{}{
		"login",
		"--username",
		"sheikh-arman",
		"--password",
		"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",
		"ghcr.io",
	}
	data, err := sh.Command("docker", args...).Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
}

func addTag(tag string) {

	cmd := sh.Command("echo", tag)

	// Open the file with the "os" package and set the file to append mode (os.O_APPEND)
	file, err := os.OpenFile("taglist.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	cmd.Stdout = file
	if err := cmd.Run(); err != nil {
		fmt.Println("Error running command:", err)
		return
	}
	fmt.Println(tag, " Added Successfully")
	currentTag[tag] = 1
}
