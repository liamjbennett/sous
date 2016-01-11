package core

import "github.com/opentable/sous/tools/cli"

// DetectProjectType invokes Detect() for each registered pack.
//
// If a single pack is found to match, it returns that pack along with
// the object returned from its detect func. This object is subsequently
// passed into the detect step for each target supported by the pack.
func DetectProjectType(packs []Pack) Pack {
	var err error
	var pack Pack
	for _, p := range packs {
		if err = p.Detect(); err != nil {
			cli.Logf(err.Error())
			continue
		}
		if pack != nil {
			cli.Fatalf("multiple project types detected")
		}
		pack = p
	}
	if pack == nil {
		cli.Fatalf("no buildable project detected")
	}
	pack.Detect()
	return pack
}
