package main

import (
	"bytes"
	"fmt"
	"github.com/go-git/go-git/v5"
	. "github.com/go-git/go-git/v5/_examples"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sheikh-arman/docker-registry/docker"
	"gomodules.xyz/sets"
	"k8s.io/klog/v2"
	"path/filepath"
	"strings"
	"time"
)

// Read from Git directly
func main() {

	apps := map[string]AppHistory{}
	//outDir := "./library"

	err := ProcessGitRepo(apps, true)
	CheckIfError(err)
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
						docker.BuildImage(urlBlock1, Tags)
						docker.BuildImage(urlBlock2, Tags)
					} else {
						urlBlock1 := urlBlock
						urlBlock1 += b.GitCommit
						urlBlock1 += "/Dockerfile"
						docker.BuildImage(urlBlock1, Tags)
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
						docker.BuildImage(urlBlock1, Tags)
						docker.BuildImage(urlBlock2, Tags)
					} else {
						urlBlock1 := urlBlock
						urlBlock1 += b.GitCommit
						urlBlock1 += "/"
						urlBlock1 += b.Directory
						urlBlock1 += "/Dockerfile"
						docker.BuildImage(urlBlock1, Tags)
					}
				}
				fmt.Println(Tags, " Done")
				/// End Docker Image Build
			}
			return nil
		})
	}
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
