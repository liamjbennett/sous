package main

import (
	"github.com/opentable/sous/build"
	"github.com/opentable/sous/commands"
)

type SousCommand struct {
	Func      func(packs []*build.Pack, args []string)
	HelpFunc  func() string
	ShortDesc string
}

var Sous = struct {
	Commands map[string]SousCommand
}{}

func loadCommands() {
	Sous.Commands = map[string]SousCommand{
		"build": SousCommand{
			commands.Build, commands.BuildHelp,
			"build your project"},

		"push": SousCommand{
			commands.Push, commands.PushHelp,
			"push your project"},

		"run": SousCommand{
			commands.Run, commands.RunHelp,
			"run your project"},

		"dockerfile": SousCommand{
			commands.Dockerfile, commands.DockerfileHelp,
			"print current dockerfile"},

		"detect": SousCommand{
			commands.Detect, commands.DetectHelp,
			"detect available actions"},

		"test": SousCommand{
			commands.Test, commands.TestHelp,
			"test your project"},

		"build-path": SousCommand{
			commands.BuildPath, commands.BuildPathHelp,
			"build state directory"},

		"help": SousCommand{
			help, helphelp,
			"show this help"},

		"version": SousCommand{
			version, versionHelp,
			"show version info"},
	}
}