package commands

import (
	"github.com/opentable/sous/core"
	"github.com/opentable/sous/tools/cli"
)

func ImageHelp() string {
	return `sous image prints the last built image tag for this project`
}

func Image(sous *core.Sous, args []string) {
	target := "app"
	if len(args) != 0 {
		target = args[0]
	}
	_, context := sous.AssembleTargetContext(target)
	if context.BuildNumber() == 0 {
		cli.Fatalf("no builds yet")
	}
	cli.Outf(context.DockerTag())
	cli.Success()
}
