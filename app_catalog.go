package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zimap/internal/geofabrik"
	"zimap/internal/provider"
)

type RegionData struct {
	ID       string       `json:"id"`
	ParentID string       `json:"parentId"`
	Name     string       `json:"name"`
	PBFURL   string       `json:"pbfUrl"`
	Children []RegionData `json:"children"`
}

type CatalogData struct {
	Roots []RegionData `json:"roots"`
}

func (a *App) GetProviders() []provider.Info {
	return provider.AvailableProviders()
}

func (a *App) GetCatalog() (*CatalogData, error) {
	return a.GetCatalogByProvider("geofabrik")
}

func (a *App) ForceRefreshCatalog() (*CatalogData, error) {
	return a.ForceRefreshCatalogByProvider("geofabrik")
}

func (a *App) GetCatalogByProvider(providerID string) (*CatalogData, error) {
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	if providerID == "" {
		providerID = "geofabrik"
	}

	a.catalogMutex.RLock()
	if providerID == "geofabrik" {
		if a.catalogCache != nil && time.Since(a.catalogTime) < 24*time.Hour {
			defer a.catalogMutex.RUnlock()
			return a.catalogCache, nil
		}
	} else if a.providerCatalogs != nil {
		if cached, ok := a.providerCatalogs[providerID]; ok && cached != nil && time.Since(a.providerTimes[providerID]) < 24*time.Hour {
			defer a.catalogMutex.RUnlock()
			return cached, nil
		}
	}
	a.catalogMutex.RUnlock()

	if providerID == "bbbike" {
		cat := provider.BuildBBBikeCatalog()
		return a.populateProviderCache(providerID, cat)
	}

	cacheDir := getCacheDir()
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s-index.json", providerID))
	if info, err := os.Stat(cacheFile); err == nil && time.Since(info.ModTime()) < 24*time.Hour {
		if file, err := os.Open(cacheFile); err == nil {
			defer file.Close()
			if catalog, err := geofabrik.ParseCatalog(file); err == nil {
				return a.populateProviderCache(providerID, catalog)
			}
		}
	}

	return a.ForceRefreshCatalogByProvider(providerID)
}

func (a *App) ForceRefreshCatalogByProvider(providerID string) (*CatalogData, error) {
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	if providerID == "" {
		providerID = "geofabrik"
	}

	if providerID == "bbbike" {
		cat := provider.BuildBBBikeCatalog()
		return a.populateProviderCache(providerID, cat)
	}

	catalog, err := geofabrik.FetchCatalog(nil)
	if err != nil {
		return nil, err
	}

	cacheDir := getCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err == nil {
		cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s-index.json", providerID))
		if resp, err := http.Get(geofabrik.IndexURL); err == nil {
			defer resp.Body.Close()
			if file, err := os.Create(cacheFile); err == nil {
				defer file.Close()
				io.Copy(file, resp.Body)
			}
		}
	}

	return a.populateProviderCache(providerID, catalog)
}

func (a *App) populateMemoryCache(catalog *geofabrik.Catalog) (*CatalogData, error) {
	return a.populateProviderCache("geofabrik", catalog)
}

func (a *App) populateProviderCache(providerID string, catalog *geofabrik.Catalog) (*CatalogData, error) {
	var roots []RegionData
	for _, r := range catalog.Roots() {
		roots = append(roots, mapRegionDataWithProvider(r, catalog, providerID))
	}
	catalogData := &CatalogData{Roots: roots}

	a.catalogMutex.Lock()
	if a.providerCatalogs == nil {
		a.providerCatalogs = make(map[string]*CatalogData)
		a.providerTimes = make(map[string]time.Time)
	}
	if providerID == "geofabrik" {
		a.catalogCache = catalogData
		a.catalogTime = time.Now()
	}
	a.providerCatalogs[providerID] = catalogData
	a.providerTimes[providerID] = time.Now()
	a.catalogMutex.Unlock()

	return catalogData, nil
}

func mapRegionData(r *geofabrik.Region, c *geofabrik.Catalog) RegionData {
	return mapRegionDataWithProvider(r, c, "geofabrik")
}

func mapRegionDataWithProvider(r *geofabrik.Region, c *geofabrik.Catalog, providerID string) RegionData {
	pbfURL := r.PBFURL
	if providerID == "osm-france" && pbfURL != "" {
		pbfURL = provider.MapGeofabrikURLToOSMFrance(pbfURL)
	}

	data := RegionData{
		ID:       r.ID,
		ParentID: r.ParentID,
		Name:     r.Name,
		PBFURL:   pbfURL,
	}
	for _, child := range c.Children(r.ID) {
		data.Children = append(data.Children, mapRegionDataWithProvider(child, c, providerID))
	}
	return data
}

var regionAliases = map[string][]string{
	"россия":               {"russia", "russian federation"},
	"рф":                   {"russia", "russian federation"},
	"российская федерация": {"russia", "russian federation"},
	"беларусь":             {"belarus"},
	"белоруссия":           {"belarus"},
	"рб":                   {"belarus"},
	"украина":              {"ukraine"},
	"казахстан":            {"kazakhstan"},
	"грузия":               {"georgia"},
	"армения":              {"armenia"},
	"азербайджан":          {"azerbaijan"},
	"узбекистан":           {"uzbekistan"},
	"киргизия":             {"kyrgyzstan"},
	"кыргызстан":           {"kyrgyzstan"},
	"таджикистан":          {"tajikistan"},
	"туркменистан":         {"turkmenistan"},
	"туркмения":            {"turkmenistan"},
	"молдова":              {"moldova"},
	"молдавия":             {"moldova"},
	"латвия":               {"latvia"},
	"литва":                {"lithuania"},
	"эстония":              {"estonia"},

	"европа":             {"europe"},
	"германия":           {"germany"},
	"франция":            {"france"},
	"италия":             {"italy"},
	"испания":            {"spain"},
	"португалия":         {"portugal"},
	"польша":             {"poland"},
	"финляндия":          {"finland"},
	"норвегия":           {"norway"},
	"швеция":             {"sweden"},
	"дания":              {"denmark"},
	"нидерланды":         {"netherlands"},
	"голландия":          {"netherlands"},
	"бельгия":            {"belgium"},
	"австрия":            {"austria"},
	"швейцария":          {"switzerland"},
	"великобритания":     {"great-britain", "united kingdom", "britain"},
	"англия":             {"england"},
	"ирландия":           {"ireland"},
	"шотландия":          {"scotland"},
	"чехия":              {"czech"},
	"словакия":           {"slovakia"},
	"венгрия":            {"hungary"},
	"румыния":            {"romania"},
	"болгария":           {"bulgaria"},
	"греция":             {"greece"},
	"кипр":               {"cyprus"},
	"сербия":             {"serbia"},
	"черногория":         {"montenegro"},
	"хорватия":           {"croatia"},
	"словения":           {"slovenia"},
	"босния":             {"bosnia"},
	"македония":          {"macedonia"},
	"северная македония": {"macedonia"},
	"албания":            {"albania"},
	"исландия":           {"iceland"},

	"азия":              {"asia"},
	"турция":            {"turkey"},
	"китай":             {"china"},
	"япония":            {"japan"},
	"корея":             {"korea"},
	"южная корея":       {"south korea", "south-korea"},
	"северная корея":    {"north korea", "north-korea"},
	"таиланд":           {"thailand"},
	"тайланд":           {"thailand"},
	"вьетнам":           {"vietnam"},
	"индия":             {"india"},
	"индонезия":         {"indonesia"},
	"малайзия":          {"malaysia"},
	"филиппины":         {"philippines"},
	"сингапур":          {"singapore"},
	"израиль":           {"israel"},
	"египет":            {"egypt"},
	"оаэ":               {"united arab emirates", "emirates"},
	"эмираты":           {"united arab emirates", "emirates"},
	"иран":              {"iran"},
	"ирак":              {"iraq"},
	"саудовская аравия": {"saudi arabia", "saudi-arabia"},
	"монголия":          {"mongolia"},

	"северная америка": {"north-america", "north america"},
	"южная америка":    {"south-america", "south america"},
	"америка":          {"america", "united states"},
	"сша":              {"united states", "usa"},
	"штаты":            {"united states", "usa"},
	"канада":           {"canada"},
	"мексика":          {"mexico"},
	"бразилия":         {"brazil"},
	"аргентина":        {"argentina"},
	"чили":             {"chile"},
	"куба":             {"cuba"},

	"африка":  {"africa"},
	"юар":     {"south-africa", "south africa"},
	"марокко": {"morocco"},
	"тунис":   {"tunisia"},

	"австралия":      {"australia"},
	"океания":        {"oceania"},
	"новая зеландия": {"new zealand", "new-zealand"},
	"антарктида":     {"antarctica"},

	"москва":                {"moscow"},
	"московская область":    {"moscow", "moskovskaya"},
	"санкт-петербург":       {"saint petersburg", "petersburg", "st-petersburg", "sanktpetersburg", "sankt petersburg", "sankt-petersburg"},
	"петербург":             {"saint petersburg", "petersburg", "sanktpetersburg", "sankt petersburg"},
	"питер":                 {"saint petersburg", "petersburg", "sanktpetersburg", "sankt petersburg"},
	"спб":                   {"saint petersburg", "petersburg", "sanktpetersburg", "sankt petersburg"},
	"ленинградская область": {"leningrad"},
	"крым":                  {"crimea"},
	"кавказ":                {"caucasus"},
	"сибирь":                {"siberia", "siberian"},
	"урал":                  {"ural"},
	"дальний восток":        {"far eastern", "far-eastern"},
	"приволжье":             {"volga"},
	"приволжский":           {"volga"},
	"северо-запад":          {"northwestern"},
	"северо-западный":       {"northwestern"},
	"центральный":           {"central"},
	"южный":                 {"southern"},
	"алтай":                 {"altai"},
	"байкал":                {"baikal"},
	"камчатка":              {"kamchatka"},
	"сахалин":               {"sakhalin"},
	"краснодар":             {"krasnodar"},
	"краснодарский край":    {"krasnodar"},
	"сочи":                  {"sochi"},
	"карелия":               {"karelia"},
	"татарстан":             {"tatarstan"},
	"башкирия":              {"bashkortostan"},
	"башкортостан":          {"bashkortostan"},
	"дагестан":              {"dagestan"},
	"чечня":                 {"chechnya"},

	"вена":           {"vienna", "wien"},
	"мюнхен":         {"munich", "muenchen"},
	"кёльн":          {"cologne", "koeln"},
	"кельн":          {"cologne", "koeln"},
	"нюрнберг":       {"nuremberg", "nuernberg"},
	"дюссельдорф":    {"duesseldorf", "dusseldorf"},
	"цюрих":          {"zurich", "zuerich"},
	"женева":         {"geneva", "genf"},
	"киев":           {"kyiv", "kiev", "kiew"},
	"краков":         {"krakow", "cracow"},
	"прага":          {"prague", "prag"},
	"брюссель":       {"brussels", "bruxelles", "bruessel"},
	"страсбург":      {"strasbourg", "strassburg"},
	"гаага":          {"hague", "den haag", "denhaag"},
	"рио-де-жанейро": {"rio", "riodejaneiro", "rio de janeiro"},
}

func (a *App) SearchCatalog(query string) ([]RegionData, error) {
	return a.SearchCatalogByProvider("geofabrik", query)
}

func (a *App) SearchCatalogByProvider(providerID, query string) ([]RegionData, error) {
	catalogData, err := a.GetCatalogByProvider(providerID)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []RegionData{}, nil
	}

	termSet := make(map[string]bool)
	var searchTerms []string
	addTerm := func(t string) {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !termSet[t] {
			termSet[t] = true
			searchTerms = append(searchTerms, t)
		}
	}

	addTerm(query)

	if aliases, ok := regionAliases[query]; ok {
		for _, a := range aliases {
			addTerm(a)
		}
	} else if len([]rune(query)) >= 3 {
		for ruKey, aliases := range regionAliases {
			if strings.Contains(ruKey, query) {
				for _, a := range aliases {
					addTerm(a)
				}
			}
		}
	}

	matches := func(r RegionData) bool {
		rName := strings.ToLower(r.Name)
		rID := strings.ToLower(r.ID)
		for _, term := range searchTerms {
			if strings.Contains(rName, term) || strings.Contains(rID, term) {
				return true
			}
		}
		return false
	}

	seen := make(map[string]bool)
	var markSeen func(r RegionData)
	markSeen = func(r RegionData) {
		seen[r.ID] = true
		for _, child := range r.Children {
			markSeen(child)
		}
	}

	data := make([]RegionData, 0)
	var search func(regions []RegionData)
	search = func(regions []RegionData) {
		for _, r := range regions {
			if seen[r.ID] {
				continue
			}
			if matches(r) {
				markSeen(r)
				data = append(data, r)
			} else {
				search(r.Children)
			}
		}
	}

	search(catalogData.Roots)

	return data, nil
}
