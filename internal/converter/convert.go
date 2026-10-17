package converter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"zimap/internal/setup"
)

type MapConverter struct {
}

// Supported input extensions accepted by splitter.jar / mkgmap.jar. The splitter natively
// reads OSM PBF as well as plain and compressed OSM XML extracts.
var supportedInputExtensions = []string{".osm.pbf", ".pbf", ".osm", ".osm.gz", ".osm.bz2"}

// DetailLevel selects the rendering detail preset applied during mkgmap compilation.
type DetailLevel string

const (
	DetailStandard DetailLevel = "standard" // mkgmap default zoom levels (splitter template)
	DetailHigh     DetailLevel = "high"     // extra resolution bits at every zoom level
	DetailTopo     DetailLevel = "topo"     // outdoor profile: extra detail, routing emphasis, transparent overlay
)

// NormalizeDetailLevel validates a detail level string coming from the UI or CLI and
// falls back to DetailStandard for empty values.
func NormalizeDetailLevel(raw string) (DetailLevel, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(DetailStandard):
		return DetailStandard, nil
	case string(DetailHigh):
		return DetailHigh, nil
	case string(DetailTopo):
		return DetailTopo, nil
	default:
		return "", fmt.Errorf("invalid detail level %q; expected one of: standard, high, topo", raw)
	}
}

// mkgmapDetailArguments maps a detail level preset onto additional mkgmap options.
//   - standard: keeps the splitter template zoom levels (current default behaviour)
//   - high:     adds one extra high-resolution zoom level per band (0:24,1:23,2:21,3:19)
//     and raises the draw priority so the map wins over installed base maps
//   - topo:     outdoor profile: maximum detail, transparent rendering to stack over a
//     base map, routing graph and road-name POIs for hiking devices
func mkgmapDetailArguments(level DetailLevel) []string {
	switch level {
	case DetailHigh:
		return []string{
			"--levels=0:24,1:23,2:21,3:19",
			"--draw-priority=25",
		}
	case DetailTopo:
		return []string{
			"--levels=0:24,1:23,2:21,3:19",
			"--draw-priority=20",
			"--transparent",
			"--route",
			"--road-name-pois",
		}
	default:
		return nil
	}
}

type ConvertOptions struct {
	MapName     string
	DetailLevel DetailLevel
	Progress    func(stage string, percent int)
	Log         func(msg string)
}

func Run(mapFilePath string, outputDir string) (string, error) {
	return RunWithContext(context.Background(), mapFilePath, outputDir, ConvertOptions{})
}

func RunNamed(mapFilePath string, outputDir string, mapName string) (string, error) {
	return RunWithContext(context.Background(), mapFilePath, outputDir, ConvertOptions{MapName: mapName})
}

// IsSupportedInput reports whether the given file path has an extension the pipeline accepts.
func IsSupportedInput(mapFilePath string) bool {
	lowered := strings.ToLower(mapFilePath)
	for _, ext := range supportedInputExtensions {
		if strings.HasSuffix(lowered, ext) {
			return true
		}
	}
	return false
}

func RunWithContext(ctx context.Context, mapFilePath string, outputDir string, opts ConvertOptions) (string, error) {
	// Validate the caller's input first: a bad map path must report the input
	// problem, not a missing-tools error. Also keeps the validation testable
	// on machines without Java/mkgmap installed.
	if info, err := os.Stat(mapFilePath); err != nil || info.IsDir() {
		return "", fmt.Errorf("invalid input map %s", mapFilePath)
	}
	if !IsSupportedInput(mapFilePath) {
		return "", fmt.Errorf("unsupported input format %s; supported formats: .osm.pbf, .pbf, .osm, .osm.gz, .osm.bz2", filepath.Base(mapFilePath))
	}
	detailLevel, err := NormalizeDetailLevel(string(opts.DetailLevel))
	if err != nil {
		return "", err
	}
	javaPath, err := setup.JavaBinPath()
	if err != nil {
		return "", err
	}
	mkgmapPath, err := setup.MkgmapBinPath()
	if err != nil {
		return "", err
	}
	splitterPath, err := setup.SplitterBinPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	outputPath := filepath.Join(outputDir, "gmapsupp.img")
	heapOption, err := javaHeapOption()
	if err != nil {
		return "", err
	}
	splitDirectory := filepath.Join(outputDir, ".splitter")
	if err := os.RemoveAll(splitDirectory); err != nil {
		return "", fmt.Errorf("clean splitter workspace: %w", err)
	}
	if err := os.MkdirAll(splitDirectory, 0755); err != nil {
		return "", fmt.Errorf("create splitter workspace: %w", err)
	}

	reportStage := func(stage string) {
		if opts.Progress != nil {
			opts.Progress(stage, 0)
		}
	}

	fmt.Println("Splitting map into Garmin-sized tiles...")
	reportStage("splitter")
	if err := runExternal(ctx, javaPath, opts.Log, splitterArguments(heapOption, splitterPath, splitDirectory, mapFilePath)...); err != nil {
		return "", fmt.Errorf("run splitter: %w", err)
	}
	templatePath := filepath.Join(splitDirectory, "template.args")
	if _, err := os.Stat(templatePath); err != nil {
		return "", fmt.Errorf("splitter did not create %s: %w", templatePath, err)
	}

	fmt.Println("Building Garmin map...")
	reportStage("mkgmap")
	if err := runExternal(ctx, javaPath, opts.Log, mkgmapArguments(heapOption, mkgmapPath, outputDir, templatePath, opts.MapName, string(detailLevel))...); err != nil {
		return "", fmt.Errorf("run mkgmap: %w", err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("mkgmap did not create %s: %w", outputPath, err)
	}
	_ = os.RemoveAll(splitDirectory)
	return outputPath, nil
}

func runExternal(ctx context.Context, name string, logCb func(string), arguments ...string) error {
	cmd := exec.CommandContext(ctx, name, arguments...)

	if logCb != nil {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		go func() {
			scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
			for scanner.Scan() {
				logCb(scanner.Text())
			}
			if err := scanner.Err(); err != nil && logCb != nil {
				logCb(fmt.Sprintf("[WARN] Failed reading tool output: %v", err))
			}
		}()
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func splitterArguments(heapOption string, splitterPath string, outputDir string, mapFilePath string) []string {
	return []string{
		heapOption,
		"-jar",
		splitterPath,
		"--output-dir=" + outputDir,
		mapFilePath,
	}
}

func mkgmapArguments(heapOption string, mkgmapPath string, outputDir string, templatePath string, mapNames ...string) []string {
	arguments := []string{
		heapOption,
		"-jar",
		mkgmapPath,
		"--gmapsupp",
		"--output-dir=" + outputDir,
	}
	if len(mapNames) > 0 && mapNames[0] != "" {
		arguments = append(arguments,
			"--family-name="+mapNames[0],
			"--series-name="+mapNames[0],
			"--description="+mapNames[0],
		)
	}
	arguments = append(arguments, detailArguments(mapNames)...)
	return append(arguments, "-c", templatePath)
}

// detailArguments extracts the trailing DetailLevel argument (if any) from the variadic
// mapNames parameter and maps it onto extra mkgmap options. It keeps mkgmapArguments
// backward compatible for callers that only pass a map name.
func detailArguments(values []string) []string {
	if len(values) < 2 {
		return nil
	}
	detailLevel, err := NormalizeDetailLevel(values[1])
	if err != nil {
		return nil
	}
	return mkgmapDetailArguments(detailLevel)
}

var heapSizePattern = regexp.MustCompile(`^[1-9][0-9]*[gGmMkK]?$`)

func javaHeapOption() (string, error) {
	heapSize := os.Getenv("ZIMAP_JAVA_HEAP")
	if heapSize == "" {
		heapSize = os.Getenv("GARMIN_MAP_JAVA_HEAP")
	}
	if heapSize == "" {
		return "-XX:MaxRAMPercentage=50.0", nil
	}
	if !heapSizePattern.MatchString(heapSize) {
		return "", fmt.Errorf("invalid ZIMAP_JAVA_HEAP %q; expected a value such as 8g or 4096m", heapSize)
	}
	return "-Xmx" + heapSize, nil
}
