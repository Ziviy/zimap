package setup

import (
	"errors"
	"fmt"
	"zimap/internal/download"
)

type SetupManager struct {
	ToolsPath string
	OsType    string
	Client    *download.Client
}

func CheckToolsDirectory() error {
	var toolErrors []error
	if _, err := MkgmapBinPath(); err != nil {
		toolErrors = append(toolErrors, err)
	}
	if _, err := SplitterBinPath(); err != nil {
		toolErrors = append(toolErrors, err)
	}
	if _, err := JavaBinPath(); err != nil {
		toolErrors = append(toolErrors, err)
	}
	return errors.Join(toolErrors...)
}

func UpdateTools() error {
	var toolErrors []error

	for _, tool := range Tools {
		err := tool.RunUpdate(tool)
		if err != nil {
			toolErrors = append(toolErrors, fmt.Errorf("%s: %w", tool.Name, err))
		}
	}

	return errors.Join(toolErrors...)
}

func DownloadTool(tool Tool) error {
	path, err := projectPath(tool.Path)
	if err != nil {
		return err
	}
	err = download.DownloadTool(tool.URL, path)

	if err != nil {
		return err
	}

	return nil
}
