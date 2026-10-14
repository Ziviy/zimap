package cli

import (
	"bufio"
	"bytes"
	"zimap/internal/geofabrik"
	"strings"
	"testing"
)

func testCatalog(t *testing.T) *geofabrik.Catalog {
	t.Helper()
	catalog, err := geofabrik.ParseCatalog(strings.NewReader(`{"features":[
    {"properties":{"id":"africa","name":"Africa","urls":{"pbf":"https://example.test/africa.pbf"}}},
    {"properties":{"id":"europe","name":"Europe","urls":{"pbf":"https://example.test/europe.pbf"}}},
    {"properties":{"id":"germany","parent":"europe","name":"Germany","urls":{"pbf":"https://example.test/germany.pbf"}}},
    {"properties":{"id":"russia","parent":"europe","name":"Russian Federation","urls":{"pbf":"https://example.test/russia.pbf"}}}
  ]}`))
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestSelectGeofabrikRegionSupportsBackNavigation(t *testing.T) {
	input := bufio.NewScanner(strings.NewReader("2\n2\nb\n1\n0\n"))
	var output bytes.Buffer

	region, err := selectGeofabrikRegion(input, &output, testCatalog(t))
	if err != nil {
		t.Fatalf("selectGeofabrikRegion() error = %v", err)
	}
	if region == nil || region.ID != "germany" {
		t.Fatalf("selected region = %#v, want Germany", region)
	}
	if !strings.Contains(output.String(), "b. Назад") {
		t.Fatalf("menu output does not contain back action: %q", output.String())
	}
}

func TestSelectGeofabrikRegionCanQuit(t *testing.T) {
	region, err := selectGeofabrikRegion(bufio.NewScanner(strings.NewReader("q\n")), &bytes.Buffer{}, testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if region != nil {
		t.Fatalf("selected region = %#v, want nil", region)
	}
}

func TestConfirmRegionDownloadAcceptsRussianYes(t *testing.T) {
	region := &geofabrik.Region{Name: "Europe", PBFURL: "https://example.test/europe.pbf"}
	confirmed, err := confirmRegionDownload(bufio.NewScanner(strings.NewReader("да\n")), &bytes.Buffer{}, region, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Fatal("confirmRegionDownload() = false, want true")
	}
}

func TestRegionFileNameUsesIndexNameSafely(t *testing.T) {
	tests := map[string]string{
		"Northwestern Federal District": "northwestern-federal-district",
		"Côte d'Ivoire":                 "côte-d-ivoire",
		"../../North/West":              "north-west",
		"...":                           "region",
	}
	for input, want := range tests {
		if got := regionFileName(input); got != want {
			t.Errorf("regionFileName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatByteSize(t *testing.T) {
	if got := formatByteSize(5 * 1024 * 1024); got != "5.0 MiB" {
		t.Fatalf("formatByteSize() = %q, want 5.0 MiB", got)
	}
}
