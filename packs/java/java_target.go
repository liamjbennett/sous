package java

import "github.com/opentable/sous/core"

// JavaTarget is the base for all Java targets
type JavaTarget struct {
	*core.TargetBase
	pack *Pack
}

// NewJavaTarget creates a new JavaTarget based on a known target name
// from core.
func NewJavaTarget(name string, pack *Pack) *JavaTarget {
	return &JavaTarget{core.MustGetTargetBase(name, pack), pack}
}
