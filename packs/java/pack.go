package java

import (
	"fmt"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/core"
	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/file"
)

type Pack struct {
	Config *config.JavaConfig
	Project MavenProject
}

func New(c *config.JavaConfig) *Pack {
	return &Pack{Config: c}
}

func (p *Pack) Name() string {
	return "Java"
}

func (p *Pack) Desc() string {
	return "Java Build Pack"
}

func (p *Pack) Detect() error {
	if len(file.Find("pom.xml")) == 0 {
		return fmt.Errorf("No file matching pom.xml file found")
	}

	if !file.ReadXML(&p.Project, "pom.xml") {
		return fmt.Errorf("no pom.xml file found")
	}
	// TODO: if pom includes <modules> then Err out
	// TODO: if the pom does not contain a version then read the parent pom.xml to find it
	return nil
}

func (p *Pack) Problems() core.ErrorCollection {
	return core.ErrorCollection{}
}

func (p *Pack) AppVersion() string {
	return p.Project.Version
}

func (p *Pack) AppDesc() string {
	return "a Java project"
}

func (p *Pack) Targets() []core.Target {
	return []core.Target{
		NewCompileTarget(p),
		NewAppTarget(p),
		NewTestTarget(p),
	}
}

func (p *Pack) String() string {
	return p.Name()
}

func (p *Pack) baseImageTag(target string) string {
	baseImageTag, ok := p.Config.AvailableVersions.GetBaseImageTag(
		p.Config.DefaultJavaVersion, target)
	if !ok {
		cli.Fatalf("Java build pack misconfigured, default version %s not available for target %s",
			p.Config.DefaultJavaVersion, target)
	}
	return baseImageTag
}
