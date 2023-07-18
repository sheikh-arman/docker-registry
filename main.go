package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/go-git/go-git/v5"
	. "github.com/go-git/go-git/v5/_examples"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/moby/term"
	"github.com/pkg/errors"
	_ "github.com/sheikh-arman/docker-registry/buildimage"
	"gomodules.xyz/semvers"
	"gomodules.xyz/sets"
	"io"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

// Read from Git directly
func main() {

	apps := map[string]AppHistory{}
	//outDir := "./library"

	err := ProcessGitRepo(apps, true)
	CheckIfError(err)
	//buildImage("https://raw.githubusercontent.com/aerospike/aerospike-server.docker/fe338f8af95f2b20e4a6db3aa2d29a6d3373a2ff/enterprise/debian11/Dockerfile", "aerospike:ll")
	//err = PrintUnifiedHistory(outDir, apps)
	/*if err != nil {
		panic(err)
	}*/
}

func ProcessGitRepo(apps map[string]AppHistory, fullHistory bool) error {
	repoURL := "https://github.com/docker-library/official-images"

	// Clones the given repository, creating the remote, the local branches
	// and fetching the objects, everything in memory:
	Info("git clone " + repoURL)
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return err
	}

	// Gets the HEAD history from HEAD, just like this command:
	Info("git log")

	// ... retrieves the branch pointed by HEAD
	ref, err := r.Head()
	if err != nil {
		return err
	}

	// ... retrieves the commit history
	opts := git.LogOptions{From: ref.Hash()}
	if !fullHistory {
		from := time.Now().UTC()
		to := from.Add(-14 * 24 * time.Hour)
		opts.Since = &to
		opts.Until = &from
	}
	cIter, err := r.Log(&opts)
	if err != nil {
		return err
	}

	return cIter.ForEach(ProcessCommit(apps))
}

func ProcessCommit(apps map[string]AppHistory) func(c *object.Commit) error {
	return func(c *object.Commit) error {
		files, err := c.Files()
		if err != nil {
			return err
		}
		return files.ForEach(func(file *object.File) error {
			if !strings.HasPrefix(file.Name, "library/") {
				return nil
			}

			lines, err := file.Lines()
			if err != nil {
				return err
			}
			app, err := ParseLibraryFileContent(filepath.Base(file.Name), lines)
			if err != nil || app == nil {
				return err
			}

			//Start Docker Image Build
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
					//fmt.Println(b)
					//CheckIfError(err)
				}

				urlBlock := url
				if len(b.Directory) == 0 {
					if len(b.GitCommit) == 0 {
						urlBlock1 := urlBlock
						urlBlock2 := urlBlock
						urlBlock1 += "main"
						urlBlock2 += "master"
						urlBlock1 += "/Dockerfile"
						urlBlock2 += "/Dockerfile"
						buildImage(urlBlock1, Tags)
						buildImage(urlBlock2, Tags)
					} else {
						urlBlock1 := urlBlock
						urlBlock1 += b.GitCommit
						urlBlock1 += "/Dockerfile"
						buildImage(urlBlock1, Tags)
					}
				} else {
					if len(b.GitCommit) == 0 {
						urlBlock1 := urlBlock
						urlBlock2 := urlBlock
						urlBlock1 += "main/"
						urlBlock2 += "master/"
						urlBlock1 += b.Directory
						urlBlock2 += b.Directory
						urlBlock1 += "/Dockerfile"
						urlBlock2 += "/Dockerfile"
						buildImage(urlBlock1, Tags)
						buildImage(urlBlock2, Tags)
					} else {
						urlBlock1 := urlBlock
						urlBlock1 += b.GitCommit
						urlBlock1 += "/"
						urlBlock1 += b.Directory
						urlBlock1 += "/Dockerfile"
						buildImage(urlBlock1, Tags)
					}
				}
				fmt.Println(Tags, " Done")
				/// End Docker Image Build

				/*var dockerUrl string

				dockerUrl = app.GitRepo
				fmt.Println(app)

				buildImage("sdgfg", "asjhdu")
				fmt.Println(b)
				CheckIfError(err)*/
			}
			/*	fmt.Println(app.GitRepo, "\n\n")
				fmt.Println(repoUrl, "\n\n")
				fmt.Println(app.Blocks, "\n\n")
				klog.InfoS("processed", "commit", c.ID(), "file", file.Name, "blocks", len(app.Blocks))*/

			/*h, found := apps[app.Name]
			if !found {
				h = AppHistory{
					Name:      app.Name,
					GitRepo:   app.GitRepo,
					KnownTags: sets.NewString(),
					Blocks:    nil,
				}
			}
			GatherHistory(&h, app)
			apps[app.Name] = h*/

			return nil
		})
	}
}

func main_local() {
	apps := map[string]AppHistory{}
	dir := "./official-images/library"
	outDir := "./library"

	err := ProcessRepo(apps, dir)
	if err != nil {
		panic(err)
	}
	err = PrintUnifiedHistory(outDir, apps)
	if err != nil {
		panic(err)
	}

	//entries, err := os.ReadDir(dir)
	//if err != nil {
	//	panic(err)
	//}
	//for _, entry := range entries {
	//	if entry.IsDir() {
	//		continue
	//	}
	//
	//	filename := filepath.Join(dir, entry.Name())
	//	if app, err := ParseLibraryFile(filename); err != nil {
	//		panic(err)
	//	} else {
	//		klog.InfoS("processed", "file", filename, "blocks", len(app.Blocks))
	//	}
	//}

	//// official-images/library/postgres
	//if app, err := ParseLibraryFile("./official-images/library/sl"); err != nil {
	//	panic(err)
	//} else {
	//	fmt.Printf("%+v\n", app)
	//}
}

func PrintUnifiedHistory(outDir string, apps map[string]AppHistory) error {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	for appName, h := range apps {
		dir := filepath.Join(outDir, appName)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		buf.Reset()
		buf.WriteString("GitRepo: ")
		buf.WriteString(h.GitRepo)
		buf.WriteRune('\n')

		for _, b := range h.Blocks {
			buf.WriteRune('\n')
			buf.WriteString(b.String())
		}

		filename := filepath.Join(dir, "app.txt")
		err = os.WriteFile(filename, buf.Bytes(), 0644)
		if err != nil {
			return errors.Wrap(err, "file: "+filename)
		}

		filename = filepath.Join(dir, "app.json")
		data, err := json.MarshalIndent(h, "", "  ")
		if err != nil {
			return errors.Wrap(err, "file: "+filename)
		}
		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			return errors.Wrap(err, "file: "+filename)
		}

		filename = filepath.Join(dir, "app.yaml")
		data, err = yaml.Marshal(h)
		if err != nil {
			return errors.Wrap(err, "file: "+filename)
		}
		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			return errors.Wrap(err, "file: "+filename)
		}

		filename = filepath.Join(dir, "tags.txt")
		tags := h.KnownTags.List()
		semvers.SortVersions(tags, func(vi, vj string) bool {
			return !semvers.Compare(vi, vj)
		})
		data = []byte(strings.Join(tags, "\n"))
		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			return errors.Wrap(err, "file: "+filename)
		}

		{
			tags := make([]string, 0, h.KnownTags.Len())
			for tag := range h.KnownTags {
				if _, err := semver.NewVersion(tag); err == nil {
					tags = append(tags, tag)
				}
			}
			semvers.SortVersions(tags, func(vi, vj string) bool {
				return !semvers.Compare(vi, vj)
			})
			filename = filepath.Join(dir, "semver.txt")
			data = []byte(strings.Join(tags, "\n"))
			err = os.WriteFile(filename, data, 0644)
			if err != nil {
				return errors.Wrap(err, "file: "+filename)
			}
		}
	}
	return nil
}

var acceptedPreReleases = sets.NewString(
	"",
	"bullseye",
	"bookworm",
	"alpine",
	"centos",
	"management-alpine", // rabbitmq
	"management",        // rabbitmq
	"slim",              // debian
	"jammy",             // ubuntu
	"focal",             // ubuntu
	"temurin",           // java
	"openjdk",           // java
)

func SupportedPreRelease(v *semver.Version) bool {
	_, found := acceptedPreReleases[v.Prerelease()]
	return found
}

func ProcessRepo(apps map[string]AppHistory, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		app, err := ParseLibraryFile(filename)

		if err != nil || app == nil {
			return err
		}
		klog.InfoS("processed", "file", filename, "blocks", len(app.Blocks))

		h, found := apps[app.Name]
		if !found {
			h = AppHistory{
				Name:      app.Name,
				GitRepo:   app.GitRepo,
				KnownTags: sets.NewString(),
				Blocks:    nil,
			}
		}
		GatherHistory(&h, app)
		apps[app.Name] = h
	}
	return nil
}

func GatherHistory(h *AppHistory, app *App) {
	for _, b := range app.Blocks {
		if nb := processBlock(h, &b); nb != nil {
			h.Blocks = append(h.Blocks, *nb)
		}
	}
}

func processBlock(h *AppHistory, b *Block) *Block {
	var result *Block

	newTags := make([]string, 0, len(b.Tags))
	for _, tag := range b.Tags {
		if !h.KnownTags.Has(tag) {
			newTags = append(newTags, tag)
		}
	}
	if len(newTags) > 0 {
		result = &Block{
			Tags:          newTags,
			Architectures: b.Architectures,
			GitCommit:     b.GitCommit,
			Directory:     b.Directory,
		}
		h.KnownTags.Insert(newTags...)
	}
	return result
}

type AppHistory struct {
	Name      string
	GitRepo   string
	KnownTags sets.String
	Blocks    []Block
}

type App struct {
	Name    string
	GitRepo string
	Blocks  []Block
}

type Block struct {
	Tags          []string
	Architectures []string
	GitCommit     string
	Directory     string
}

func (b Block) String() string {
	var buf bytes.Buffer
	if len(b.Tags) > 0 {
		buf.WriteString("Tags: ")
		buf.WriteString(strings.Join(b.Tags, ","))
		buf.WriteRune('\n')
	}
	if len(b.Architectures) > 0 {
		buf.WriteString("Architectures: ")
		buf.WriteString(strings.Join(b.Architectures, ","))
		buf.WriteRune('\n')
	}
	if len(b.GitCommit) > 0 {
		buf.WriteString("GitCommit: ")
		buf.WriteString(b.GitCommit)
		buf.WriteRune('\n')
	}
	if len(b.Directory) > 0 {
		buf.WriteString("Directory: ")
		buf.WriteString(b.Directory)
		buf.WriteRune('\n')
	}
	return buf.String()
}

func ParseLibraryFile(filename string) (*App, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseLibraryFileContent(filepath.Base(filename), strings.Split(string(data), "\n"))
}

func ParseLibraryFileContent(appName string, lines []string) (*App, error) {
	var app App

	var curBlock *Block
	var curProp string
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}

		if line == "" {
			if curBlock != nil {
				// process cur block
				app.Blocks = append(app.Blocks, *curBlock)
			}
			curBlock = nil
			curProp = ""
			continue
		}

		before, after, found := strings.Cut(line, ":")
		var parts []string
		if found {
			curProp = before
			parts = strings.Split(after, ",")
		} else {
			parts = strings.Split(before, ",")
		}
		parts = filter(parts)

		switch curProp {
		case "GitRepo":
			app.Name = appName
			app.GitRepo = parts[0]
		case "Tags":
			if curBlock == nil {
				curBlock = new(Block)
			}
			curBlock.Tags = append(curBlock.Tags, parts...)
		case "Architectures":
			if curBlock == nil {
				curBlock = new(Block)
			}
			curBlock.Architectures = append(curBlock.Architectures, parts...)
		case "GitCommit":
			if curBlock == nil {
				curBlock = new(Block)
			}
			curBlock.GitCommit = parts[0]
		case "Directory":
			if curBlock == nil {
				curBlock = new(Block)
			}
			curBlock.Directory = parts[0]
		default:
			klog.V(5).InfoS("ignoring property", before, after)
		}
	}

	// last block
	if curBlock != nil {
		// process cur block
		app.Blocks = append(app.Blocks, *curBlock)
	}

	// eg: ./official-images/library/sourcemage
	if app.Name == "" {
		return nil, nil
	}
	return &app, nil
}

func filter(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

const (
	DockerFileName = "Dockerfile"
	DockerFilePath = "/home/user/go/src/github.com/sheikh-arman/docker-registry/buildimage/"
)

func buildImage(DockerFileURL string, tag []string) {
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
