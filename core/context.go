package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/core/resources"
	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
	"github.com/opentable/sous/tools/docker"
	"github.com/opentable/sous/tools/file"
	"github.com/opentable/sous/tools/git"
	"github.com/opentable/sous/tools/path"
	"github.com/opentable/sous/tools/version"
)

type Context struct {
	Git                  *git.Info
	WorkDir              string
	TargetName           string
	DockerRegistry       string
	Host, FullHost, User string
	BuildState           *BuildState
	BuildVersion         *BuildVersion
	PackInfo             interface{}
	changes              *Changes
}

func (bc *Context) IsCI() bool {
	return bc.User == "ci"
}

func GetContext(action string) *Context {
	var c = config.Load()
	registry := c.DockerRegistry
	gitInfo := git.GetInfo()
	bs := GetBuildState(action, gitInfo)
	wd, err := os.Getwd()
	if err != nil {
		cli.Fatalf("Unable to get current working directory: %s", err)
	}
	return &Context{
		Git:            gitInfo,
		WorkDir:        wd,
		TargetName:     action,
		DockerRegistry: registry,
		Host:           cmd.Stdout("hostname"),
		FullHost:       cmd.Stdout("hostname", "-f"),
		User:           getUser(),
		BuildState:     bs,
		BuildVersion:   buildVersion(gitInfo),
	}
}

// BuildVersion represents the semver string for the current build.
// The idea is to distinguish builds of exact tagged versions vs
// builds in between tags, by appending +revision to those in-between
// builds.
type BuildVersion struct {
	MajorMinorPatch, PlusRevision string
}

// String returns a semver-compatible string representing this build version.
func (bv *BuildVersion) String() string {
	if bv.PlusRevision == "" {
		return bv.MajorMinorPatch
	}
	return fmt.Sprintf("%s+%s", bv.MajorMinorPatch, bv.PlusRevision[:8])
}

func defaultBuildVersion(revision string) *BuildVersion {
	return &BuildVersion{
		MajorMinorPatch: "0.0.0",
		PlusRevision:    revision,
	}
}

// buildVersion constructs a BuildVersion from git info.
func buildVersion(i *git.Info) *BuildVersion {
	// Try to parse the nearest tag as a version. If it isn't a valid version,
	// we just give up and return  a default for now.
	// TODO: It's possible to walk through the tags in order of distance from
	// the current commit, to find the nearest semver tag, so consider doing
	// that, if this becomes an issue.
	if i.NearestTag == "" {
		cli.Warn("No git tags found in the format X.Y.Z, defaulting to v0.0.0", i.NearestTag)
		return defaultBuildVersion(i.CommitSHA)
	}
	v, err := version.NewVersion(i.NearestTag)
	if err != nil {
		cli.Warn("Latest git tag '%s' not in the format X.Y.Z, defaulting to v0.0.0", i.NearestTag)
		return defaultBuildVersion(i.CommitSHA)
	}
	if i.NearestTagSHA == i.CommitSHA {
		// We're building an exact version
		return &BuildVersion{MajorMinorPatch: v.String()}
	}
	// We're building a commit between named versions, so add the commit SHA
	return &BuildVersion{MajorMinorPatch: v.String(), PlusRevision: i.CommitSHA}
}

// DockerTag returns the docker tag used for the current build.
func (c *Context) DockerTag() string {
	return c.DockerTagForBuildNumber(c.BuildNumber())
}

// BuildNumber returns the build number for the current project at its
// present commit on this machine with this user login. Heh, a mouthful.
func (c *Context) BuildNumber() int {
	return c.BuildState.CurrentCommit().BuildNumber
}

// PrevDockerTag returns the previously built docker tag for this project.
// This is useful for re-using builds when appropriate.
func (c *Context) PrevDockerTag() string {
	return c.DockerTagForBuildNumber(c.BuildNumber() - 1)
}

// DockerTag for build number returns a full docker image name including
// registry, repository, and tag, for the current project at the specified
// build number.
func (c *Context) DockerTagForBuildNumber(n int) string {
	name := c.CanonicalPackageName()
	// Special case: for primary target "app" we don't
	// append the target name.
	if c.TargetName != "app" {
		name += "_" + c.TargetName
	}
	repo := fmt.Sprintf("%s/%s", c.User, name)
	buildNumber := strconv.Itoa(n)
	if c.User != "teamcity" {
		buildNumber = c.Host + "-" + buildNumber
	}
	tag := fmt.Sprintf("v%s-%s", c.BuildVersion, buildNumber)
	// Docker tags do not yet support semver, so replace + with _.
	// See https://github.com/docker/distribution/issues/1201
	// and https://github.com/docker/distribution/pull/1202
	tag = strings.Replace(tag, "+", "_", -1)
	// e.g. on local dev machine:
	//   some.registry.com/username/widget-factory:v0.12.1_912eeeab-host-1
	return fmt.Sprintf("%s/%s:%s", c.DockerRegistry, repo, tag)
}

func (c *Context) ChangesSinceLastBuild() *Changes {
	cc := c.BuildState.CurrentCommit()
	if c.changes == nil {
		c.changes = &Changes{
			NoBuiltImage:       !c.LastBuildImageExists(),
			NewCommit:          c.BuildState.CommitSHA != c.BuildState.LastCommitSHA,
			WorkingTreeChanged: cc.TreeHash != cc.OldTreeHash,
			SousUpdated:        cc.SousHash != cc.OldSousHash,
		}
	}
	return c.changes
}

// Changes is a set of flags indicating what's changed since the last time
// this project was built.
type Changes struct {
	NoBuiltImage, NewCommit, WorkingTreeChanged, SousUpdated bool
}

// Any returns true if there are any changes at all since the last build.
func (c *Changes) Any() bool {
	return c.NoBuiltImage || c.NewCommit || c.WorkingTreeChanged || c.SousUpdated
}

// LastBuildImageExists checks that the previously build image, if any, still
// exists on this machine. If there is no previously built image, or it's been
// deleted, return false, otherwise true.
func (c *Context) LastBuildImageExists() bool {
	return docker.ImageExists(c.PrevDockerTag())
}

// CurrentCommit returns the data for the current commit at HEAD in the repo.
func (s *BuildState) CurrentCommit() *Commit {
	return s.Commits[s.CommitSHA]
}

// Commit should be called after a build is successful, to permanently increment
// the build number for this commit.
func (bc *Context) Commit() {
	bc.BuildState.Commit()
}

// CanonicalPackageName returns the last path component of the canonical git
// repo name, which is used as the name of the application.
func (bc *Context) CanonicalPackageName() string {
	c := bc.Git.CanonicalRepoName()
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

func (c *Context) IncrementBuildNumber() {
	if !buildingInCI() {
		c.BuildState.CurrentCommit().BuildNumber++
	}
}

func (s *BuildState) Commit() {
	if s.path == "" {
		panic("BuildState.path is empty")
	}
	file.WriteJSON(s, s.path)
}

func (c *Context) SaveFile(content, name string) {
	filePath := c.FilePath(name)
	if filePath == "" {
		panic("Context file path was empty")
	}
	file.WriteString(content, filePath)
}

func (c *Context) TemporaryLinkResource(name string) {
	fileContents, ok := resources.Files[name]
	if !ok {
		cli.Fatalf("Cannot find resource %s, ensure go generate succeeded", name)
	}
	c.SaveFile(fileContents, name)
	file.TemporaryLink(c.FilePath(name), name)
}

// FilePath returns a path to a named file within the state directory
// of the current build target. This is used for things like passing
// artifacts from one build step to the next.
func (c *Context) FilePath(name string) string {
	return path.Resolve(c.BaseDir() + "/" + name)
}

// BaseDir return the build state base directory for the current target.
func (c *Context) BaseDir() string {
	return path.BaseDir(c.BuildState.path)
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
