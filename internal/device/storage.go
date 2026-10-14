package device

import (
	"fmt"
)

// FormatStorageBytes formats bytes into human-readable B, KB, MB, or GB string.
func FormatStorageBytes(b uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// GetStorageInfo calculates total, free, and used space for the specified mount path.
func GetStorageInfo(mountPath string) (StorageInfo, error) {
	total, free, err := getDiskSpace(mountPath)
	if err != nil {
		return StorageInfo{}, err
	}

	used := uint64(0)
	if total > free {
		used = total - free
	}

	var pct float64
	if total > 0 {
		pct = (float64(used) / float64(total)) * 100.0
	}

	return StorageInfo{
		TotalBytes:     total,
		FreeBytes:      free,
		UsedBytes:      used,
		UsedPercentage: pct,
		FormattedTotal: FormatStorageBytes(total),
		FormattedFree:  FormatStorageBytes(free),
		FormattedUsed:  FormatStorageBytes(used),
	}, nil
}
