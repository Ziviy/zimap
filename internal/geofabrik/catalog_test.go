package geofabrik

import (
	"strings"
	"testing"
)

func TestParseCatalogBuildsSortedHierarchy(t *testing.T) {
	catalog, err := ParseCatalog(strings.NewReader(`{
  "type": "FeatureCollection",
  "features": [
    {"properties":{"id":"russia","parent":"europe","name":"Russian Federation","urls":{"pbf":"https://example.test/russia.pbf"}}},
    {"properties":{"id":"africa","name":"Africa","urls":{"pbf":"https://example.test/africa.pbf"}}},
    {"properties":{"id":"germany","parent":"europe","name":"Germany","urls":{"pbf":"https://example.test/germany.pbf"}}},
    {"properties":{"id":"europe","name":"Europe","urls":{"pbf":"https://example.test/europe.pbf"}}}
  ]
}`))
	if err != nil {
		t.Fatalf("ParseCatalog() error = %v", err)
	}

	roots := catalog.Roots()
	if len(roots) != 2 || roots[0].Name != "Africa" || roots[1].Name != "Europe" {
		t.Fatalf("Roots() = %#v, want Africa and Europe", roots)
	}
	children := catalog.Children("europe")
	if len(children) != 2 || children[0].Name != "Germany" || children[1].Name != "Russian Federation" {
		t.Fatalf("Children(europe) = %#v, want Germany and Russian Federation", children)
	}
	if parent := catalog.Parent(children[0]); parent == nil || parent.ID != "europe" {
		t.Fatalf("Parent(Germany) = %#v, want Europe", parent)
	}
}

func TestParseCatalogRejectsDuplicateIDs(t *testing.T) {
	_, err := ParseCatalog(strings.NewReader(`{"features":[
    {"properties":{"id":"same","name":"One"}},
    {"properties":{"id":"same","name":"Two"}}
  ]}`))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("ParseCatalog() error = %v, want duplicate error", err)
	}
}
