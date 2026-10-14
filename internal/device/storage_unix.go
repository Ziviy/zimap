//go:build !windows

package device

import (
	"syscall"
)

func getDiskSpace(path string) (total uint64, free uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	total = uint64(stat.Blocks) * uint64(stat.Bsize)
	free = uint64(stat.Bavail) * uint64(stat.Bsize)
	return total, free, nil
}
