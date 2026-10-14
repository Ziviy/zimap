package geofabrik

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const IndexURL = "https://download.geofabrik.de/index-v1-nogeom.json"

type Region struct {
	ID       string
	ParentID string
	Name     string
	PBFURL   string
}

type Catalog struct {
	regions  map[string]*Region
	children map[string][]*Region
	roots    []*Region
}

type indexDocument struct {
	Features []indexFeature `json:"features"`
}

type indexFeature struct {
	Properties struct {
		ID     string `json:"id"`
		Parent string `json:"parent"`
		Name   string `json:"name"`
		URLs   struct {
			PBF string `json:"pbf"`
		} `json:"urls"`
	} `json:"properties"`
}

func FetchCatalog(client *http.Client) (*Catalog, error) {
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Get(IndexURL)
	if err != nil {
		return nil, fmt.Errorf("download Geofabrik index: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download Geofabrik index: URL %s returned status %s", IndexURL, response.Status)
	}

	catalog, err := ParseCatalog(response.Body)
	if err != nil {
		return nil, fmt.Errorf("parse Geofabrik index: %w", err)
	}
	return catalog, nil
}

func ParseCatalog(reader io.Reader) (*Catalog, error) {
	var document indexDocument
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if len(document.Features) == 0 {
		return nil, fmt.Errorf("index contains no regions")
	}

	catalog := &Catalog{
		regions:  make(map[string]*Region, len(document.Features)),
		children: make(map[string][]*Region),
	}
	for _, feature := range document.Features {
		properties := feature.Properties
		properties.ID = strings.TrimSpace(properties.ID)
		properties.Name = strings.TrimSpace(properties.Name)
		if properties.ID == "" || properties.Name == "" {
			return nil, fmt.Errorf("region has an empty id or name")
		}
		if _, exists := catalog.regions[properties.ID]; exists {
			return nil, fmt.Errorf("duplicate region id %q", properties.ID)
		}
		catalog.regions[properties.ID] = &Region{
			ID:       properties.ID,
			ParentID: strings.TrimSpace(properties.Parent),
			Name:     properties.Name,
			PBFURL:   strings.TrimSpace(properties.URLs.PBF),
		}
	}

	for _, region := range catalog.regions {
		if region.ParentID == "" || catalog.regions[region.ParentID] == nil {
			catalog.roots = append(catalog.roots, region)
			continue
		}
		catalog.children[region.ParentID] = append(catalog.children[region.ParentID], region)
	}
	sortRegions(catalog.roots)
	for parentID := range catalog.children {
		sortRegions(catalog.children[parentID])
	}
	return catalog, nil
}

func (catalog *Catalog) Roots() []*Region {
	return catalog.roots
}

func (catalog *Catalog) Children(regionID string) []*Region {
	return catalog.children[regionID]
}

func (catalog *Catalog) Parent(region *Region) *Region {
	if region == nil {
		return nil
	}
	return catalog.regions[region.ParentID]
}

func sortRegions(regions []*Region) {
	sort.Slice(regions, func(left, right int) bool {
		return strings.ToLower(regions[left].Name) < strings.ToLower(regions[right].Name)
	})
}
