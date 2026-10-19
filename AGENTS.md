# AGENTS.md — Agent & Developer Guide for ZiMap Downloader

Welcome, AI Agent or human contributor! This document serves as the primary technical briefing for working with the **ZiMap Downloader** (formerly `zimap`) codebase. Read this document thoroughly before proposing or implementing changes.

**Start with [the architecture and module guide](docs/ARCHITECTURE.md) for source orientation.** It maps tasks to files, explains module ownership and runtime workflows, and records IPC contracts, storage, testing prerequisites and implementation caveats. Use its task-to-file index instead of re-examining the entire project. Keep it updated when those boundaries change; use `TODO.md` for planned work rather than treating historical plans as implemented behavior.

---

## 1. Project Overview & Mission

**ZiMap Downloader** is a cross-platform desktop application and command-line utility for downloading OpenStreetMap (OSM) regional extracts and compiling them into GPS-compatible **`.img` format** (ready to copy directly onto GPS navigators, handhelds, and watch devices).

### Core Pipeline:
1. **Source Data**: Downloads `.osm.pbf` extracts from Geofabrik (or alternative providers such as OpenStreetMap France and BBBike).
2. **Splitting (`splitter.jar`)**: Divides large regional OSM PBF files into memory-efficient geographic tiles.
3. **Compiling (`mkgmap.jar`)**: Renders tiles into compatible `.img` format with routing graphs, points of interest, and elevation contours.
4. **Output**: Produces a unified `gmapsupp.img` or named `<region>.img` in the `output/` directory.

### Visual & Platform Goals:
- Native look-and-feel following **Apple Human Interface Guidelines (HIG)**: frosted glass vibrancy (`backdrop-filter: blur(25px)`), dynamic Dark/Light themes, SF Pro / SF Mono typography, segmented controls, and pipeline progress steppers.
- Runs natively on both **macOS (Apple Silicon & Intel)** and **Windows (10/11 x86-64)** with zero external runtime prerequisites on client machines (self-contained Go + Webview2 / WebKit).

---

## 2. Technology Stack

- **Desktop Framework**: [Wails v2](https://wails.io/) (`github.com/wailsapp/wails/v2 v2.15.0`)
- **Backend Language**: Go, as required by [`go.mod`](go.mod) (currently 1.26.5+)
- **Frontend Stack**: React 19, Vite, Lucide React icons
- **Styling**: Pure CSS with Apple HIG design system tokens (`AppleTheme.css`)
- **OS Runtimes**: WebKit (macOS), Evergreen WebView2 (Windows)
- **External Tools Managed**: Adoptium Temurin JDK 21, Mkgmap (r4924), Splitter (r654)

---

## 3. Directory Layout

```
zimap/
├── AGENTS.md                    # This developer & agent guide
├── TODO.md                      # Active backlog & completed task history
├── docs/ARCHITECTURE.md         # Current module map, workflows, IPC & agent handoff
├── app.go                       # Wails backend service struct (App) lifecycle & shared types
├── app_catalog.go               # Catalog fetching, caching, search & Russian aliases
├── app_convert.go               # Conversion pipeline (URL & local), BBBike direct downloads, cancel
├── app_device.go                # Device discovery, protected map management, freshness & sync
├── app_outputs.go               # Output maps listing, deletion, Finder reveal, file export
├── app_tools.go                 # Adoptium JDK, mkgmap, splitter check & updates, system info
├── app_updater.go               # GitHub Releases updater & semver comparator
├── app_storage.go               # Storage metrics, temp & cache cleanup
├── app_proxy.go                 # Proxy configuration (HTTP/SOCKS5)
├── app_notify.go                # Native OS desktop notifications (macOS, Windows, Linux)
├── app_test.go                  # Backend unit tests with mock HTTP servers
├── main.go                      # Application entrypoint: CLI dispatcher + Wails Desktop runner
├── wails.json                   # Wails project build configuration
├── go.mod / go.sum              # Go dependencies
│
├── cache/                       # Local disk cache (geofabrik-index.json, TTL 24h)
├── cmd/                         # CLI-only entry point & zimap-mtp USB sidecar
├── internal/
│   ├── archive/                 # .zip, .tar, and .tar.gz extraction utilities & tests
│   ├── cli/                     # CLI commands registry (download, convert, update) & tests
│   ├── converter/               # Splitter & Mkgmap execution with progress and context cancel & tests
│   ├── device/                  # Mounted/MTP devices, map protection, freshness & transfer engine
│   ├── download/                # Streaming HTTP client with progress reader, retries, and proxy
│   ├── geofabrik/               # Geofabrik index-v1.json client and region hierarchy model
│   ├── provider/                # Multi-provider catalog (Geofabrik, OSM France, BBBike)
│   └── setup/                   # Automated download, validation, and paths for JDK, mkgmap, splitter
│
├── frontend/                    # Frontend React SPA
│   ├── src/
│   │   ├── App.jsx              # UI state, queue, backend actions & Wails event orchestration
│   │   ├── AppleTheme.css       # Apple HIG CSS system (vibrancy, tokens, typography, dark/light)
│   │   ├── i18n.js              # Localization dictionaries (RU/EN)
│   │   ├── i18n.test.js         # Vitest i18n key parity tests
│   │   ├── components/          # Modularized React components
│   │   │   ├── common/          # CircularProgress, ToastNotification
│   │   │   ├── modals/          # DeleteConfirmationModal, UpdateModal, StorageCleanModal
│   │   │   ├── navigation/      # Sidebar, Toolbar
│   │   │   └── views/           # CatalogView, OutputsView, ConsoleView, LocalConvertView, ToolsView, SettingsView, AboutView
│   │   └── utils/               # Formatting, URL validation & queue scheduling helpers
│   │       ├── formatters.js    # formatBytes, formatDate, formatSpeed, formatEta, isValidExternalURL
│   │       ├── formatters.test.js
│   │       ├── queue.js         # Job queue management utilities
│   │       └── queue.test.js
│   ├── wailsjs/                 # Auto-generated Wails JavaScript bindings and runtime
│   ├── dist/                    # Compiled production frontend assets (embedded into Go binary)
│   └── package.json             # NPM dependencies (vite, react, lucide-react, vitest)
│
├── build/                       # Build assets & generated binaries
│   ├── bin/
│   │   ├── zimap.app            # macOS application bundle
│   │   └── zimap.exe            # Standalone Windows x86-64 executable
│   ├── darwin/                  # macOS Info.plist and icons
│   └── windows/                 # Windows manifest and .ico resources
│
├── output/                      # Default directory where compiled .img files are stored
└── tools/                       # Downloaded local tools (jdk/, mkgmap/, splitter/)
```

---

## 4. Development & Build Commands

Ensure Go and Wails CLI are accessible:
```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Running Backend Tests:
Always run Go tests without cache before and after changes:
```bash
go test -count=1 ./...
```

### Running Frontend Tests:
Run Vitest unit tests:
```bash
cd frontend && npm test
```

### Building the Frontend:
```bash
cd frontend && npm run build && cd ..
```

### Building macOS App:
```bash
export PATH="$(go env GOPATH)/bin:$PATH"
wails build
```
The output will be placed in `build/bin/ZiMap.app`.
To test launch:
```bash
open build/bin/ZiMap.app
```

### Cross-Compiling for Windows on macOS:
Requires `mingw-w64` (`brew install mingw-w64`):
```bash
export PATH="$(go env GOPATH)/bin:$PATH"
CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ wails build -platform windows/amd64
```
The output will be placed in `build/bin/ZiMap.exe`.

### Running in Dev Mode (Hot Reload):
```bash
export PATH="$(go env GOPATH)/bin:$PATH"
wails dev
```

---

## 5. Architectural Principles & Golden Rules

When making changes to this codebase, you **must adhere** to the following principles:

### Rule 1: Preserve CLI Backward Compatibility
The binary must function both as a desktop GUI and as a headless CLI tool:
- When invoked with CLI commands (`download`, `convert`, `update`), `main.go` **must dispatch directly** to `cli.ProcessCommand(os.Args[1:])`.
- Help arguments (`--help`, `-h`, `help`) are handled directly in `main.go` without starting the GUI.
- When invoked with zero arguments, it initializes and runs `wails.Run(...)`.
- **Never** introduce GUI dependencies into `internal/converter` or `internal/cli` that prevent headless command-line execution.

### Rule 2: Non-Blocking IPC & Real-Time Event Streaming
- Wails bindings in `app.go` (`ConvertURL`, `ConvertLocal`, `UpdateTools`) must **never block the main IPC thread** with long-running operations.
- Long jobs must launch in background goroutines (`go func() { ... }()`).
- **Current implementation caveat:** conversion and tool-update methods still perform their main work before returning, while update/device start methods launch background work. Read the architecture guide's execution-ownership notes before changing binding return semantics; this responsiveness rule describes the required design, not a guarantee that every current method already returns immediately.
- Progress, stages, and logs must be streamed to the frontend via Wails events:
  - `convert:stage` (`"download"` | `"splitter"` | `"mkgmap"` | `"compile"` | `"finalize"` | `"error"`; the frontend sets `"done"` on `convert:done`)
  - `convert:progress` (integer percent 0-100)
  - `convert:log` (stdout/stderr lines)
  - `convert:done` (emitted **only** upon true successful completion)
  - `convert:error` (emitted on failure)
  - `tools:log` (tools download & installation log)

### Rule 3: Error Isolation & Never Mask Failures
- If a download fails (e.g. HTTP 503, 404, connection reset), or if Splitter/Mkgmap fails, the pipeline must:
  1. Halt immediately.
  2. Emit `convert:error` with a descriptive message.
  3. **DO NOT** emit `convert:done` if an error occurred.
  4. Display a red error alert in the UI instead of checkmarks.

### Rule 4: Cross-Platform Filesystem Hygiene
- Always use `filepath.Join` and `filepath.FromSlash` rather than raw `/` or `\\`.
- Handle OS differences for system utilities (e.g. `open -R <path>` on macOS vs `explorer.exe /select,<path>` on Windows).
- Support both embedded tools in `tools/` and external system binaries cleanly.

### Rule 5: Apple Human Interface Guidelines (HIG) Aesthetics
- Retain the frosted glass vibrancy (`backdrop-filter: blur(25px) saturate(190%)`).
- Use `-apple-system, BlinkMacSystemFont, "SF Pro Text", "SF Pro Display"` typography.
- Use `"SF Mono", Menlo, Consolas` for log terminals and metrics.
- Keep window drag regions responsive using `--wails-draggable: drag` and `--wails-draggable: none`.

---

## 6. Project Status & Roadmap

Consult [`TODO.md`](TODO.md) for the active task backlog.

### ✅ Completed Tasks (50 of 53):
- **macOS Window Dragging & IPC (Task 1)**: Full `--wails-draggable: drag` implementation with clean 48px top strip and non-blocking IPC calls.
- **Catalog Caching (Task 2)**: Two-tier caching (in-memory + disk cache in `cache/geofabrik-index.json`, 24h TTL) with `ForceRefreshCatalog()`.
- **Accurate Tools & Java Detection (Task 3)**: `JavaDetails()` and `ToolDetails()` distinguish embedded `tools/jdk` from system Java in `PATH`, showing file paths and check animations.
- **Sidebar Rebranding (Task 4)**: Relocated branding (*ZiMap Downloader*, *by Ziviy*, `v1.0.0`, status dot) to the bottom of the sidebar.
- **"About" Tab (Task 5)**: Added *«О проекте»* screen with *FOR ALL TRAVEL & GPS LOVERS* banner, author GitHub link, and technology credits.
- **Settings & Localization (Task 6)**: Added Settings tab with real-time bilingual switching (RU/EN), theme toggle (Dark/Light), JVM heap selector, and localStorage persistence.
- **Recursive Catalog Search (Task 7)**: Traverses parent and descendant regions with duplicate suppression.
- **System Notifications (Task 8)**: Native cross-platform desktop notifications on map compile completion or error via AppleScript, PowerShell, and `notify-send`.
- **Strict Error Handling (Task 9)**: Guarded completion event to prevent false 100% success on HTTP/split errors; added Apple HIG error card in console.
- **Modernized Progress Bar (Task 10)**: Expanded to 18px pill squircle with meta stats header.
- **Real-Time Download Transfer Metrics (Task 11)**: Emits live speed (`MB/s`), downloaded/total size (`MB`), and remaining time (`ETA`).
- **Alternative Data Providers (Task 12)**: Unified provider architecture supporting Geofabrik, OpenStreetMap France mirror, and BBBike.org (240+ metropolitan cities worldwide) with Apple segmented control switching.
- **Proxy Configuration (Task 13)**: Support system proxy (`HTTP_PROXY`, `HTTPS_PROXY`) by default and configurable custom HTTP/SOCKS5 proxy settings in Settings.
- **Sidebar Reordering (Task 14)**: Reordered navigation: Catalog → My Maps → Converter → Console → Tools → Settings → About.
- **Context-Sensitive Search Bar (Task 15)**: Search bar rendered only on Catalog and My Maps tabs with dynamic view placeholders; `⌘K` scoped to active search.
- **Delete Map & One-Click Update (Task 22 & 16)**: Secure `DeleteOutput` backend method with path protection and companion `.meta.json` cleanup; Apple HIG confirmation sheet modal; one-click update button on each map card (fulfills Task 16).
- **Application Self-Updater (Task 17)**: GitHub Releases API check, semver comparison, update checking in Settings and About, full in-app update pipeline (`InstallUpdate`) with live progress/metrics, archive extraction, atomic app replacement, quarantine stripping, and one-click restart (`RestartApp`). Also includes simulation mode for instant UI verification.
- **Branding & Version (0.0.1 BETA)**: Globally set version to `0.0.1 BETA` across Go backend, Wails configuration, and UI components.
- **Branding Renaming (Task 18)**: Complete bundle, window title, metadata, and CLI help renaming to "ZiMap Downloader" (`com.ziviy.zimap`).
- **Catalog Tree Left-Alignment (Task 23)**: Added `.chevron-spacer` (14x14px, 6px margin) ensuring leaf nodes (e.g. Antarctica) align with pixel perfection.
- **Apple Micro-Animations (Task 24)**: Fluid view transitions (`@keyframes appleViewFadeSlide`), button `:active` scale(0.97), and smooth 90° tree chevron rotation.
- **Connection Retry & Timeout Resilience (Task 25)**: 90s header timeout, 3-attempt exponential backoff retry loop with live warning logs, fast fail on 404/403, and false success prevention.
- **Catalog Download Status Badge (Task 26)**: Apple HIG frosted green status badge and in-place "Обновить" ("Update") button for existing maps.
- **Standardized Download Label (Task 27)**: Standardized catalog action buttons to "Загрузить" / "Download".
- **Apple Circular Progress Ring (Task 28)**: In-place SVG circular progress indicator with animated stroke offset and central stop button in both Catalog and My Maps active download cards.
- **"My Maps" Tab (Task 29)**: Renamed "Ready Maps" to "My Maps" ("Мои карты") with live name filtering.
- **Background Map Downloading (Task 30)**: Downloads run in background without forcing redirection to console; active job indicated by green sidebar pulse.
- **Hierarchical Tree Search in Catalog (Task 31)**: Search preserves nested tree structure (collapsible countries and federal districts) with Cyrillic-to-English alias mapping (`россия`/`рф` -> `russia`, etc.) and individual download buttons.
- **Search Header Spacing (Task 32)**: Added 16px bottom margin and Apple HIG typography beneath "Результаты поиска".
- **Adaptive Progress Bar Contrast (Task 33)**: Dynamic contrast classes (`.uncovered` / `.covered`) ensuring text is crisp and readable in both Light and Dark themes.
- **Uniform Catalog Action Buttons & Icon Badge (Task 34)**: Standardized 28px height and 92px min-width for Download/Update buttons, and compact 28×28px icon-only squircle badge (`CheckCircle2`).
- **Sequential Multi-Map Download Queue (Task 35)**: Non-blocking download requests; queued maps show amber `.btn-queued` with Clock icon; dedicated "Очередь загрузки" in My Maps; automatic progression upon job completion.
- **Truncate Long Map Titles in My Maps (Task 36)**: Ellipsis truncation with hover tooltip on long map names, and `flex-shrink: 0` protection preventing button wrapping.
- **Clickable Progress for Console Navigation (Task 37)**: Click progress ring or active card to navigate directly to the build console, while isolating cancel click propagation.
- **Catalog Renaming & Startup Auto-Load (Task 38)**: Renamed section to «Каталог карт» / "Map Catalog"; immediate background catalog loading upon startup; default source selector in Settings.
- **Apple Squircle Icon (Task 21)**: Redesigned brand-neutral 1024x1024 app icon with contour lines, mountain peak, and compass needle (`.icns` for macOS, `.ico` for Windows).
- **Clean Temp & Cache Files (Task 48)**: Purge temporary files, build working directories, and cache with reclaimed space stats and modal sheet options.
- **Full-Bleed Squircle Icon (Task 40)**: Tight bounding box crop with superellipse curve ($n=4.4$) eliminating white borders in Dock and tray (`build/appicon.png`, `iconfile.icns`, `icon.ico`).
- **About Hero App Icon (Task 41)**: Replaced placeholder icon with crisp authentic ZiMap squircle app icon, smooth hover elevation and drop-shadow.
- **Check Tools Success Toast (Task 42)**: Apple HIG floating confirmation toast upon tools check validation.
- **Branding Refinement (Task 43)**: ZiMap as primary brand name, "Map Downloader" as descriptive subtitle across sidebar and about screens.
- **Save Map As Export Dialog (Task 47)**: Native Save/Export dialog to Finder/Explorer with streaming file copy in "Мои карты" and build completion card.
- **Fix Map Name Truncation & Compact Action Buttons (Task 49)**: Relocated `regionID` to metadata, platform-adaptive concise action buttons (`В Finder` / `In Finder`, `Экспорт` / `Export`), container queries ensuring full map titles never get crushed in "My Maps".
- **Dedicated GPS Device Tab (Task 19)**: Cross-platform volume detection (`/Volumes/*`, Windows drives, Linux mounts), XML manifest parsing (`GarminDevice.xml`), .img header internal title reading (offset 0x49), device storage meter bar, safe ejection, system maps protection (TopoActive, basemap, timezone blacklist), and MTP diagnostic guidance banner.
- **Map Freshness Check & Sync (Task 20)**: Real-time map freshness comparison against local builds with FAT32 2-second timestamp tolerance, 4-status classification, safe streaming copy with progress and cancellation, `fsync()` persistence, `os.Chtimes` timestamp synchronization, batch "Sync All" queue, and Apple HIG comparison cards.
- **Automated macOS MTP File Transfer (Task 51)**: Standalone CLI sidecar (`cmd/zimap-mtp`) powered by `ganeshrvel/go-mtpx`, non-blocking JSON event streaming, detection of MTP smartwatches (Fenix, Epix, Forerunner) on macOS, USB port conflict handling (Official Sync App), Apple HIG `.badge-mtp` and direct one-click map upload/sync.
- **GitHub Actions Release CI/CD (Task 52)**: Multi-platform automated matrix build (`macos-latest` arm64 & amd64 with embedded MTP sidecar, `windows-2022` amd64), static `libusb` zero-dependency bundling, asset naming parity with `scoreAsset` in `app_updater.go`, ditto bundle preservation, and GitHub Releases publishing.
- **Fresh-Install Output Directory Fix (Task 53)**: `ConvertURL`, `runConversion`/`ConvertLocal`, and `saveCompanionMetadata` create `output/` via `os.MkdirAll` before `os.MkdirTemp`/meta writes, eliminating `GetFileAttributesEx ... cannot find the file specified` on fresh standalone installs; nil Wails context tolerated in convert jobs; regression tests added in `app_test.go`.
- **Release Asset Renaming & Version Bump (0.0.2 BETA)**: Release assets renamed `ZiMap-darwin-*` → `ZiMap-macos-*` (CI workflow, Makefile, docs; updater-compatible naming), release workflow security hardening (scoped permissions, env-var template-injection guards, `persist-credentials: false`), and version bump to `0.0.2 BETA`. The macOS notification icon (Task 39) remains in rework — native `NSUserNotification` attempt still shows the wrong banner icon.

### 🎯 Active Backlog (3 Tasks Remaining):
- **Notification App Icon (Task 39)**: Native bundle icon association in macOS notification banners [REWORK NEEDED — NSUserNotification attempt in `app_notify_darwin.go` still shows the wrong icon; try UNUserNotificationCenter, a bundled Swift helper, or terminal-notifier; verify from a real built `.app`].
- **Output Format Selector (Task 44)**: Support multiple output formats (.img, BaseCamp/MapSource .gmapi).
- **Self-Contained Storage & Packaging (Task 46)**: Standalone packaging without source code dependencies, hidden internal map storage.





