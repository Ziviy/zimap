package converter

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestJavaHeapOptionUsesHalfOfAvailableMemoryByDefault(t *testing.T) {
	t.Setenv("GARMIN_MAP_JAVA_HEAP", "")
	option, err := javaHeapOption()
	if err != nil {
		t.Fatal(err)
	}
	if option != "-XX:MaxRAMPercentage=50.0" {
		t.Fatalf("javaHeapOption() = %q", option)
	}
}

func TestJavaHeapOptionSupportsOverride(t *testing.T) {
	t.Setenv("GARMIN_MAP_JAVA_HEAP", "12g")
	option, err := javaHeapOption()
	if err != nil {
		t.Fatal(err)
	}
	if option != "-Xmx12g" {
		t.Fatalf("javaHeapOption() = %q", option)
	}
}

func TestJavaHeapOptionRejectsInvalidOverride(t *testing.T) {
	t.Setenv("GARMIN_MAP_JAVA_HEAP", "-jar other.jar")
	if _, err := javaHeapOption(); err == nil {
		t.Fatal("javaHeapOption() error = nil, want invalid value error")
	}
}

func TestSplitterArguments(t *testing.T) {
	want := []string{"-Xmx8g", "-jar", "splitter.jar", "--output-dir=tiles", "map.osm.pbf"}
	got := splitterArguments("-Xmx8g", "splitter.jar", "tiles", "map.osm.pbf")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitterArguments() = %#v, want %#v", got, want)
	}
}

func TestMkgmapArgumentsUseSplitterTemplate(t *testing.T) {
	want := []string{"-Xmx8g", "-jar", "mkgmap.jar", "--gmapsupp", "--output-dir=output", "-c", "tiles/template.args"}
	got := mkgmapArguments("-Xmx8g", "mkgmap.jar", "output", "tiles/template.args")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mkgmapArguments() = %#v, want %#v", got, want)
	}
}

func TestMkgmapArgumentsIncludeSelectedRegionName(t *testing.T) {
	want := []string{
		"-Xmx8g", "-jar", "mkgmap.jar", "--gmapsupp", "--output-dir=output",
		"--family-name=Northwestern Federal District",
		"--series-name=Northwestern Federal District",
		"--description=Northwestern Federal District",
		"-c", "tiles/template.args",
	}
	got := mkgmapArguments("-Xmx8g", "mkgmap.jar", "output", "tiles/template.args", "Northwestern Federal District")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mkgmapArguments() = %#v, want %#v", got, want)
	}
}

func TestNormalizeDetailLevel(t *testing.T) {
	cases := []struct {
		raw  string
		want DetailLevel
	}{
		{"", DetailStandard},
		{"standard", DetailStandard},
		{"high", DetailHigh},
		{"topo", DetailTopo},
		{" HIGH ", DetailHigh},
	}
	for _, test := range cases {
		got, err := NormalizeDetailLevel(test.raw)
		if err != nil {
			t.Fatalf("NormalizeDetailLevel(%q) unexpected error: %v", test.raw, err)
		}
		if got != test.want {
			t.Errorf("NormalizeDetailLevel(%q) = %q, want %q", test.raw, got, test.want)
		}
	}
	if _, err := NormalizeDetailLevel("ultra"); err == nil {
		t.Error("NormalizeDetailLevel(\"ultra\") error = nil, want invalid value error")
	}
}

func TestMkgmapDetailArguments(t *testing.T) {
	if got := mkgmapDetailArguments(DetailStandard); got != nil {
		t.Errorf("mkgmapDetailArguments(standard) = %#v, want nil", got)
	}

	wantHigh := []string{"--levels=0:24,1:23,2:21,3:19", "--draw-priority=25"}
	if got := mkgmapDetailArguments(DetailHigh); !reflect.DeepEqual(got, wantHigh) {
		t.Errorf("mkgmapDetailArguments(high) = %#v, want %#v", got, wantHigh)
	}

	wantTopo := []string{
		"--levels=0:24,1:23,2:21,3:19",
		"--draw-priority=20",
		"--transparent",
		"--route",
		"--road-name-pois",
	}
	if got := mkgmapDetailArguments(DetailTopo); !reflect.DeepEqual(got, wantTopo) {
		t.Errorf("mkgmapDetailArguments(topo) = %#v, want %#v", got, wantTopo)
	}
}

func TestMkgmapArgumentsIncludeDetailLevel(t *testing.T) {
	want := []string{
		"-Xmx8g", "-jar", "mkgmap.jar", "--gmapsupp", "--output-dir=output",
		"--family-name=Moscow", "--series-name=Moscow", "--description=Moscow",
		"--levels=0:24,1:23,2:21,3:19",
		"--draw-priority=20",
		"--transparent",
		"--route",
		"--road-name-pois",
		"-c", "tiles/template.args",
	}
	got := mkgmapArguments("-Xmx8g", "mkgmap.jar", "output", "tiles/template.args", "Moscow", "topo")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mkgmapArguments() = %#v, want %#v", got, want)
	}
}

func TestMkgmapArgumentsInvalidDetailLevelIsIgnored(t *testing.T) {
	want := []string{
		"-Xmx8g", "-jar", "mkgmap.jar", "--gmapsupp", "--output-dir=output",
		"--family-name=Moscow", "--series-name=Moscow", "--description=Moscow",
		"-c", "tiles/template.args",
	}
	got := mkgmapArguments("-Xmx8g", "mkgmap.jar", "output", "tiles/template.args", "Moscow", "bogus")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mkgmapArguments() = %#v, want %#v", got, want)
	}
}

func TestIsSupportedInput(t *testing.T) {
	supported := []string{"map.osm.pbf", "map.pbf", "map.osm", "map.osm.gz", "map.osm.bz2", "MAP.OSM.PBF"}
	for _, name := range supported {
		if !IsSupportedInput(name) {
			t.Errorf("IsSupportedInput(%q) = false, want true", name)
		}
	}
	unsupported := []string{"map.img", "map.zip", "map.gpx", "map", "map.pbf.zip"}
	for _, name := range unsupported {
		if IsSupportedInput(name) {
			t.Errorf("IsSupportedInput(%q) = true, want false", name)
		}
	}
}

func TestRunWithContextValidation(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "output")

	// 1. Non-existent file
	_, err := Run(filepath.Join(tempDir, "missing.osm.pbf"), outDir)
	if err == nil || !strings.Contains(err.Error(), "invalid input map") {
		t.Errorf("expected invalid input map error for missing file, got: %v", err)
	}

	// 2. Directory as input map
	dirPath := filepath.Join(tempDir, "folder.osm.pbf")
	_ = os.Mkdir(dirPath, 0755)
	_, err = Run(dirPath, outDir)
	if err == nil || !strings.Contains(err.Error(), "invalid input map") {
		t.Errorf("expected invalid input map error for directory, got: %v", err)
	}

	// 3. Unsupported format
	unsupportedFile := filepath.Join(tempDir, "map.zip")
	_ = os.WriteFile(unsupportedFile, []byte("PK..."), 0644)
	_, err = Run(unsupportedFile, outDir)
	if err == nil || !strings.Contains(err.Error(), "unsupported input format") {
		t.Errorf("expected unsupported input format error, got: %v", err)
	}

	// 4. Invalid detail level
	validInput := filepath.Join(tempDir, "valid.osm.pbf")
	_ = os.WriteFile(validInput, []byte("dummy pbf"), 0644)
	_, err = RunWithContext(nil, validInput, outDir, ConvertOptions{
		DetailLevel: "invalid-detail",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid detail level") {
		t.Errorf("expected invalid detail level error, got: %v", err)
	}
}

