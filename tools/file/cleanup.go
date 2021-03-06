package file

import (
	"fmt"

	"github.com/opentable/sous/tools/cli"
)

// RemoveOnExit is a convenience wrapper that safely ensures the
// named file is removed after execution, by adding a cleanup task
// to the cli singleton.
func RemoveOnExit(path string) {
	cli.AddCleanupTask(func() error {
		if Exists(path) {
			Remove(path)
		}
		if Exists(path) {
			return fmt.Errorf("Unable to remove temporary object %s; please remove it manually.", path)
		}
		return nil
	})
}

func TemporaryLink(path, newPath string) {
	if Exists(newPath) {
		cli.Fatalf("Unable to link file to %s, it already exists", newPath)
	}
	RemoveOnExit(newPath)
	Link(path, newPath)
}
