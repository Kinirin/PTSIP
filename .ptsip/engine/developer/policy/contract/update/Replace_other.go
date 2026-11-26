//go:build !windows

package update

import "os"

func replaceFile(from, to string) error { return os.Rename(from, to) }

func publishReplayFile(from, to string) error { return os.Link(from, to) }

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
