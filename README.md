<p align="center">
  <img src="build/appicon.png" width="120" height="120" alt="ZiMap App Icon" />
</p>

<h1 align="center">ZiMap Downloader <br/><sub>[BETA]</sub></h1>

<p align="center">
  <strong>Desktop application for downloading OpenStreetMap regional extracts and compiling them into native <code>.img</code> format for GPS navigators and outdoor smartwatches.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Status-Beta%20v0.0.1-orange.svg" alt="Beta Status" />
  <img src="https://img.shields.io/badge/Made%20with-AI-8A2BE2.svg" alt="Made with AI" />
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Windows-lightgrey.svg" alt="Platforms" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License" />
</p>

---

> **Built with AI / AI-Assisted Engineering**:  
> This project was architected and developed with the active assistance of Artificial Intelligence (LLM coding agents). The application is currently in **BETA** (active development and testing).

---

## Key Features

- **Multi-Provider Map Catalogs**: Instant region search and downloads from Geofabrik, OpenStreetMap France, and pre-compiled BBBike styles (240+ metropolitan cities worldwide).
- **Automated Compilation Pipeline**: Intelligent tile splitting (`splitter`) and compilation (`mkgmap`) into a unified `.img` file complete with routing graphs and contour lines.
- **Device & Smartwatch Sync**:
  - Automatic detection of connected USB mass storage GPS devices.
  - Native macOS MTP driver sidecar (`zimap-mtp`) for seamless direct file transfer to smartwatches without external tools.
  - Real-time map freshness check comparing PC builds with device storage, plus one-click sync.
- **Apple HIG Native Interface**: Frosted glass Vibrancy UI (Dark/Light themes), background download queue, live transfer speed, and ETA indicators.
- **Headless CLI Mode**: Run `zimap` directly from terminal for automated pipelines and headless environments.

---

## Installation & Download

### Option 1: Download Pre-built Release (Recommended)

Pre-compiled standalone packages are available on the [GitHub Releases](https://github.com/Ziviy/zimap/releases) page:

- **macOS**: Download `ZiMap-darwin-arm64.zip` (Apple Silicon M1-M4) or `ZiMap-darwin-amd64.zip` (Intel). Extract the archive and drag `ZiMap.app` to your `/Applications` folder.
- **Windows**: Download `ZiMap-windows-amd64.zip`. Extract the archive and run `ZiMap.exe`.

*Note: Required Java runtime (Adoptium JDK 21) and compilation utilities (`mkgmap`, `splitter`) are automatically downloaded and managed by the app into a local `tools/` directory on first launch if not found in your system PATH.*

### Option 2: Build from Source

#### Prerequisites
- Go 1.22+
- Node.js 18+ and npm
- Wails v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Java 21 runtime

#### Build Instructions

```bash
# Clone repository
git clone https://github.com/Ziviy/zimap.git
cd zimap

# Build MTP sidecar, frontend production assets, and macOS desktop app
make build

# Or launch in development mode with hot reload
wails dev
```

The compiled application bundle will be created at `build/bin/ZiMap.app`.

---

## Command-Line Interface (CLI)

```bash
# Show CLI commands and help
zimap --help

# Download and compile a regional extract
zimap download https://download.geofabrik.de/europe/germany/berlin-latest.osm.pbf

# Convert a local .osm.pbf file
zimap convert ./path/to/region.osm.pbf --detail=high
```

---

## Legal & Trademark Notice

*Garmin® and associated device model names are registered trademarks of Garmin Ltd. or its subsidiaries. ZiMap is an independent, community-driven open-source project and is not affiliated with, sponsored by, or endorsed by Garmin Ltd. References to compatible hardware and file formats are made strictly under nominative fair use to describe technical interoperability with the `.img` map format.*

---

## License

Open-source software licensed under the [MIT License](LICENSE).
