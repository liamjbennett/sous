package golang

import (
	"fmt"

	"github.com/opentable/sous/core"
	"github.com/opentable/sous/tools/cmd"
	"github.com/opentable/sous/tools/dir"
	"github.com/opentable/sous/tools/docker"
)

type CompileTarget struct {
	*GoTarget
}

func NewCompileTarget(pack *Pack) *CompileTarget {
	return &CompileTarget{NewGoTarget("compile", pack)}
}

func (t *CompileTarget) DependsOn() []core.Target {
	return nil
}

func (t *CompileTarget) RunAfter() []string { return nil }

func (t *CompileTarget) Desc() string {
	return "The Go compile target generates a single binary file"
}

// Checking a Go project always passes.
func (t *CompileTarget) Check() error {
	return nil
}

func (t *CompileTarget) Dockerfile(c *core.Context) *docker.Dockerfile {
	df := &docker.Dockerfile{}
	df.From = t.pack.baseImageTag("compile")
	uid := cmd.Stdout("id", "-u")
	gid := cmd.Stdout("id", "-g")
	username := cmd.Stdout("whoami")
	// Just use the username for group name, it doesn't matter as long as
	// the IDs are right.
	df.AddRun(fmt.Sprintf("groupadd -g %s %s", gid, username))
	// Explanation of some of the below useradd flags:
	//   -M means do not create home directory, which we do not need
	//   --no-log-init means do not create a 32G sparse file (which Docker commit
	//       cannot handle properly, and tries to create a non-sparse 32G file.)
	df.AddRun(fmt.Sprintf("useradd --no-log-init -M --uid %s --gid %s %s", uid, gid, username))
	return df
}

func (t *CompileTarget) ContainerName(c *core.Context) string {
	return fmt.Sprintf("%s_reusable-builder", c.CanonicalPackageName())
}

func (t *CompileTarget) ContainerIsStale(c *core.Context) (bool, string) {
	return true, "it is not reusable"
}

func (t CompileTarget) DockerRun(c *core.Context) *docker.Run {
	containerName := t.ContainerName(c)
	run := docker.NewRun(c.DockerTag())
	run.Name = containerName
	run.AddEnv("ARTIFACT_NAME", t.artifactName(c))
	uid := cmd.Stdout("id", "-u")
	gid := cmd.Stdout("id", "-g")
	artifactOwner := fmt.Sprintf("%s:%s", uid, gid)
	run.AddEnv("ARTIFACT_OWNER", artifactOwner)
	artDir := t.artifactDir(c)
	dir.EnsureExists(artDir)
	run.AddVolume(artDir, "/artifacts")
	run.AddVolume(c.WorkDir, "/wd")
	binName := fmt.Sprintf("%s-%s", c.CanonicalPackageName(), c.AppVersion)
	run.Command = fmt.Sprintf("[ -d Godeps ] && godep go build -o %s || go build -o %s",
		binName, binName)
	return run
}

func (t *CompileTarget) artifactPath(c *core.Context) string {
	return fmt.Sprintf("%s/%s.tar.gz", t.artifactDir(c), t.artifactName(c))
}

func (t *CompileTarget) artifactDir(c *core.Context) string {
	return c.FilePath("artifacts")
}

func (t *CompileTarget) artifactName(c *core.Context) string {
	return fmt.Sprintf("%s-%s-%s-%d", c.CanonicalPackageName(), c.AppVersion, c.Git.CommitSHA, c.BuildNumber())
}