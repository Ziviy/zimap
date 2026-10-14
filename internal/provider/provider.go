package provider

import (
	"fmt"
	"zimap/internal/geofabrik"
	"net/url"
	"sort"
	"strings"
)

// Info holds metadata about a map provider.
type Info struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
}

// AvailableProviders returns list of supported map data providers.
func AvailableProviders() []Info {
	return []Info{
		{
			ID:          "geofabrik",
			Name:        "Geofabrik",
			Description: "Полная всемирная иерархия континентов, стран и регионов OpenStreetMap",
			Homepage:    "https://download.geofabrik.de/",
		},
		{
			ID:          "osm-france",
			Name:        "OpenStreetMap France",
			Description: "Зеркало ежедневных региональных выгрузок OSM France",
			Homepage:    "https://download.openstreetmap.fr/extracts/",
		},
		{
			ID:          "bbbike",
			Name:        "BBBike Cities",
			Description: "Выгрузки для более чем 240 мегаполисов и агломераций мира",
			Homepage:    "https://download.bbbike.org/osm/bbbike/",
		},
	}
}

// MapGeofabrikURLToOSMFrance converts a Geofabrik PBF URL into the OpenStreetMap France mirror URL.
func MapGeofabrikURLToOSMFrance(geofabrikURL string) string {
	if geofabrikURL == "" {
		return ""
	}
	u, err := url.Parse(geofabrikURL)
	if err != nil {
		return geofabrikURL
	}
	// Geofabrik path format: /europe/germany-latest.osm.pbf -> /extracts/europe/germany-latest.osm.pbf
	cleanPath := strings.TrimPrefix(u.Path, "/")
	return fmt.Sprintf("https://download.openstreetmap.fr/extracts/%s", cleanPath)
}

// BBBikeCity represents a metropolitan city extract available on BBBike.
type BBBikeCity struct {
	Slug      string // exact BBBike extract directory slug, e.g. "SanktPetersburg". If empty, Name is used.
	Name      string // display name, e.g. "Saint Petersburg"
	Country   string
	Continent string
}

// DefaultBBBikeCities contains the 238 metropolitan areas available on BBBike.
var DefaultBBBikeCities = []BBBikeCity{
	{"Aachen", "Aachen", "Germany", "Europe"},
	{"Aarhus", "Aarhus", "Denmark", "Europe"},
	{"Adelaide", "Adelaide", "Australia", "Oceania"},
	{"Albuquerque", "Albuquerque", "USA", "North America"},
	{"Alexandria", "Alexandria", "Egypt", "Africa"},
	{"Amsterdam", "Amsterdam", "Netherlands", "Europe"},
	{"Antwerpen", "Antwerpen", "Belgium", "Europe"},
	{"Arnhem", "Arnhem", "Netherlands", "Europe"},
	{"Auckland", "Auckland", "New Zealand", "Oceania"},
	{"Augsburg", "Augsburg", "Germany", "Europe"},
	{"Austin", "Austin", "USA", "North America"},
	{"Baghdad", "Baghdad", "Iraq", "Asia"},
	{"Baku", "Baku", "Azerbaijan", "Asia"},
	{"Balaton", "Lake Balaton", "Hungary", "Europe"},
	{"Bamberg", "Bamberg", "Germany", "Europe"},
	{"Bangkok", "Bangkok", "Thailand", "Asia"},
	{"Barcelona", "Barcelona", "Spain", "Europe"},
	{"Basel", "Basel", "Switzerland", "Europe"},
	{"Beijing", "Beijing", "China", "Asia"},
	{"Beirut", "Beirut", "Lebanon", "Asia"},
	{"Berkeley", "Berkeley", "USA", "North America"},
	{"Berlin", "Berlin", "Germany", "Europe"},
	{"Bern", "Bern", "Switzerland", "Europe"},
	{"Bielefeld", "Bielefeld", "Germany", "Europe"},
	{"Birmingham", "Birmingham", "United Kingdom", "Europe"},
	{"Bochum", "Bochum", "Germany", "Europe"},
	{"Bogota", "Bogota", "Colombia", "South America"},
	{"Bombay", "Mumbai (Bombay)", "India", "Asia"},
	{"Bonn", "Bonn", "Germany", "Europe"},
	{"Bordeaux", "Bordeaux", "France", "Europe"},
	{"Boulder", "Boulder", "USA", "North America"},
	{"BrandenburgHavel", "Brandenburg an der Havel", "Germany", "Europe"},
	{"Braunschweig", "Braunschweig", "Germany", "Europe"},
	{"Bremen", "Bremen", "Germany", "Europe"},
	{"Bremerhaven", "Bremerhaven", "Germany", "Europe"},
	{"Brisbane", "Brisbane", "Australia", "Oceania"},
	{"Bristol", "Bristol", "United Kingdom", "Europe"},
	{"Brno", "Brno", "Czech Republic", "Europe"},
	{"Bruegge", "Bruges (Br\u00fcgge)", "Belgium", "Europe"},
	{"Bruessel", "Brussels (Br\u00fcssel)", "Belgium", "Europe"},
	{"Budapest", "Budapest", "Hungary", "Europe"},
	{"BuenosAires", "Buenos Aires", "Argentina", "South America"},
	{"Cairo", "Cairo", "Egypt", "Africa"},
	{"Calgary", "Calgary", "Canada", "North America"},
	{"Cambridge", "Cambridge", "United Kingdom", "Europe"},
	{"CambridgeMa", "Cambridge (MA)", "USA", "North America"},
	{"Canberra", "Canberra", "Australia", "Oceania"},
	{"CapeTown", "Cape Town", "South Africa", "Africa"},
	{"Chemnitz", "Chemnitz", "Germany", "Europe"},
	{"Chicago", "Chicago", "USA", "North America"},
	{"ClermontFerrand", "Clermont-Ferrand", "France", "Europe"},
	{"Colmar", "Colmar", "France", "Europe"},
	{"Copenhagen", "Copenhagen", "Denmark", "Europe"},
	{"Cork", "Cork", "Ireland", "Europe"},
	{"Corsica", "Corsica", "France", "Europe"},
	{"Corvallis", "Corvallis", "USA", "North America"},
	{"Cottbus", "Cottbus", "Germany", "Europe"},
	{"Cracow", "Krakow (Cracow)", "Poland", "Europe"},
	{"CraterLake", "Crater Lake", "USA", "North America"},
	{"Curitiba", "Curitiba", "Brazil", "South America"},
	{"Cusco", "Cusco", "Peru", "South America"},
	{"Dallas", "Dallas", "USA", "North America"},
	{"Damaskus", "Damascus", "Syria", "Asia"},
	{"Darmstadt", "Darmstadt", "Germany", "Europe"},
	{"Davis", "Davis", "USA", "North America"},
	{"DenHaag", "The Hague (Den Haag)", "Netherlands", "Europe"},
	{"Denver", "Denver", "USA", "North America"},
	{"Dessau", "Dessau", "Germany", "Europe"},
	{"Dortmund", "Dortmund", "Germany", "Europe"},
	{"Dresden", "Dresden", "Germany", "Europe"},
	{"Dublin", "Dublin", "Ireland", "Europe"},
	{"Duesseldorf", "Dusseldorf (D\u00fcsseldorf)", "Germany", "Europe"},
	{"Duisburg", "Duisburg", "Germany", "Europe"},
	{"Edinburgh", "Edinburgh", "United Kingdom", "Europe"},
	{"Eindhoven", "Eindhoven", "Netherlands", "Europe"},
	{"Emden", "Emden", "Germany", "Europe"},
	{"Erfurt", "Erfurt", "Germany", "Europe"},
	{"Erlangen", "Erlangen", "Germany", "Europe"},
	{"Eugene", "Eugene", "USA", "North America"},
	{"Flensburg", "Flensburg", "Germany", "Europe"},
	{"FortCollins", "Fort Collins", "USA", "North America"},
	{"Frankfurt", "Frankfurt am Main", "Germany", "Europe"},
	{"FrankfurtOder", "Frankfurt (Oder)", "Germany", "Europe"},
	{"Freiburg", "Freiburg", "Germany", "Europe"},
	{"Gdansk", "Gdansk", "Poland", "Europe"},
	{"Genf", "Geneva (Genf)", "Switzerland", "Europe"},
	{"Gent", "Ghent (Gent)", "Belgium", "Europe"},
	{"Gera", "Gera", "Germany", "Europe"},
	{"Glasgow", "Glasgow", "United Kingdom", "Europe"},
	{"Gliwice", "Gliwice", "Poland", "Europe"},
	{"Goerlitz", "G\u00f6rlitz", "Germany", "Europe"},
	{"Goeteborg", "Gothenburg (G\u00f6teborg)", "Sweden", "Europe"},
	{"Goettingen", "G\u00f6ttingen", "Germany", "Europe"},
	{"Graz", "Graz", "Austria", "Europe"},
	{"Groningen", "Groningen", "Netherlands", "Europe"},
	{"Halifax", "Halifax", "Canada", "North America"},
	{"Halle", "Halle (Saale)", "Germany", "Europe"},
	{"Hamburg", "Hamburg", "Germany", "Europe"},
	{"Hamm", "Hamm", "Germany", "Europe"},
	{"Hannover", "Hannover", "Germany", "Europe"},
	{"Heilbronn", "Heilbronn", "Germany", "Europe"},
	{"Helsinki", "Helsinki", "Finland", "Europe"},
	{"Hertogenbosch", "s-Hertogenbosch", "Netherlands", "Europe"},
	{"Huntsville", "Huntsville", "USA", "North America"},
	{"Innsbruck", "Innsbruck", "Austria", "Europe"},
	{"Istanbul", "Istanbul", "Turkey", "Europe"},
	{"Jena", "Jena", "Germany", "Europe"},
	{"Jerusalem", "Jerusalem", "Israel", "Asia"},
	{"Johannesburg", "Johannesburg", "South Africa", "Africa"},
	{"Kaiserslautern", "Kaiserslautern", "Germany", "Europe"},
	{"Karlsruhe", "Karlsruhe", "Germany", "Europe"},
	{"Kassel", "Kassel", "Germany", "Europe"},
	{"Katowice", "Katowice", "Poland", "Europe"},
	{"Kaunas", "Kaunas", "Lithuania", "Europe"},
	{"Kiel", "Kiel", "Germany", "Europe"},
	{"Kiew", "Kyiv (Kiev)", "Ukraine", "Europe"},
	{"Koblenz", "Koblenz", "Germany", "Europe"},
	{"Koeln", "Cologne (K\u00f6ln)", "Germany", "Europe"},
	{"Konstanz", "Konstanz", "Germany", "Europe"},
	{"LaPaz", "La Paz", "Bolivia", "South America"},
	{"LaPlata", "La Plata", "Argentina", "South America"},
	{"LakeGarda", "Lake Garda", "Italy", "Europe"},
	{"Lausanne", "Lausanne", "Switzerland", "Europe"},
	{"Leeds", "Leeds", "United Kingdom", "Europe"},
	{"Leipzig", "Leipzig", "Germany", "Europe"},
	{"Lima", "Lima", "Peru", "South America"},
	{"Linz", "Linz", "Austria", "Europe"},
	{"Lisbon", "Lisbon", "Portugal", "Europe"},
	{"Liverpool", "Liverpool", "United Kingdom", "Europe"},
	{"Ljubljana", "Ljubljana", "Slovenia", "Europe"},
	{"Lodz", "Lodz", "Poland", "Europe"},
	{"London", "London", "United Kingdom", "Europe"},
	{"LosAngeles", "Los Angeles", "USA", "North America"},
	{"Luebeck", "L\u00fcbeck", "Germany", "Europe"},
	{"Luxemburg", "Luxembourg", "Luxembourg", "Europe"},
	{"Lyon", "Lyon", "France", "Europe"},
	{"Maastricht", "Maastricht", "Netherlands", "Europe"},
	{"Madison", "Madison", "USA", "North America"},
	{"Madrid", "Madrid", "Spain", "Europe"},
	{"Magdeburg", "Magdeburg", "Germany", "Europe"},
	{"Mainz", "Mainz", "Germany", "Europe"},
	{"Malmoe", "Malm\u00f6", "Sweden", "Europe"},
	{"Manchester", "Manchester", "United Kingdom", "Europe"},
	{"Mannheim", "Mannheim", "Germany", "Europe"},
	{"Marseille", "Marseille", "France", "Europe"},
	{"Melbourne", "Melbourne", "Australia", "Oceania"},
	{"Memphis", "Memphis", "USA", "North America"},
	{"MexicoCity", "Mexico City", "Mexico", "North America"},
	{"Miami", "Miami", "USA", "North America"},
	{"Minsk", "Minsk", "Belarus", "Europe"},
	{"Moenchengladbach", "M\u00f6nchengladbach", "Germany", "Europe"},
	{"Montevideo", "Montevideo", "Uruguay", "South America"},
	{"Montpellier", "Montpellier", "France", "Europe"},
	{"Montreal", "Montreal", "Canada", "North America"},
	{"Moscow", "Moscow", "Russia", "Europe"},
	{"Muenchen", "Munich (M\u00fcnchen)", "Germany", "Europe"},
	{"Muenster", "M\u00fcnster", "Germany", "Europe"},
	{"NewDelhi", "New Delhi", "India", "Asia"},
	{"NewOrleans", "New Orleans", "USA", "North America"},
	{"NewYork", "New York", "USA", "North America"},
	{"Nuernberg", "Nuremberg (N\u00fcrnberg)", "Germany", "Europe"},
	{"Oldenburg", "Oldenburg", "Germany", "Europe"},
	{"Oranienburg", "Oranienburg", "Germany", "Europe"},
	{"Orlando", "Orlando", "USA", "North America"},
	{"Oslo", "Oslo", "Norway", "Europe"},
	{"Osnabrueck", "Osnabr\u00fcck", "Germany", "Europe"},
	{"Ostrava", "Ostrava", "Czech Republic", "Europe"},
	{"Ottawa", "Ottawa", "Canada", "North America"},
	{"Paderborn", "Paderborn", "Germany", "Europe"},
	{"Palma", "Palma de Mallorca", "Spain", "Europe"},
	{"PaloAlto", "Palo Alto", "USA", "North America"},
	{"Paris", "Paris", "France", "Europe"},
	{"Perth", "Perth", "Australia", "Oceania"},
	{"Philadelphia", "Philadelphia", "USA", "North America"},
	{"PhnomPenh", "Phnom Penh", "Cambodia", "Asia"},
	{"Portland", "Portland", "USA", "North America"},
	{"PortlandME", "Portland (ME)", "USA", "North America"},
	{"Porto", "Porto", "Portugal", "Europe"},
	{"PortoAlegre", "Porto Alegre", "Brazil", "South America"},
	{"Potsdam", "Potsdam", "Germany", "Europe"},
	{"Poznan", "Poznan", "Poland", "Europe"},
	{"Prag", "Prague (Prag)", "Czech Republic", "Europe"},
	{"Providence", "Providence", "USA", "North America"},
	{"Regensburg", "Regensburg", "Germany", "Europe"},
	{"Riga", "Riga", "Latvia", "Europe"},
	{"RiodeJaneiro", "Rio de Janeiro", "Brazil", "South America"},
	{"Rostock", "Rostock", "Germany", "Europe"},
	{"Rotterdam", "Rotterdam", "Netherlands", "Europe"},
	{"Ruegen", "R\u00fcgen", "Germany", "Europe"},
	{"Saarbruecken", "Saarbr\u00fccken", "Germany", "Europe"},
	{"Sacramento", "Sacramento", "USA", "North America"},
	{"Saigon", "Ho Chi Minh City (Saigon)", "Vietnam", "Asia"},
	{"Salzburg", "Salzburg", "Austria", "Europe"},
	{"SanFrancisco", "San Francisco", "USA", "North America"},
	{"SanJose", "San Jose", "USA", "North America"},
	{"SanktPetersburg", "Saint Petersburg", "Russia", "Europe"},
	{"SantaBarbara", "Santa Barbara", "USA", "North America"},
	{"SantaCruz", "Santa Cruz", "USA", "North America"},
	{"Santiago", "Santiago", "Chile", "South America"},
	{"Sarajewo", "Sarajevo", "Bosnia and Herzegovina", "Europe"},
	{"Schwerin", "Schwerin", "Germany", "Europe"},
	{"Seattle", "Seattle", "USA", "North America"},
	{"Seoul", "Seoul", "South Korea", "Asia"},
	{"Sheffield", "Sheffield", "United Kingdom", "Europe"},
	{"Singapore", "Singapore", "Singapore", "Asia"},
	{"Sofia", "Sofia", "Bulgaria", "Europe"},
	{"Stockholm", "Stockholm", "Sweden", "Europe"},
	{"Stockton", "Stockton", "USA", "North America"},
	{"Strassburg", "Strasbourg (Stra\u00dfburg)", "France", "Europe"},
	{"Stuttgart", "Stuttgart", "Germany", "Europe"},
	{"Sucre", "Sucre", "Bolivia", "South America"},
	{"Sydney", "Sydney", "Australia", "Oceania"},
	{"Szczecin", "Szczecin", "Poland", "Europe"},
	{"Tallinn", "Tallinn", "Estonia", "Europe"},
	{"Tehran", "Tehran", "Iran", "Asia"},
	{"Tilburg", "Tilburg", "Netherlands", "Europe"},
	{"Tokyo", "Tokyo", "Japan", "Asia"},
	{"Toronto", "Toronto", "Canada", "North America"},
	{"Toulouse", "Toulouse", "France", "Europe"},
	{"Trondheim", "Trondheim", "Norway", "Europe"},
	{"Tucson", "Tucson", "USA", "North America"},
	{"Turin", "Turin", "Italy", "Europe"},
	{"UlanBator", "Ulaanbaatar", "Mongolia", "Asia"},
	{"Ulm", "Ulm", "Germany", "Europe"},
	{"Usedom", "Usedom", "Germany", "Europe"},
	{"Utrecht", "Utrecht", "Netherlands", "Europe"},
	{"Vancouver", "Vancouver", "Canada", "North America"},
	{"Victoria", "Victoria", "Canada", "North America"},
	{"WarenMueritz", "Waren (M\u00fcritz)", "Germany", "Europe"},
	{"Warsaw", "Warsaw", "Poland", "Europe"},
	{"WashingtonDC", "Washington, D.C.", "USA", "North America"},
	{"Waterloo", "Waterloo", "Canada", "North America"},
	{"Wien", "Vienna (Wien)", "Austria", "Europe"},
	{"Wroclaw", "Wroclaw", "Poland", "Europe"},
	{"Wuerzburg", "W\u00fcrzburg", "Germany", "Europe"},
	{"Wuppertal", "Wuppertal", "Germany", "Europe"},
	{"Zagreb", "Zagreb", "Croatia", "Europe"},
	{"Zuerich", "Zurich (Z\u00fcrich)", "Switzerland", "Europe"},
}

// BBBikeSlugAliases maps known English, common, or alternate spellings of city names to canonical BBBike directory slugs.
var BBBikeSlugAliases = map[string]string{
	"saintpetersburg":  "SanktPetersburg",
	"saint-petersburg": "SanktPetersburg",
	"saint petersburg": "SanktPetersburg",
	"st-petersburg":    "SanktPetersburg",
	"st petersburg":    "SanktPetersburg",
	"stpetersburg":     "SanktPetersburg",
	"petersburg":       "SanktPetersburg",
	"sanktpetersburg":  "SanktPetersburg",
	"sankt-petersburg": "SanktPetersburg",
	"sankt petersburg": "SanktPetersburg",
	"spb":              "SanktPetersburg",
	"munich":           "Muenchen",
	"muenchen":         "Muenchen",
	"cologne":          "Koeln",
	"koeln":            "Koeln",
	"dusseldorf":       "Duesseldorf",
	"duesseldorf":      "Duesseldorf",
	"nuremberg":        "Nuernberg",
	"nuernberg":        "Nuernberg",
	"vienna":           "Wien",
	"wien":             "Wien",
	"zurich":           "Zuerich",
	"zuerich":          "Zuerich",
	"geneva":           "Genf",
	"genf":             "Genf",
	"kyiv":             "Kiew",
	"kiev":             "Kiew",
	"kiew":             "Kiew",
	"krakow":           "Cracow",
	"cracow":           "Cracow",
	"prague":           "Prag",
	"prag":             "Prag",
	"brussels":         "Bruessel",
	"bruxelles":        "Bruessel",
	"bruessel":         "Bruessel",
	"sarajevo":         "Sarajewo",
	"sarajewo":         "Sarajewo",
	"strasbourg":       "Strassburg",
	"strassburg":       "Strassburg",
	"damascus":         "Damaskus",
	"damaskus":         "Damaskus",
	"the hague":        "DenHaag",
	"thehague":         "DenHaag",
	"den haag":         "DenHaag",
	"denhaag":          "DenHaag",
	"rio de janeiro":   "RiodeJaneiro",
	"riodejaneiro":     "RiodeJaneiro",
	"delhi":            "NewDelhi",
	"new delhi":        "NewDelhi",
	"newdelhi":         "NewDelhi",
	"mumbai":           "Bombay",
	"bombay":           "Bombay",
	"bruges":           "Bruegge",
	"bruegge":          "Bruegge",
	"ghent":            "Gent",
	"gent":             "Gent",
	"gothenburg":       "Goeteborg",
	"goeteborg":        "Goeteborg",
	"malmo":            "Malmoe",
	"malmoe":           "Malmoe",
	"luxembourg":       "Luxemburg",
	"luxemburg":        "Luxemburg",
	"ulaanbaatar":      "UlanBator",
	"ulanbator":        "UlanBator",
	"saigon":           "Saigon",
	"hochiminh":        "Saigon",
	"ho chi minh":      "Saigon",
}

// BBBikeGarminStyle represents one of the pre-compiled Garmin map styles offered by
// BBBike for every city extract (e.g. "Moscow.osm.garmin-onroad-latin1.zip").
type BBBikeGarminStyle struct {
	ID          string `json:"id"`          // style id used in API/UI: "onroad", "ontrail", "opentopo", "osm"
	Name        string `json:"name"`        // display name, e.g. "Onroad"
	Description string `json:"description"` // short human-readable description
	Suffix      string `json:"suffix"`      // file suffix on download.bbbike.org, e.g. ".osm.garmin-onroad-latin1.zip"
	Compiled    bool   `json:"compiled"`    // true when BBBike ships a ready gmapsupp.img (no local compile needed)
}

// BBBikeGarminStyles lists the pre-compiled Garmin styles published by BBBike for each city
// at https://download.bbbike.org/osm/bbbike/<City>/.
var BBBikeGarminStyles = []BBBikeGarminStyle{
	{
		ID:          "onroad",
		Name:        "Onroad",
		Description: "Roads & routing only — smallest size, fastest transfer",
		Suffix:      ".osm.garmin-onroad-latin1.zip",
		Compiled:    true,
	},
	{
		ID:          "ontrail",
		Name:        "Ontrail",
		Description: "Bicycle & hiking trails emphasis for cyclists",
		Suffix:      ".osm.garmin-ontrail-latin1.zip",
		Compiled:    true,
	},
	{
		ID:          "opentopo",
		Name:        "OpenTopoMap",
		Description: "Topographic map with elevation contours — hiking & outdoors",
		Suffix:      ".osm.garmin-opentopo-latin1.zip",
		Compiled:    true,
	},
	{
		ID:          "osm",
		Name:        "OSM (UTF-8)",
		Description: "Full standard OSM map, UTF-8 encoded",
		Suffix:      ".osm.garmin-osm.zip",
		Compiled:    true,
	},
}

// BuildBBBikeStyleURL formats the download URL for a pre-compiled BBBike Garmin style ZIP
// of the given city. It resolves any known city alias variations first.
func BuildBBBikeStyleURL(cityName string, styleID string) string {
	cleanName := strings.TrimSpace(cityName)
	normalizedKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(cleanName, "-", ""), " ", ""))
	if canonical, ok := BBBikeSlugAliases[normalizedKey]; ok {
		cleanName = canonical
	}
	suffix := ".osm.pbf"
	for _, s := range BBBikeGarminStyles {
		if s.ID == styleID {
			suffix = s.Suffix
			break
		}
	}
	return fmt.Sprintf("https://download.bbbike.org/osm/bbbike/%s/%s%s", cleanName, cleanName, suffix)
}

// DetectBBBikeStyleID returns the BBBike pre-compiled Garmin style id for a download URL
// (e.g. ".../Moscow.osm.garmin-opentopo-latin1.zip" -> "opentopo"), or "" when the URL
// does not reference a pre-compiled Garmin style archive.
func DetectBBBikeStyleID(downloadURL string) string {
	lowered := strings.ToLower(downloadURL)
	for _, style := range BBBikeGarminStyles {
		if strings.HasSuffix(lowered, strings.ToLower(style.Suffix)) {
			return style.ID
		}
	}
	return ""
}

// BuildBBBikeURL formats the download URL for a given BBBike city extract, resolving any known aliases.
func BuildBBBikeURL(cityName string) string {
	cleanName := strings.TrimSpace(cityName)
	normalizedKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(cleanName, "-", ""), " ", ""))
	if canonical, ok := BBBikeSlugAliases[normalizedKey]; ok {
		cleanName = canonical
	}
	return fmt.Sprintf("https://download.bbbike.org/osm/bbbike/%s/%s.osm.pbf", cleanName, cleanName)
}

// NormalizeBBBikeURL normalizes known city alias variations in BBBike download URLs to canonical BBBike paths.
func NormalizeBBBikeURL(rawURL string) string {
	if !strings.Contains(rawURL, "bbbike.org/osm/bbbike/") {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parts := strings.Split(parsed.Path, "/")
	for i, p := range parts {
		if p == "bbbike" && i+1 < len(parts) {
			citySegment := parts[i+1]
			normalizedKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(citySegment, "-", ""), " ", ""))
			if canonical, ok := BBBikeSlugAliases[normalizedKey]; ok {
				parts[i+1] = canonical
				if i+2 < len(parts) && strings.HasSuffix(parts[i+2], ".osm.pbf") {
					parts[i+2] = canonical + ".osm.pbf"
				}
				parsed.Path = strings.Join(parts, "/")
				return parsed.String()
			}
		}
	}
	return rawURL
}

// BuildBBBikeCatalog constructs a hierarchical continent -> country -> city catalog for BBBike.
func BuildBBBikeCatalog() *geofabrik.Catalog {
	type cityNode struct {
		id     string
		name   string
		pbfURL string
	}

	continents := make(map[string]map[string][]cityNode)

	for _, c := range DefaultBBBikeCities {
		slug := c.Slug
		if slug == "" {
			slug = c.Name
		}
		cont := c.Continent
		if cont == "" {
			cont = "Other"
		}
		country := c.Country
		if country == "" {
			country = "Other"
		}

		if continents[cont] == nil {
			continents[cont] = make(map[string][]cityNode)
		}
		continents[cont][country] = append(continents[cont][country], cityNode{
			id:     fmt.Sprintf("bbbike-%s", strings.ToLower(slug)),
			name:   c.Name,
			pbfURL: BuildBBBikeURL(slug),
		})
	}

	type featureProp struct {
		ID     string `json:"id"`
		Parent string `json:"parent,omitempty"`
		Name   string `json:"name"`
		URLs   struct {
			PBF string `json:"pbf,omitempty"`
		} `json:"urls"`
	}

	type featureDoc struct {
		Properties featureProp `json:"properties"`
	}

	var features []featureDoc

	var sortedContinents []string
	for cont := range continents {
		sortedContinents = append(sortedContinents, cont)
	}
	sort.Strings(sortedContinents)

	for _, cont := range sortedContinents {
		contID := fmt.Sprintf("bbbike-%s", strings.ToLower(strings.ReplaceAll(cont, " ", "-")))
		fCont := featureDoc{
			Properties: featureProp{
				ID:   contID,
				Name: cont,
			},
		}
		features = append(features, fCont)

		var sortedCountries []string
		for country := range continents[cont] {
			sortedCountries = append(sortedCountries, country)
		}
		sort.Strings(sortedCountries)

		for _, country := range sortedCountries {
			countryID := fmt.Sprintf("%s-%s", contID, strings.ToLower(strings.ReplaceAll(country, " ", "-")))
			fCountry := featureDoc{
				Properties: featureProp{
					ID:     countryID,
					Parent: contID,
					Name:   country,
				},
			}
			features = append(features, fCountry)

			cities := continents[cont][country]
			sort.Slice(cities, func(i, j int) bool {
				return cities[i].name < cities[j].name
			})

			for _, city := range cities {
				fCity := featureDoc{
					Properties: featureProp{
						ID:     city.id,
						Parent: countryID,
						Name:   city.name,
					},
				}
				fCity.Properties.URLs.PBF = city.pbfURL
				features = append(features, fCity)
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(`{"features":[`)
	for i, f := range features {
		if i > 0 {
			builder.WriteString(",")
		}
		pbfPart := ""
		if f.Properties.URLs.PBF != "" {
			pbfPart = fmt.Sprintf(`,"urls":{"pbf":%q}`, f.Properties.URLs.PBF)
		}
		parentPart := ""
		if f.Properties.Parent != "" {
			parentPart = fmt.Sprintf(`,"parent":%q`, f.Properties.Parent)
		}
		builder.WriteString(fmt.Sprintf(`{"properties":{"id":%q,"name":%q%s%s}}`,
			f.Properties.ID, f.Properties.Name, parentPart, pbfPart))
	}
	builder.WriteString(`]}`)

	cat, _ := geofabrik.ParseCatalog(strings.NewReader(builder.String()))
	return cat
}
