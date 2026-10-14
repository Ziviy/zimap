package device

import "time"

// FreshnessStatus represents the currency status of a map relative to local builds.
type FreshnessStatus string

const (
	StatusUpToDate        FreshnessStatus = "up_to_date"
	StatusUpdateAvailable FreshnessStatus = "update_available"
	StatusNotOnDevice     FreshnessStatus = "not_on_device"
	StatusOnlyOnDevice    FreshnessStatus = "only_on_device"
)

// LocalMapItem represents a locally compiled map on the computer.
type LocalMapItem struct {
	Name      string    `json:"name"`      // Filename on disk, e.g. "russia-central.img"
	Path      string    `json:"path"`      // Absolute path on computer
	Size      int64     `json:"size"`      // Size in bytes
	ModTime   time.Time `json:"modTime"`   // Last modification time
	SourceURL string    `json:"sourceUrl,omitempty"`
	RegionID  string    `json:"regionID,omitempty"`
	Title     string    `json:"title,omitempty"` // User-friendly map name
}

// MapFreshnessItem represents the version comparison for a single map between PC and Garmin device.
type MapFreshnessItem struct {
	ID             string          `json:"id"`            // Unique identifier (filename or slug)
	MapName        string          `json:"mapName"`       // Clean readable map name
	FileName       string          `json:"fileName"`      // Filename (.img)
	Status         FreshnessStatus `json:"status"`        // up_to_date | update_available | not_on_device | only_on_device
	StatusMessage  string          `json:"statusMessage"` // Human-readable status hint

	// Local computer map data (if present)
	LocalPath    string    `json:"localPath,omitempty"`
	LocalSize    int64     `json:"localSize,omitempty"`
	LocalSizeStr string    `json:"localSizeStr,omitempty"`
	LocalModTime time.Time `json:"localModTime,omitempty"`

	// Garmin device map data (if present)
	DevicePath    string    `json:"devicePath,omitempty"`
	DeviceSize    int64     `json:"deviceSize,omitempty"`
	DeviceSizeStr string    `json:"deviceSizeStr,omitempty"`
	DeviceModTime time.Time `json:"deviceModTime,omitempty"`
	DeviceDir     string    `json:"deviceDir,omitempty"` // "GARMIN" or "GARMIN/Map"
	InternalName  string    `json:"internalName,omitempty"`

	// Sync calculations
	BytesToTransfer int64  `json:"bytesToTransfer"`
	HasEnoughSpace  bool   `json:"hasEnoughSpace"`
	DaysOutdated    int    `json:"daysOutdated,omitempty"`
	IsPreinstalled  bool   `json:"isPreinstalled"`          // true if pre-installed factory Garmin map
	SignatureType   string `json:"signatureType,omitempty"` // "gma", "unl", "header_signature", "oem_system"
}

// DeviceFreshnessReport is the aggregated freshness report for a connected Garmin device.
type DeviceFreshnessReport struct {
	MountPath            string             `json:"mountPath"`
	DeviceName           string             `json:"deviceName"`
	CheckedAt            time.Time          `json:"checkedAt"`
	Items                []MapFreshnessItem `json:"items"`
	PreinstalledMaps     []DeviceMap        `json:"preinstalledMaps,omitempty"`
	PreinstalledCount    int                `json:"preinstalledCount"`
	UpToDateCount        int                `json:"upToDateCount"`
	UpdateAvailableCount int                `json:"updateAvailableCount"`
	NotOnDeviceCount     int                `json:"notOnDeviceCount"`
	OnlyOnDeviceCount    int                `json:"onlyOnDeviceCount"`
	TotalSyncBytes       int64              `json:"totalSyncBytes"`
	TotalSyncBytesStr    string             `json:"totalSyncBytesStr"`
	DeviceFreeBytes      uint64             `json:"deviceFreeBytes"`
	DeviceFreeStr        string             `json:"deviceFreeStr"`
}

// SyncProgressEvent represents real-time streaming progress of a file transfer to Garmin.
type SyncProgressEvent struct {
	MapID          string `json:"mapId"`
	FileName       string `json:"fileName"`
	BytesCopied    int64  `json:"bytesCopied"`
	TotalBytes     int64  `json:"totalBytes"`
	Percentage     int    `json:"percentage"`
	SpeedBytesSec  int64  `json:"speedBytesSec"`
	SpeedFormatted string `json:"speedFormatted"`
	ETASeconds     int    `json:"etaSeconds"`
}
