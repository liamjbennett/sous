package build

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
	"github.com/opentable/sous/tools/dir"
	"github.com/opentable/sous/tools/docker"
	"github.com/opentable/sous/tools/file"
	"github.com/opentable/sous/tools/git"
	"github.com/opentable/sous/tools/path"
)

type Context struct {
	Git                  *git.Info
	Action               string
	DockerRegistry       string
	Host, FullHost, User string
	BuildState           *BuildState
	AppVersion           string
}

func (bc *Context) IsCI() bool {
	return bc.User == "ci"
}

func GetContext(action string) *Context {
	c := config.Load()
	registry := c["docker-registry"]
	if registry == "" {
		cli.Fatalf("Missing config: Please set your docker registry using `sous config docker-registry <registry>`")
	}
	gitInfo := git.GetInfo()
	bs := GetBuildState(action, gitInfo)
	return &Context{
		Git:            gitInfo,
		Action:         action,
		DockerRegistry: registry,
		Host:           cmd.Stdout("hostname"),
		FullHost:       cmd.Stdout("hostname", "-f"),
		User:           getUser(),
		BuildState:     bs,
	}
}

func (c *Context) DockerTag() string {
	return c.DockerTagForBuildNumber(c.BuildNumber())
}

func (c *Context) BuildNumber() int {
	return c.BuildState.CurrentCommit().BuildNumber
}

func (c *Context) PrevDockerTag() string {
	return c.DockerTagForBuildNumber(c.BuildNumber() - 1)
}

func (c *Context) DockerTagForBuildNumber(n int) string {
	if c.AppVersion == "" {
		cli.Fatalf("AppVersion not set")
	}
	name := c.CanonicalPackageName()
	if c.Action != "build" {
		name += "_" + c.Action
	}
	repo := fmt.Sprintf("%s/%s", c.User, name)
	buildNumber := strconv.Itoa(n)
	if c.User != "teamcity" {
		buildNumber = c.Host + "-" + buildNumber
	}
	tag := fmt.Sprintf("v%s-%s-%s",
		c.AppVersion, c.Git.CommitSHA[0:8], buildNumber)
	// e.g. on local dev machine:
	//   some.registry.com/username/widget-factory:v0.12.1-912eeeab-host-1
	return fmt.Sprintf("%s/%s:%s", c.DockerRegistry, repo, tag)
}

func (c *Context) NeedsBuild() bool {
	if !c.LastBuildImageExists() {
		return true
	}
	if c.BuildState.CommitSHA != c.BuildState.LastCommitSHA {
		return true
	}
	cc := c.BuildState.CurrentCommit()
	if cc.Hash != cc.OldHash {
		return true
	}
	return false
}

func (c *Context) LastBuildImageExists() bool {
	return docker.ImageExists(c.PrevDockerTag())
}

func (s *BuildState) CurrentCommit() *Commit {
	return s.Commits[s.CommitSHA]
}

func (bc *Context) Commit() {
	if bc.IsCI() {
		return
	}
	bc.BuildState.Commit()
}

func (bc *Context) CanonicalPackageName() string {
	c := bc.Git.CanonicalName()
	p := strings.Split(c, "/")
	return p[len(p)-1]
}

func buildingInCI() bool {
	return os.Getenv("TEAMCITY_VERSION") != ""
}

func getUser() string {
	if buildingInCI() {
		return "ci"
	}
	return cmd.Stdout("id", "-un")
}

func getStateFile(action string, g *git.Info) string {
	dirPath := fmt.Sprintf("~/.ot/sous/builds/%s/%s", g.CanonicalName(), action)
	dir.EnsureExists(dirPath)
	return fmt.Sprintf("%s/state", dirPath)
}

func GetBuildState(action string, g *git.Info) *BuildState {
	filePath := getStateFile(action, g)
	var state *BuildState
	if !file.ReadJSON(&state, filePath) {
		state = &BuildState{
			Commits: map[string]*Commit{},
		}
	}
	if state == nil {
		cli.Fatalf("Nil state at %s", filePath)
	}
	if state.Commits == nil {
		cli.Fatalf("Nil commits at %s", filePath)
	}
	c, ok := state.Commits[g.CommitSHA]
	if !ok {
		state.Commits[g.CommitSHA] = &Commit{}
	}
	state.LastCommitSHA = state.CommitSHA
	state.CommitSHA = g.CommitSHA
	state.path = filePath
	c = state.Commits[g.CommitSHA]
	if buildingInCI() {
		bn, ok := tryGetBuildNumberFromEnv()
		if !ok {
			cli.Fatalf("unable to get build number from $BUILD_NUMBER TeamCity")
		}
		c.BuildNumber = bn
	}
	c.OldHash = c.Hash
	c.Hash = CalculateHash()
	return state
}

func (c *Context) IncrementBuildNumber() {
	if !buildingInCI() {
		c.BuildState.CurrentCommit().BuildNumber++
	}
}

func (s *BuildState) Commit() {
	file.WriteJSON(s, s.path)
}

func (c *Context) SaveFile(content, name string) {
	file.WriteString(content, c.FilePath(name))
}

func (c *Context) FilePath(name string) string {
	return c.BaseDir() + "/" + name
}

func (c *Context) BaseDir() string {
	return path.BaseDir(c.BuildState.path)
}

func CalculateHash() string {
	h := sha1.New()
	toolVersion := cmd.Stdout("sous", "version")
	io.WriteString(h, toolVersion)
	indexDiffs := cmd.Stdout("git", "diff-index", "HEAD")
	if len(indexDiffs) != 0 {
		io.WriteString(h, indexDiffs)
	}
	newFiles := git.UntrackedUnignoredFiles()
	if len(newFiles) != 0 {
		for _, f := range newFiles {
			io.WriteString(h, f)
			if content, ok := file.ReadString(f); ok {
				io.WriteString(h, content)
			}
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

type BuildState struct {
	CommitSHA, LastCommitSHA string
	Commits                  map[string]*Commit
	path                     string
}

type Commit struct {
	Hash, OldHash string
	BuildNumber   int
	ToolVersion   string
}

func tryGetBuildNumberFromEnv() (int, bool) {
	envBN := os.Getenv("BUILD_NUMBER")
	if envBN != "" {
		n, err := strconv.Atoi(envBN)
		if err != nil {
			cli.Fatalf("Unable to parse $BUILD_NUMBER (%s) to int: %s", envBN, err)
		}
		return n, true
	}
	return 0, false
}
