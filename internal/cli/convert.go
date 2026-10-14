package cli

import (
	"context"
	"fmt"
	"zimap/internal/converter"
	"zimap/internal/setup"
	"path/filepath"
	"strings"
)

// trimSupportedExtension removes any recognized OSM file extension so that the
// stripped base name is used for the output .img file name.
func trimSupportedExtension(name string) string {
	lowered := strings.ToLower(name)
	for _, ext := range []string{".osm.pbf", ".pbf", ".osm.bz2", ".osm.gz", ".osm"} {
		if strings.HasSuffix(lowered, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}

func ConvertCommand() Command {
	return Command{
		Name:        "convert",
		Description: "convert a local OpenStreetMap file (.osm.pbf, .pbf, .osm, .osm.gz, .osm.bz2) into a named Garmin .img map",
		Run: func(args []string) error {
			detailLevel := converter.DetailStandard
			var inputArguments []string
			for _, argument := range args {
				if value, found := strings.CutPrefix(argument, "--detail="); found {
					detailLevel = converter.DetailLevel(value)
					continue
				}
				inputArguments = append(inputArguments, argument)
			}
			if len(inputArguments) != 1 {
				return fmt.Errorf("usage: zimap convert <path/to/map.osm.pbf> [--detail=standard|high|topo]")
			}
			if _, err := converter.NormalizeDetailLevel(string(detailLevel)); err != nil {
				return err
			}
			mapFilePath, err := filepath.Abs(inputArguments[0])
			if err != nil {
				return fmt.Errorf("resolve input map path: %w", err)
			}
			if err := setup.CheckToolsDirectory(); err != nil {
				return err
			}
			outputDirectory, err := setup.OutputDirectory()
			if err != nil {
				return err
			}

			fileName := regionFileName(trimSupportedExtension(filepath.Base(mapFilePath)))
			convertedMapPath, err := buildFinalMap(outputDirectory, fileName, func(workDirectory string) (string, error) {
				return converter.RunWithContext(context.Background(), mapFilePath, workDirectory, converter.ConvertOptions{
					MapName:     fileName,
					DetailLevel: detailLevel,
				})
			})
			if err != nil {
				return err
			}
			fmt.Println("Map converted successfully!")
			fmt.Println("Converted map path:", convertedMapPath)
			return nil
		},
	}
}
