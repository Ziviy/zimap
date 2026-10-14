package provider

import (
	"strings"
	"testing"
)

func TestAvailableProviders(t *testing.T) {
	providers := AvailableProviders()
	if len(providers) < 3 {
		t.Fatalf("expected at least 3 providers, got %d", len(providers))
	}
	ids := make(map[string]bool)
	for _, p := range providers {
		ids[p.ID] = true
	}
	if !ids["geofabrik"] || !ids["osm-france"] || !ids["bbbike"] {
		t.Fatalf("missing essential providers: %+v", ids)
	}
}

func TestMapGeofabrikURLToOSMFrance(t *testing.T) {
	geo := "https://download.geofabrik.de/europe/germany-latest.osm.pbf"
	want := "https://download.openstreetmap.fr/extracts/europe/germany-latest.osm.pbf"
	got := MapGeofabrikURLToOSMFrance(geo)
	if got != want {
		t.Fatalf("MapGeofabrikURLToOSMFrance(%q) = %q, want %q", geo, got, want)
	}

	// Empty URL test
	if MapGeofabrikURLToOSMFrance("") != "" {
		t.Fatal("expected empty string for empty input")
	}
}

func TestBuildBBBikeURL(t *testing.T) {
	url := BuildBBBikeURL("Berlin")
	want := "https://download.bbbike.org/osm/bbbike/Berlin/Berlin.osm.pbf"
	if url != want {
		t.Fatalf("BuildBBBikeURL('Berlin') = %q, want %q", url, want)
	}
}

func TestBuildBBBikeCatalog(t *testing.T) {
	cat := BuildBBBikeCatalog()
	if cat == nil {
		t.Fatal("BuildBBBikeCatalog returned nil")
	}
	roots := cat.Roots()
	if len(roots) == 0 {
		t.Fatal("expected non-empty roots in BBBike catalog")
	}

	// Check that Europe continent exists and contains Berlin
	var europeRootFound bool
	var berlinFound bool
	for _, r := range roots {
		if strings.Contains(strings.ToLower(r.Name), "europe") {
			europeRootFound = true
			for _, country := range cat.Children(r.ID) {
				for _, city := range cat.Children(country.ID) {
					if city.Name == "Berlin" {
						berlinFound = true
						if !strings.HasSuffix(city.PBFURL, "Berlin.osm.pbf") {
							t.Fatalf("city PBFURL = %q, want suffix Berlin.osm.pbf", city.PBFURL)
						}
					}
				}
			}
		}
	}

	if !europeRootFound {
		t.Fatal("expected Europe root in BBBike catalog")
	}
	if !berlinFound {
		t.Fatal("expected Berlin city in BBBike catalog")
	}
}

func TestBuildBBBikeURL_SaintPetersburg(t *testing.T) {
	cases := []string{
		"SaintPetersburg",
		"Saint Petersburg",
		"saint-petersburg",
		"SanktPetersburg",
		"sankt petersburg",
		"petersburg",
	}
	want := "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.pbf"
	for _, c := range cases {
		got := BuildBBBikeURL(c)
		if got != want {
			t.Errorf("BuildBBBikeURL(%q) = %q, want %q", c, got, want)
		}
	}
}

func TestNormalizeBBBikeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "https://download.bbbike.org/osm/bbbike/SaintPetersburg/SaintPetersburg.osm.pbf",
			want:  "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.pbf",
		},
		{
			input: "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.pbf",
			want:  "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.pbf",
		},
		{
			input: "https://download.bbbike.org/osm/bbbike/Munich/Munich.osm.pbf",
			want:  "https://download.bbbike.org/osm/bbbike/Muenchen/Muenchen.osm.pbf",
		},
		{
			input: "https://download.bbbike.org/osm/bbbike/Berlin/Berlin.osm.pbf",
			want:  "https://download.bbbike.org/osm/bbbike/Berlin/Berlin.osm.pbf",
		},
		{
			input: "https://download.geofabrik.de/russia/northwestern-fed-district-latest.osm.pbf",
			want:  "https://download.geofabrik.de/russia/northwestern-fed-district-latest.osm.pbf",
		},
	}

	for _, tt := range tests {
		got := NormalizeBBBikeURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeBBBikeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildBBBikeCatalog_SaintPetersburg(t *testing.T) {
	cat := BuildBBBikeCatalog()
	if cat == nil {
		t.Fatal("BuildBBBikeCatalog returned nil")
	}

	foundSpb := false
	for _, root := range cat.Roots() {
		for _, country := range cat.Children(root.ID) {
			if country.Name == "Russia" {
				for _, city := range cat.Children(country.ID) {
					if strings.Contains(city.Name, "Petersburg") {
						foundSpb = true
						wantURL := "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.pbf"
						if city.PBFURL != wantURL {
							t.Fatalf("expected PBFURL %q, got %q", wantURL, city.PBFURL)
						}
					}
				}
			}
		}
	}

	if !foundSpb {
		t.Fatal("expected Saint Petersburg to be found under Russia in BBBike catalog")
	}
}

func TestBuildBBBikeCatalog_AllCitiesValid(t *testing.T) {
	cat := BuildBBBikeCatalog()
	count := 0
	for _, root := range cat.Roots() {
		for _, country := range cat.Children(root.ID) {
			for _, city := range cat.Children(country.ID) {
				count++
				if city.PBFURL == "" {
					t.Errorf("city %q in country %q has empty PBFURL", city.Name, country.Name)
				}
				if !strings.HasPrefix(city.PBFURL, "https://download.bbbike.org/osm/bbbike/") {
					t.Errorf("city %q has unexpected URL prefix: %s", city.Name, city.PBFURL)
				}
				if !strings.HasSuffix(city.PBFURL, ".osm.pbf") {
					t.Errorf("city %q has unexpected URL suffix: %s", city.Name, city.PBFURL)
				}
			}
		}
	}
	if count != 238 {
		t.Errorf("expected 238 cities in BBBike catalog, got %d", count)
	}
}

func TestBuildBBBikeStyleURL(t *testing.T) {
	cases := []struct {
		city    string
		styleID string
		want    string
	}{
		{"Moscow", "opentopo", "https://download.bbbike.org/osm/bbbike/Moscow/Moscow.osm.garmin-opentopo-latin1.zip"},
		{"Moscow", "onroad", "https://download.bbbike.org/osm/bbbike/Moscow/Moscow.osm.garmin-onroad-latin1.zip"},
		{"Munich", "ontrail", "https://download.bbbike.org/osm/bbbike/Muenchen/Muenchen.osm.garmin-ontrail-latin1.zip"},
		{"Saint Petersburg", "osm", "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.garmin-osm.zip"},
	}
	for _, test := range cases {
		got := BuildBBBikeStyleURL(test.city, test.styleID)
		if got != test.want {
			t.Errorf("BuildBBBikeStyleURL(%q, %q) = %q, want %q", test.city, test.styleID, got, test.want)
		}
	}
	// Unknown style falls back to the raw PBF extract URL.
	if got := BuildBBBikeStyleURL("Moscow", "unknown-style"); !strings.HasSuffix(got, "Moscow.osm.pbf") {
		t.Errorf("BuildBBBikeStyleURL with unknown style = %q, want raw .osm.pbf URL", got)
	}
}

func TestDetectBBBikeStyleID(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://download.bbbike.org/osm/bbbike/Moscow/Moscow.osm.garmin-onroad-latin1.zip", "onroad"},
		{"https://download.bbbike.org/osm/bbbike/Koeln/Koeln.osm.garmin-ontrail-latin1.zip", "ontrail"},
		{"https://download.bbbike.org/osm/bbbike/Muenchen/Muenchen.osm.garmin-opentopo-latin1.zip", "opentopo"},
		{"https://download.bbbike.org/osm/bbbike/Prag/Prag.osm.garmin-osm.zip", "osm"},
		{"https://download.bbbike.org/osm/bbbike/Moscow/Moscow.osm.pbf", ""},
		{"https://download.geofabrik.de/russia-latest.osm.pbf", ""},
	}
	for _, test := range cases {
		if got := DetectBBBikeStyleID(test.url); got != test.want {
			t.Errorf("DetectBBBikeStyleID(%q) = %q, want %q", test.url, got, test.want)
		}
	}
}
