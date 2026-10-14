package device

import (
	"encoding/xml"
	"io"
	"os"
	"time"
)

// StorageInfo represents device disk space capacity and usage metrics.
type StorageInfo struct {
	TotalBytes     uint64  `json:"totalBytes"`
	FreeBytes      uint64  `json:"freeBytes"`
	UsedBytes      uint64  `json:"usedBytes"`
	UsedPercentage float64 `json:"usedPercentage"`
	FormattedTotal string  `json:"formattedTotal"`
	FormattedFree  string  `json:"formattedFree"`
	FormattedUsed  string  `json:"formattedUsed"`
}

// DeviceMap represents a map file located on the Garmin device.
type DeviceMap struct {
	FileName      string    `json:"fileName"`
	FilePath      string    `json:"filePath"`
	InternalName  string    `json:"internalName"` // Read from .img header at offset 0x49
	SizeBytes     int64     `json:"sizeBytes"`
	SizeFormatted string    `json:"sizeFormatted"`
	ModTime       time.Time `json:"modTime"`
	IsCustom       bool      `json:"isCustom"`       // true if custom user map (can be deleted)
	IsPreinstalled bool      `json:"isPreinstalled"` // true if official factory / preinstalled Garmin map
	SignatureType  string    `json:"signatureType,omitempty"` // "gma", "unl", "header_signature", "oem_system"
	RelativeDir    string    `json:"relativeDir"`    // e.g. "GARMIN" or "GARMIN/Map"
	ObjectID       uint32    `json:"objectId,omitempty"`
}

// GarminDevice represents a detected and mounted Garmin GPS device or watch.
type GarminDevice struct {
	MountPath       string      `json:"mountPath"`
	VolumeName      string      `json:"volumeName"`
	ModelName       string      `json:"modelName"` // From GarminDevice.xml Description
	DisplayName     string      `json:"displayName"`
	UnitID          string      `json:"unitId"`
	SoftwareVersion string      `json:"softwareVersion"`
	Storage          StorageInfo `json:"storage"`
	CustomMaps       []DeviceMap `json:"customMaps"`
	PreinstalledMaps []DeviceMap `json:"preinstalledMaps,omitempty"`
	TotalMapsCount   int         `json:"totalMapsCount"` // Custom + factory system maps
	ConnectedAt     time.Time   `json:"connectedAt"`
	IsMTP           bool        `json:"isMtp"`
	StorageID       uint32      `json:"storageId,omitempty"`
	MTPConflict     string      `json:"mtpConflict,omitempty"`
}

// USBStatus provides diagnostics on whether a Garmin device is physically attached via USB,
// especially useful on macOS when MTP watches do not mount as storage volumes.
type USBStatus struct {
	GarminUSBAttached bool   `json:"garminUsbAttached"`
	DeviceDescription string `json:"deviceDescription"`
	NeedsMTPNotice    bool   `json:"needsMtpNotice"`
	IsGarminMode      bool   `json:"isGarminMode"` // true if watch is specifically in proprietary "Garmin" mode (PID 0x0003)
	MTPConflict       string `json:"mtpConflict,omitempty"`
}

// GarminDeviceXML represents the schema of GARMIN/GarminDevice.xml.
type GarminDeviceXML struct {
	XMLName xml.Name `xml:"Device"`
	Model   struct {
		PartNumber      string `xml:"PartNumber"`
		SoftwareVersion string `xml:"SoftwareVersion"`
		Description     string `xml:"Description"`
	} `xml:"Model"`
	UnitID      string `xml:"Id"`
	DisplayName string `xml:"DisplayName"`
}

// ParseGarminDeviceXML reads and unmarshals a GarminDevice.xml stream.
func ParseGarminDeviceXML(r io.Reader) (*GarminDeviceXML, error) {
	var doc GarminDeviceXML
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// ParseGarminDeviceXMLFile reads and unmarshals a GarminDevice.xml file on disk.
func ParseGarminDeviceXMLFile(filePath string) (*GarminDeviceXML, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseGarminDeviceXML(f)
}
