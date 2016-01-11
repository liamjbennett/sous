package core

import (
	"os"
	"strings"
	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/docker"
	"github.com/opentable/sous/tools/git"
	"github.com/opentable/sous/tools/version"
)

func CheckForProblems(pack Pack) (fatal bool) {
	// Now we know that the user was asking for something possible with the detected build pack,
	// let's make sure that build pack is properly compatible with this project
	issues := pack.Problems()
	warnings, errors := issues.Warnings(), issues.Errors()
	if len(warnings) != 0 {
		cli.LogBulletList("WARNING:", issues.Strings())
	}
	if len(errors) != 0 {
		cli.LogBulletList("ERROR:", errors.Strings())
		cli.Logf("ERROR: Your project cannot be built by Sous until the above errors are rectified")
		return true
	}
	return false
}

func (s *Sous) AssembleTargetContext(targetName string) (Target, *Context) {
	packs := s.Packs
	p := DetectProjectType(packs)
	if p == nil {
		cli.Fatalf("no buildable project detected")
	}
	pack := CompiledPack{Pack: p}
	target, ok := pack.GetTarget(targetName)
	if !ok {
		cli.Fatalf("The %s build pack does not support %s", pack, targetName)
	}
	if fatal := CheckForProblems(pack.Pack); fatal {
		cli.Fatal()
	}
	context := GetContext(targetName)
	err := target.Check()
	if err != nil {
		cli.Fatalf("unable to %s %s project: %s", targetName, pack, err)
	}
	// If the pack specifies a version, check it matches the tagged version
	packAppVersion := strings.Split(pack.AppVersion(), "+")[0]
	if packAppVersion != "" {
		pv := version.Version(packAppVersion)
		gv := version.Version(context.BuildVersion.MajorMinorPatch)
		if !pv.Version.LimitedEqual(gv.Version) {
			cli.Warn("using latest git tagged version %s; your code reports version %s, which is ignored", gv, pv)
		}
	}
	return target, context
}

func RequireDocker() {
	docker.RequireVersion(version.Range(">=1.8.2"))
	docker.RequireDaemon()
}

func RequireGit() {
	git.RequireVersion(version.Range(">=1.9.1"))
	git.RequireRepo()
}

func DivineTaskHost() string {
	taskHost := os.Getenv("TASK_HOST")
	if taskHost != "" {
		return taskHost
	}
	return docker.GetDockerHost()
}
