package cli

import (
	"bufio"
	"fmt"
	"zimap/internal/converter"
	"zimap/internal/download"
	"zimap/internal/geofabrik"
	"zimap/internal/setup"
	"os"
	"path/filepath"
)

func DownloadCommand() Command {
	return Command{
		Name:        "download",
		Description: "download and convert a regional .osm.pbf file",
		Run: func(args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("usage: zimap download [https://host/region.osm.pbf]")
			}

			if len(args) == 1 {
				return downloadAndConvert(args[0], "", "")
			}

			fmt.Println("Загрузка списка регионов Geofabrik...")
			indexClient := download.New(nil)
			catalog, err := geofabrik.FetchCatalog(indexClient.Client)
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(os.Stdin)
			region, err := selectGeofabrikRegion(scanner, os.Stdout, catalog)
			if err != nil {
				return err
			}
			if region == nil {
				fmt.Println("Загрузка отменена.")
				return nil
			}

			size, sizeErr := download.NewStreaming().ContentLength(region.PBFURL)
			if sizeErr != nil {
				size = -1
			}
			confirmed, err := confirmRegionDownload(scanner, os.Stdout, region, size)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("Загрузка отменена.")
				return nil
			}

			return downloadAndConvert(region.PBFURL, region.Name, regionFileName(region.Name))
		},
	}
}

func downloadAndConvert(rawURL string, regionName string, fileName string) error {
	if err := setup.CheckToolsDirectory(); err != nil {
		return err
	}
	outputDirectory, err := setup.OutputDirectory()
	if err != nil {
		return err
	}

	if fileName == "" {
		fileName = mapFileNameFromURL(rawURL)
	}
	if regionName == "" {
		regionName = fileName
	}

	fmt.Println("Downloading map...")
	convertedMapPath, err := buildFinalMap(outputDirectory, fileName, func(workDirectory string) (string, error) {
		mapFilePath := filepath.Join(workDirectory, fileName+".osm.pbf")
		if _, err := download.ProcessMapDownloadTo(rawURL, mapFilePath); err != nil {
			return "", err
		}
		return converter.RunNamed(mapFilePath, workDirectory, regionName)
	})
	if err != nil {
		return err
	}

	fmt.Printf("Map %s downloaded and converted successfully!\n", regionName)
	fmt.Println("Converted map path:", convertedMapPath)
	return nil
}
