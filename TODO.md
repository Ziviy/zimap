# ZiMap Downloader — Task Backlog & Implementation Roadmap

This document tracks planned improvements, bug fixes, and feature additions for the **ZiMap Downloader** application. 

**Progress Tracker**:
- **Total Tasks**: 52
- **Completed**: 49 (`#1`, `#2`, `#3`, `#4`, `#5`, `#6`, `#7`, `#8`, `#9`, `#10`, `#11`, `#12`, `#13`, `#14`, `#15`, `#16`, `#17`, `#18`, `#19`, `#20`, `#21`, `#22`, `#23`, `#24`, `#25`, `#26`, `#27`, `#28`, `#29`, `#30`, `#31`, `#32`, `#33`, `#34`, `#35`, `#36`, `#37`, `#38`, `#40`, `#41`, `#42`, `#43`, `#45`, `#47`, `#48`, `#49`, `#50`, `#51`, `#52`)
- **Active / Pending**: 3 (`#39`, `#44`, `#46`)

---

## 🎯 Active Backlog (Pending Tasks)

### 39. Application Icon in System Notifications
- **Issue**: macOS notification banners currently display the generic Script Editor / AppleScript paper icon instead of the ZiMap app icon.
- **Cause**: Invoking notifications via `osascript` causes macOS to attribute the notification to `/usr/bin/osascript`.
- **Solution**:
  - Pass the application bundle identifier `com.ziviy.zimap` or use macOS JXA / notification helper with proper bundle icon association.
  - Alternatively, pass the icon path directly to Web Notifications or native notification dispatcher.
- **Affected Files**:
  - `app.go`
  - `frontend/src/App.jsx`

---

### 44. Multiple Output Map Format Selector
- **User Requirement**: Add selection of output map file formats (assess feasibility and complexity).
- **Technical Analysis & Complexity**:
  - **Within GPS navigator ecosystem (Low / Medium Complexity)**:
    - `.img` (default standalone device image, ready for GPS navigators and watches).
    - `.gmapi` (Mac MapSource / BaseCamp format for route planning on macOS).
    - `.tdb` + tile `.img` files + `.reg` (BaseCamp / MapSource installer format on Windows).
    - *Feasibility*: Mkgmap can generate `.gmapi` and `.tdb` natively with flags `--gmapi` and `--tdbfile`.
  - **Across Other GPS Applications (High Complexity)**:
    - OsmAnd `.obf` (requires integrating `OsmAndMapCreator` Java pipeline).
    - Mapsforge `.map` for Locus Map / Cruiser (requires integrating `osmosis` + `mapsforge-map-writer` plugin).
    - Vector MBTiles / PMTiles (requires `tilemaker` or `planetiler`).
- **Solution**:
  - Implement a format selector dropdown/segmented control in Converter and Settings tabs starting with standard `.img` and BaseCamp/MapSource `.gmapi` bundle formats, with architecture prepared for additional format plugins.
- **Affected Files**:
  - `internal/converter/convert.go`
  - `app.go`
  - `frontend/src/App.jsx`
  - `frontend/src/i18n.js`

---

### 46. Self-Contained Storage & Portable Application Packaging
- **User Requirement**: Package the application into a single standalone container/binary without needing external source code (`go.mod`, git repository, etc.) for both macOS and Windows (portable `.exe`). The application should keep compiled maps hidden in its own internal private storage.
- **Technical Analysis & Complexity**:
  - **Complexity: 🟢 Medium (3-4 hours)**.
  - **Paths & Storage Isolation**:
    - Current code in `internal/setup/paths.go` looks for `go.mod` in parent directories. In standalone `.app` on macOS (e.g. in `/Applications`) or standalone `.exe` on Windows, `go.mod` does not exist.
    - Solution: Update `ProjectRoot()` / `DataDirectory()`:
      - **macOS**: Use standard `~/Library/Application Support/ZiMap/` (`maps/`, `cache/`, `tools/`).
      - **Windows**: Use `%LOCALAPPDATA%\ZiMap\` (or portable mode: subfolder next to executable if writeable).
      - Maps are stored in `<DataDir>/maps/` rather than a loose `output/` folder in user directories, keeping them safely managed inside the app.
  - **Packaging**:
    - **macOS**: Standalone `ZiMap.app` bundle (self-contained Go binary with embedded frontend dist, can be packaged into DMG or zip).
    - **Windows**: Standalone portable `ZiMap.exe` that runs directly without an installer, storing cache and downloaded tools in `AppData` or alongside.
- **Affected Files**:
  - `internal/setup/paths.go`
  - `internal/setup/tools.go`
  - `app.go`
  - `wails.json`

---

## ✅ Completed Tasks & Changelog

### 52. Automated CI/CD Release Pipeline via GitHub Actions & In-App Auto-Update Integration [COMPLETED]
- **Implementation Plan**: 📄 [Подробный архитектурный план реализации (docs/plans/task-52-github-actions-release.md)](docs/plans/task-52-github-actions-release.md)
- **User Requirement**: Build and publish releases to GitHub automatically upon tag creation so the application can self-update seamlessly across macOS and Windows.
- **Resolution**:
  - **Workflow Configuration (`.github/workflows/release.yml`)**:
    - Multi-platform matrix build for macOS (`arm64` and `amd64`) on `macos-latest` and Windows (`amd64`) on `windows-2022`.
    - Triggers on tag pushes matching `v*` and manual `workflow_dispatch`.
    - Permissions set to `contents: write` for automated release publication.
  - **Sidecar Integration & Cross-Compilation (`scripts/build-mtp.sh`)**:
    - Enhanced `scripts/build-mtp.sh` to support cross-compilation (`arm64`, `amd64`) and static linking of `libusb` on macOS.
    - Zero external runtime dylib dependencies on end-user machines: statically links `libusb-1.0.a` with macOS system frameworks (`IOKit`, `CoreFoundation`, `Security`, `libobjc`).
    - Embeds `zimap-mtp` binary directly into `Contents/MacOS/` inside the macOS application bundle.
  - **Asset Naming & Updater Parity**:
    - Exact matching with `scoreAsset` in `app_updater.go`: produces `ZiMap-darwin-arm64.zip`, `ZiMap-darwin-amd64.zip`, and `ZiMap-windows-amd64.zip`.
    - Preserves macOS permissions and bundle attributes using `/usr/bin/ditto -c -k --keepParent`.
  - **Release Publishing**:
    - Aggregates all matrix artifacts via `actions/download-artifact@v4`.
    - Publishes release via `softprops/action-gh-release@v2` with `draft: false`, `prerelease: false`, and automatic changelog notes generation.
  - **Local Packaging Target (`Makefile`)**:
    - Added `make release-local` target to locally build frontend, sidecar, desktop app, and ditto zip bundle in seconds.
- **Files Modified**: `.github/workflows/release.yml`, `scripts/build-mtp.sh`, `Makefile`, `TODO.md`, `AGENTS.md`.

---

### 51. Automated MTP File Transfer for GPS Smartwatches on macOS (Sidecar Architecture) [COMPLETED]
- **Implementation Plan**: 📄 [Подробный архитектурный план реализации (docs/plans/task-51-macos-mtp-sync.md)](docs/plans/task-51-macos-mtp-sync.md)
- **Issue**: Modern GPS smartwatches (Fenix 5 Plus, 6, 7, 8, Epix 2, Enduro, Forerunner 255/265/955/965) only support MTP mode and cannot mount as storage volumes under `/Volumes/` on macOS. Users previously had to manually export `.img` files and use external tools like OpenMTP or Android File Transfer.
- **Resolution**:
  - **CLI Sidecar Binary (`cmd/zimap-mtp`)**:
    - Developed standalone helper in `cmd/zimap-mtp/main.go` using `ganeshrvel/go-mtpx` and `ganeshrvel/go-mtpfs/mtp`.
    - `detect`: Checks USB bus for GPS MTP devices (`0x091E`), opens session, queries primary storage capacity and free space.
    - `list-maps`: Walks `/GARMIN` (and `/GARMIN/Map`), parses `.img` headers at offset 0x49 via partial object reads (`dev.GetPartialObject`) to extract internal map names.
    - `upload`: Streams `.img` files to device with real-time newline-delimited JSON progress events (sent bytes, total, percent, speed in bytes/sec, ETA in seconds). Graceful cancellation handling via SIGINT/SIGTERM.
    - `delete`: Deletes maps on device via MTP `DeleteObject` with strict system map protection rejecting factory basemaps.
    - `check-conflicts`: Detects processes locking the USB interface (`Official Sync App`, `Android File Transfer`, `OpenMTP`).
    - Full mock mode (`--mock` flag and `ZIMAP_MOCK_GARMIN_MTP=1`) for headless testing.
  - **Backend Orchestrator & Detector (`internal/device`)**:
    - `mtp_service.go`: Resolves `zimap-mtp` binary across multiple candidate locations, manages sub-processes with `exec.CommandContext`, parses streaming progress.
    - `mtp_detector.go`: Extends `ScanDevices()` to detect connected MTP watches, constructing full `GarminDevice` structures with `IsMTP: true`, live storage metrics, and custom maps list.
    - `detector.go`: Updated `InspectGarminMount` to seamlessly handle `mtp://` mount paths.
    - `sync.go`: Extended `SyncManager` with `SyncMTP` and `StartSyncMTP`, integrating with unified busy state and cancellation.
  - **Wails IPC Bridge (`app_device.go`)**:
    - `SyncMapToDevice`, `SyncAllOutdatedMaps`, `CheckDeviceFreshness`, `DeleteDeviceMap`, and `EjectGarminDevice` fully routed for MTP watches.
    - Emits standard `device:sync_start`, `device:sync_progress`, `device:sync_done`, `device:sync_error`, `device:sync_cancelled` events.
  - **Apple HIG Frontend UI (`DeviceView.jsx`, `AppleTheme.css`, `i18n.js`)**:
    - Displays frosted `.badge-mtp` badge in hero card for MTP smartwatches.
    - Direct activation of «Загрузить на устройство» and «Синхронизировать всё» for MTP watches.
    - Conflict warning banner when `Official Sync App` is running with advice to quit it.
    - Contextual action buttons: Reveal button hidden for MTP (not present in Finder), Eject button shows «Отключить» («Disconnect»).
    - Full RU/EN localization with 100% key parity verified by unit tests.
  - **Build Automation & Packaging (`Makefile`, `scripts/build-mtp.sh`)**:
    - Build script and Makefile targets automating compilation of `zimap-mtp` into `tools/mtp/` and embedding into `ZiMap.app/Contents/MacOS/`.
- **Files Modified**: `cmd/zimap-mtp/main.go`, `internal/device/device.go`, `internal/device/detector.go`, `internal/device/usb_diagnostics.go`, `internal/device/sync.go`, `internal/device/mtp_service.go`, `internal/device/mtp_detector.go`, `internal/device/mtp_service_test.go`, `app.go`, `app_device.go`, `app_device_test.go`, `frontend/src/components/views/DeviceView.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`, `frontend/wailsjs/go/models.ts`, `Makefile`, `scripts/build-mtp.sh`, `TODO.md`, `AGENTS.md`.

### 20. Map Freshness Check & Sync for Connected GPS Device [COMPLETED]
- **Implementation Plan**: 📄 [Подробный архитектурный план реализации (docs/plans/task-20-device-sync.md)](docs/plans/task-20-device-sync.md)
- **Issue**: Users needed an automated way to verify whether custom maps installed on their GPS devices were up to date compared to fresh builds in the computer's `output/` directory, and a reliable, safe one-click mechanism to transfer updated or missing maps without manual file operations or risk of FAT32 corruption.
- **Resolution**:
  - **Freshness Engine & Timestamp Tolerance (`internal/device/freshness.go`)**:
    - 3-tier map correlation: exact case-insensitive filename matching, RegionID/clean title correlation, and `gmapsupp.img` internal header name resolution.
    - FAT32 2-second timestamp tolerance ($\Delta t \le 2\,\text{s}$) to avoid false "outdated" detections on FAT32/exFAT filesystems.
    - 4-state classification: `StatusUpToDate`, `StatusUpdateAvailable`, `StatusNotOnDevice`, `StatusOnlyOnDevice`.
    - Space pre-flight capacity validation accounting for a 20 MB safety margin for GPS device FIT/GPX logs.
  - **Safe Streaming Transfer Engine (`internal/device/sync.go`)**:
    - 128 KB buffered streaming copy with real-time speed calculation, percentage, and ETA computation.
    - Fault-tolerant staging to temporary file (`<target>.tmp-zimap`), flushed to hardware flash memory with `file.Sync()`, followed by atomic replacement.
    - Preserves local build modification times via `os.Chtimes(destPath, srcModTime, srcModTime)` so FAT32 timestamps match the computer build.
    - Cancellation support via `context.WithCancel`, safely cleaning up temporary `.tmp-zimap` files upon interruption.
    - Thread-safe `SyncManager` guarding against concurrent transfers and providing non-blocking execution.
  - **Backend IPC (`app_device.go`)**:
    - Exported methods `CheckDeviceFreshness`, `SyncMapToDevice`, `SyncAllOutdatedMaps`, and `CancelDeviceSync`.
    - Non-blocking streaming events: `device:sync_start`, `device:sync_progress`, `device:sync_done`, `device:sync_all_done`, `device:sync_error`, `device:sync_cancelled`.
  - **Apple HIG Frontend UI (`DeviceView.jsx` & `AppleTheme.css`)**:
    - Freshness summary badge bar (🟢 Up to date, 🟠 Update available, 🔵 Not on device, ⚪ Only on device).
    - Batch sync button **«Синхронизировать всё»** with count of pending maps.
    - Apple segmented control filtering (All, Updates, On Device, Missing).
    - Dual-column comparison cards displaying computer vs device version dates, sizes, relative outdated days, and status badges.
    - In-place real-time streaming transfer progress bar with speed, percent, ETA, and cancellation button.
    - Full RU/EN localization with 100% key parity in `i18n.js`.
- **Files Modified**: `internal/device/sync_types.go`, `internal/device/freshness.go`, `internal/device/freshness_test.go`, `internal/device/sync.go`, `internal/device/sync_test.go`, `app.go`, `app_device.go`, `app_device_test.go`, `frontend/src/components/views/DeviceView.jsx`, `frontend/src/App.jsx`, `frontend/src/i18n.js`, `frontend/src/AppleTheme.css`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/models.ts`.

---

### 19. Dedicated "GPS Device" Tab (Connected Watch & Custom Maps Explorer) [COMPLETED]
- **Implementation Plan**: 📄 [Подробный архитектурный план реализации (docs/plans/task-19-gps-device.md)](docs/plans/task-19-gps-device.md)
- **Issue**: Users needed a dedicated interface to inspect and manage custom maps on physically connected GPS devices (Fenix, Epix, Forerunner, Enduro, Edge, GPSMAP) directly without having to navigate internal directories manually, while strictly preventing accidental deletion of vital factory maps (TopoActive, basemap, timezones, DEM).
- **Resolution**:
  - **Volume & Device Scanner (`internal/device`)**:
    - Automatic detection across macOS (`/Volumes/*`), Windows (`C:\` - `Z:\`), and Linux (`/media`, `/run/media`, `/mnt`) by checking for `GARMIN/` or `GarminDevice.xml`.
    - Parses `GARMIN/GarminDevice.xml` schema for device model description, display name, unit ID, and software version.
    - Measures volume capacity, free space, and used space via `syscall.Statfs` on Unix and `GetDiskFreeSpaceExW` on Windows.
    - Inspects binary headers of `.img` files (offset 0x49) to extract internal map descriptions (e.g. set by mkgmap) for clean display names.
  - **Security Guardrails & Map Classification**:
    - Strict system blacklist (`gmapbmap`, `gmaptz`, `gmapdem`, `gmapprom`, `gmapgc`, `gmap3d`, and regex `^d\d{7}[a-z0-9]?\.img$`).
    - Validates paths against traversal attacks, ensuring target is an `.img` file strictly inside `<mount>/GARMIN/` or `<mount>/GARMIN/Map/`.
  - **Safe Eject & Diagnostics**:
    - Cross-platform safe volume ejection (`diskutil eject` / `diskutil unmountDisk` on macOS, Windows Shell Application COM on Windows, `umount` on Linux).
    - USB hardware diagnostic fallback checking GPS Vendor ID `0x091E` to inform macOS users if watches are in MTP mode with instructions for Mass Storage mode.
  - **Apple HIG Frontend UI**:
    - Added "GPS-устройство" / "GPS Device" tab with Lucide `Watch` icon and green online dot indicator in Sidebar.
    - Hero device card displaying model name, firmware version, Unit ID, and mount path with "Reveal in Finder/Explorer" and "Eject" actions.
    - Dual-contrast Apple HIG storage capacity progress bar with GB stats and legend.
    - Custom maps list showing internal map names, file size, relative folder, and modification date.
    - Apple HIG confirmation sheet modal (`DeleteDeviceMapModal`) for permanent map deletion.
    - Full bilingual localization (RU/EN) with 100% key parity in `i18n.js`.
- **Files Modified**: `internal/device/device.go`, `internal/device/storage.go`, `internal/device/storage_unix.go`, `internal/device/storage_windows.go`, `internal/device/img_parser.go`, `internal/device/filter.go`, `internal/device/detector.go`, `internal/device/detector_darwin.go`, `internal/device/detector_windows.go`, `internal/device/detector_linux.go`, `internal/device/usb_diagnostics.go`, `internal/device/device_test.go`, `app_device.go`, `app_device_test.go`, `frontend/src/components/views/DeviceView.jsx`, `frontend/src/components/modals/DeleteDeviceMapModal.jsx`, `frontend/src/components/navigation/Sidebar.jsx`, `frontend/src/App.jsx`, `frontend/src/i18n.js`, `frontend/src/AppleTheme.css`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/models.ts`.

---

### 49. Fix Map Name Truncation & Compact Action Buttons in "My Maps" [COMPLETED]
- **Issue**: Compiled map titles in the "My Maps" tab were being severely truncated down to 1–2 characters (e.g. "Crimean Federal District" displayed only as `(` and "Kaliningrad" as `Kalin...`), making cards unreadable.
- **Root Cause**:
  1. The four action buttons in `.output-actions` occupied over 550px of horizontal space due to verbose button labels (`buttons.reveal: "Показать в Finder / Проводнике"`, 30 chars, and `buttons.export: "Сохранить как..."`).
  2. The raw `regionID` slug (e.g. `crimean-fed-district`, ~140px wide) was rendered directly inside `.output-title-row` with a non-shrinking `.stage-pill` (`flex-shrink: 0`), competing directly with the map title.
  3. Combined with `flex-shrink: 0` on `.output-actions` and strict `overflow: hidden; text-overflow: ellipsis;` on `.output-title`, only 10–20px remained for the actual map title.
- **Resolution**:
  - **Clean Title Row**: Relocated `item.meta?.regionID` from `.output-title-row` to `.output-meta` as a secondary monospace slug (`• crimean-fed-district`), leaving the entire title line for the map name and `.IMG` badge, freeing ~140px.
  - **Platform-Adaptive Concise Buttons**:
    - Modernized the reveal button to be platform-aware via `getRevealInfo(t)`: `"В Finder"` / `"In Finder"` on macOS, `"В проводнике"` / `"In Explorer"` on Windows, and `"Показать в папке"` / `"Show in Folder"` elsewhere, with full descriptive tooltips on hover.
    - Streamlined export button to `"Экспорт"` / `"Export"` (with tooltip), reducing total `.output-actions` width from ~550px down to ~290px (saving ~260px).
  - **Container Query Responsiveness**: Added CSS container query (`container-type: inline-size` on `.card-list` and `@container (max-width: 560px)`) to automatically switch to stacked layout with full-width title on narrow windows or split view, guaranteeing titles never get crushed.
- **Files Modified**: `frontend/src/i18n.js`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `TODO.md`, `AGENTS.md`.

---

### 45. Multiple Input Formats & Map Detail Level Configuration [COMPLETED]
- **Issue**: The pipeline only accepted `.osm.pbf` input files with fixed mkgmap detail settings, and BBBike's pre-compiled .img map styles were not exposed.
- **Resolution**:
  - **Input Formats**: `converter.IsSupportedInput` now accepts `.osm.pbf`, `.pbf`, `.osm`, `.osm.gz`, and `.osm.bz2` — all natively readable by `splitter.jar` and `mkgmap.jar`. Unsupported extensions are rejected early with a descriptive error (unit-tested).
  - **Detail Level Presets**: Added `converter.DetailLevel` (`standard` / `high` / `topo`) with `NormalizeDetailLevel` validation mapping onto mkgmap flags: `high` raises zoom levels (`--levels=0:24,1:23,2:21,3:19`, `--draw-priority=25`); `topo` is an outdoor profile adding `--transparent --route --road-name-pois` for hiking devices. `standard` preserves the splitter template (previous behaviour).
  - **CLI**: `zimap convert <file> [--detail=standard|high|topo]` with validation, extended help text, and extension-aware output naming.
  - **GUI (Converter)**: Apple segmented detail-level control in the PBF Converter tab (persisted to `localStorage`), file dialog and dropzone now accept all supported formats.
  - **GUI (Settings)**: "Map Detail Level" selector defining the default preset for all catalog downloads and one-click updates; stored in `<name>.meta.json` (`detailLevel` field).
  - **BBBike Styles**: `provider.BBBikeGarminStyles` models the pre-compiled styles (Onroad, Ontrail, OpenTopoMap, OSM UTF-8) with `BuildBBBikeStyleURL` / `DetectBBBikeStyleURL` URL resolution and alias support; new Wails method `App.BuildBBBikeStyleURL` and a segmented style picker in the Catalog when BBBike is the active provider. Pre-compiled styles stream down as ZIP, extract `gmapsupp.img`, and publish directly without local compilation (`downloadPrecompiledBBBike`), with full stage/progress/log streaming.
  - Fixed a latent URL bug caught by the new tests: `BuildBBBikeStyleURL` produced a doubled `.osm` segment (`Moscow.osm.osm.garmin-...`).
  - Fixed pre-existing React rules-of-hooks violations in `DeleteConfirmationModal` and `UpdateModal` (hooks called after conditional early-return), and hardened external URL opening with protocol-validated `openExternalURL` (blocking `javascript:`/`data:`/`file:` injection).
- **Files Modified**: `internal/converter/convert.go`, `internal/converter/convert_test.go`, `internal/cli/convert.go`, `internal/provider/provider.go`, `internal/provider/provider_test.go`, `app.go`, `app_test.go`, `frontend/src/App.jsx`, `frontend/src/i18n.js`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/main/App.d.ts`.

### 40. Full-Bleed Squircle Icon Without White/Blank Borders [COMPLETED]
- **Issue**: The Dock/tray icon had visible white/empty padding around the squircle on macOS due to extra canvas margins in the source asset.
- **Resolution**:
  - Cropped the squircle tightly to its content bounding box (centered at 510.5, 511.5, size 732×732) and scaled it smoothly with bilinear interpolation to the standard Apple HIG 1024×1024 canvas.
  - Applied an authentic Apple continuous superellipse squircle curve mask ($n=4.4$) with subpixel antialiasing and crisp transparent backdrop.
  - Re-generated `build/appicon.png` (1024×1024 RGBA), `build/darwin/iconfile.icns` via macOS `iconutil` covering all 10 Retina sizes (16×16 to 1024×1024), and `build/windows/icon.ico` covering all 6 standard resolutions (16 to 256 px).
  - Copied icon directly to `frontend/public/appicon.png` and `frontend/src/assets/appicon.png`.
- **Files Modified**: `build/appicon.png`, `build/darwin/iconfile.icns`, `build/windows/icon.ico`, `frontend/public/appicon.png`, `frontend/src/assets/appicon.png`.

---

### 41. Real App Icon in "About" (О проекте) Hero Banner [COMPLETED]
- **Issue**: The "About" screen displayed a generic Lucide vector icon in its hero banner instead of the actual ZiMap brand icon.
- **Resolution**:
  - Replaced the placeholder icon with a crisp, high-resolution rendering of the new ZiMap squircle app icon (`appicon.png`).
  - Added authentic Apple HIG squircle styling with smooth hover elevation (`scale(1.03)`, translateY(-2px)), frosted glass drop-shadow, and subtle typography hierarchy.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 42. Success Toast Banner on "Check Tools" Button [COMPLETED]
- **Issue**: Clicking "Проверить" in the Tools tab updated the status badges but lacked explicit, satisfying confirmation feedback when all tools are ready.
- **Resolution**:
  - Added Apple HIG frosted floating toast/pill notification (*«Все инструменты установлены и готовы к работе!»* / *«All tools are installed and ready to work!»*) when tools check succeeds.
  - Added warning toast if tools are missing, advising the user to click "Обновить / Восстановить всё".
  - Kept initial background startup validation silent.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/i18n.js`.

---

### 43. Sidebar & About Branding Refinement: "ZiMap" + "Map Downloader" [COMPLETED]
- **Issue**: Branding distinction needed clear separation: **ZiMap** as primary brand/app name, and **Map Downloader** as descriptive subtitle.
- **Resolution**:
  - Sidebar bottom: App title **ZiMap**, subtitle **Map Downloader**, with tooltip "ZiMap Map Downloader".
  - "About" tab: Title **ZiMap**, subtitle **Map Downloader**, with description highlighting map downloading and compilation capabilities.
  - Updated bilingual dictionary mappings in `i18n.js`.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/i18n.js`, `frontend/src/AppleTheme.css`.

---

### 47. Native "Save Map As..." Export Dialog & Direct Transfer Architecture [COMPLETED]
- **Issue**: Users needed an easy way to export compiled maps directly to any folder/drive via native Finder / Explorer save dialogs.
- **Resolution**:
  - Implemented `ExportMap(path string) (string, error)` in `app.go` with security validation (preventing path traversal, ensuring source file is within output directory, resolving missing `.img` extension, and preventing in-place truncation via `os.SameFile`).
  - Prompts native OS Save File dialog via `wailsruntime.SaveFileDialog` with default filename `<name>.img` and `.img` filter.
  - Added `copyFile(src, dst string) error` streaming copy with `fsync`.
  - Added "Сохранить как..." / "Save As..." action button (`Download` icon) to each map card in "Мои карты" (`OutputsView`).
  - Added success completion card with both Export and Reveal actions in `ConsoleView` when map build finishes (`stage === 'done'`), preserving `completedJob` across build lifecycle.
  - Added full bilingual localization for `LocalConvertView` and `OutputsView` empty states in `i18n.js`.
  - Expanded `SelectLocalPBFDialog` and dropzone to support `.osm`, `.osm.gz`, and `.osm.bz2` alongside `.osm.pbf`.
  - Added ellipsis overflow protection to `.toast-pill` for long file paths and added native bundle icon to web notifications.
  - Added unit tests in `app_test.go` verifying file copying, security boundaries, error validation, relative path reveal, and dialog context checks.
- **Files Modified**: `app.go`, `app_test.go`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`.

### 17. In-App Application Self-Update & Direct UI Installation [COMPLETED]
- **Issue**: Users needed an easy way to check for, download, and install newer versions of ZiMap directly inside the application UI, and the version needed to be reset from 1.0.0 to `0.0.1 BETA`.
- **Resolution**:
  - Changed application version globally to `0.0.1 BETA` across Go backend (`app.go`), Wails configuration (`wails.json`), and dynamic UI state.
  - Enhanced `CheckForUpdate() (*UpdateInfo, error)` with platform-aware asset scoring (`findMatchingAsset`) for macOS (Apple Silicon arm64 / Intel amd64), Windows (x64), and Linux.
  - Added non-blocking in-app update pipeline `InstallUpdate(assetURL string) error` streaming events (`update:stage`, `update:progress`, `update:metrics`, `update:done`, `update:error`).
  - Added archive extraction (`ditto -xk` on macOS, Go tar/zip on other platforms) with atomic bundle/binary replacement and quarantine attribute stripping.
  - Added `RestartApp() error` to relaunch the updated application (`open -n <appBundle>` on macOS, direct execution on Windows/Linux) and exit cleanly.
  - Added `CancelUpdate() error` to safely cancel active downloads and clean up temporary workspaces.
  - Added `SimulateUpdateCheck()` and simulated download/install mode accessible via "Тест обновления" button in Settings for instant UI testing.
  - Redesigned Apple HIG `UpdateModal` dialog with animated progress bar, live transfer speed/ETA metrics, ready screen with restart button, and retry flow.
- **Files Modified**: `app.go`, `app_updater.go`, `app_tools.go`, `app_test.go`, `wails.json`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`, `frontend/src/components/modals/UpdateModal.jsx`, `frontend/src/components/views/SettingsView.jsx`, `frontend/src/components/views/AboutView.jsx`, `frontend/src/components/navigation/Sidebar.jsx`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/models.ts`.

---

### 48. Clean Temporary, Working & Cache Files [COMPLETED]
- **Issue**: Build artifacts, Splitter tile workspaces (`.splitter`), temporary download chunks (`.download-*`), tool update workdirs (`.mkgmap-update-*`), and cached indexes accumulated space without a way to purge them.
- **Resolution**:
  - Added `CacheDirectory()` and `ToolsDirectory()` helpers in `internal/setup/paths.go`.
  - Implemented `GetStorageMetrics() (StorageMetrics, error)` in `app.go` measuring live storage usage across cache, temporary files, and compiled maps.
  - Implemented `ClearTempAndCacheFiles(clearCache bool, clearTemp bool) (int64, error)` in `app.go` with safety guard against deleting files while a build job is running, while strictly preserving `proxy-config.json` and compiled `.img` maps.
  - Added unit tests in `app_test.go` verifying metric calculations, recursive purging, and configuration/map preservation.
  - Added dedicated **"Хранилище и кэш"** ("Storage & Cache") section in Settings with metrics grid and recalculate button.
  - Added Apple HIG `StorageCleanModal` with granular options to purge catalog cache and/or temporary build files.
  - Emitted Apple HIG floating confirmation toast notifying the user of reclaimed disk space (e.g., *"Освобождено 450 МБ"*).
- **Files Modified**: `internal/setup/paths.go`, `app.go`, `app_test.go`, `frontend/src/App.jsx`, `frontend/src/i18n.js`, `frontend/src/AppleTheme.css`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/models.ts`.

---

### 1. Fix macOS Window Freezing & Dragging Lag [COMPLETED]
- **Issue**: On macOS, the window occasionally hangs/freezes and refuses to be dragged across the screen.
- **Resolution**:
  - Implemented Wails native `--wails-draggable: drag` and `-webkit-app-region: drag` for `.sidebar-drag-region` (height 48px) and `.toolbar`.
  - Applied `--wails-draggable: none` and `-webkit-app-region: no-drag` to interactive elements (`.sidebar-item`, buttons, inputs, `.search-bar`).
  - Switched catalog fetching and searching to non-blocking memory/disk cache.
- **Files Modified**: `frontend/src/AppleTheme.css`, `frontend/src/App.jsx`, `app.go`.

---

### 2. Cache Geofabrik Catalog State [COMPLETED]
- **Issue**: Re-fetching the full Geofabrik `index-v1.json` on every tab change caused slow UI and unnecessary network requests.
- **Resolution**:
  - Added 24-hour disk cache in `cache/geofabrik-index.json`.
  - Added in-memory cache with `RWMutex` to `App` struct in Go.
  - Lifted catalog state in `App.jsx` so tab switching preserves data without re-requesting.
  - Implemented `ForceRefreshCatalog()` for explicit manual refresh via the "Обновить" button.
- **Files Modified**: `app.go`, `frontend/src/App.jsx`.

---

### 3. Fix "Check Tools" Button (Local vs System Tools Detection) [COMPLETED]
- **Issue**: Deleting `tools/jdk` still reported Java as installed without visual feedback because system Java in `PATH` satisfied `exec.LookPath`.
- **Resolution**:
  - Added `JavaDetails()` and `ToolDetails()` in `internal/setup/tools.go` to explicitly distinguish between embedded tools in `tools/` and external system binaries.
  - Updated `ToolsStatus` in `app.go` to return detailed metadata (`installed`, `embedded`, `path`).
  - Displayed explicit status badges (`"Встроенный (tools/)"`, `"Системный (PATH)"`, `"Не установлен"`) and file paths in monospace font (`SF Mono`).
  - Added visual loading spinner feedback to the "Проверить" button.
- **Files Modified**: `internal/setup/tools.go`, `app.go`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 4. Improve Sidebar Branding & Move Logo/Title Down [COMPLETED]
- **Issue**: Branding under macOS window traffic lights was cramped and visually awkward.
- **Resolution**:
  - Converted the top of the sidebar into a clean 48px window drag zone with generous breathing room.
  - Relocated branding to the bottom of the sidebar (`.sidebar-footer-branding`):
    - App title: **ZiMap**
    - Subtitle: **Map Downloader**
    - Version: **v1.0.0**
    - Author: **by Ziviy**
    - Ready indicator dot with text.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 5. Add "About" Tab ("О проекте") [COMPLETED]
- **Issue**: Missing "About" screen with author attribution, version, and mission statement.
- **Resolution**:
  - Added **"О проекте"** tab with `Info` icon in the navigation.
  - Displayed large compass emblem, app title, version `v1.0.0 (Build 2026.1)`.
  - Added **"Автор: by Ziviy"** button opening [https://github.com/Ziviy](https://github.com/Ziviy) via `App.OpenURL` (Wails browser runtime).
  - Added prominent banner: **"FOR ALL TRAVEL & GPS LOVERS"** with project description.
  - Included technology credits grid (*OpenStreetMap, Geofabrik, Mkgmap, Splitter, Wails v2, React*).
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `app.go`.

---

### 6. Add "Settings" Tab ("Настройки") with Language Selector [COMPLETED]
- **Issue**: Need dedicated settings screen with language switcher and configuration options.
- **Resolution**:
  - Created `frontend/src/i18n.js` with comprehensive English and Russian dictionary mappings for all screens and components.
  - Built `SettingsView` component with:
    - Segmented language switcher: **Русский** / **English**.
    - Theme switcher: **Светлая** / **Темная**.
    - JVM Default Heap Size selector (2G, 4G, 8G, 16G, 32G).
    - Application info card with version and author.
  - Settings are persisted in `localStorage` (`zimap_language`, `zimap_theme`, `zimap_heap`).
- **Files Modified**: `frontend/src/i18n.js`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 7. Recursive Catalog Search (Show Parent & Subregions) [COMPLETED]
- **Issue**: Searching for "russia" only displayed the top-level Russia region, omitting all internal federal districts and subregions.
- **Resolution**:
  - Updated `SearchCatalog` in `app.go` with recursive descent traversal (`appendAllDescendants`).
  - When a country or parent region matches the query, all its child districts and provinces are included in the search results with duplicate suppression using a `seen` map.
- **Files Modified**: `app.go`.

---

### 9. Fix Error Handling (Vietnam 503 HTTP Failure Showed 100% Success) [COMPLETED]
- **Issue**: Downloading Vietnam map failed with HTTP 503, but the UI showed 100% progress and green checkmarks everywhere due to unconditional `defer convert:done`.
- **Resolution**:
  - Guarded `convert:done` emission with `success` boolean in `ConvertURL` and `ConvertLocal` in `app.go`.
  - On error, `convert:error` is emitted and `convert:stage` is set to `"error"`. `convert:done` is never emitted.
  - In `ConsoleView`, added `.console-error-card` with an Apple HIG red-tinted alert and `XCircle` icon. The failed stepper stage is marked with red `.error` class without progress completion.
- **Files Modified**: `app.go`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 10. Make Progress Bar Wider and Taller [COMPLETED]
- **Issue**: Progress bar was only 8px high, making text cramped and hard to read.
- **Resolution**:
  - Increased height to 18px with Apple rounded pill squircle styling (`border-radius: 9px`).
  - Added `.progress-bar-meta` header row above the bar displaying the current stage label on the left and real-time transfer metrics and percentage on the right.
  - Styled with vibrant blue gradient and smooth width animation.
- **Files Modified**: `frontend/src/AppleTheme.css`, `frontend/src/App.jsx`.

---

### 11. Display Download Speed and Downloaded / Total Size [COMPLETED]
- **Issue**: Download progress only showed percentage without transfer speed or byte sizes.
- **Resolution**:
  - Enhanced `progressReader` in `internal/download/downloadClient.go` to compute rolling transfer speed (`speedBps`) using elapsed timestamps.
  - In `app.go`, created `ProgressDetail` struct and emitted `convert:progress_detail` containing `downloaded`, `total`, `speedBps`, `speedText` (e.g. `3.4 MB/s`), and `etaSeconds`.
  - In `ConsoleView`, rendered live transfer statistics: `45.2 MB / 180.5 MB (3.4 MB/s) • ETA: 42s`.
- **Files Modified**: `internal/download/downloadClient.go`, `app.go`, `frontend/src/App.jsx`.

---

### 14. Reorder Sidebar Navigation Items [COMPLETED]
- **Issue**: Sidebar order needed adjustment for more intuitive workflow.
- **Resolution**:
  - Reordered navigation items: 1. Catalog, 2. My Maps, 3. PBF Converter, 4. Console, 5. Tools, 6. Settings, 7. About.
- **Files Modified**: `frontend/src/App.jsx`.

---

### 15. Show Top Search Bar Only for Relevant Tabs [COMPLETED]
- **Issue**: Search input in top toolbar was visible across all tabs (Console, Tools, Settings, About) where it had no function.
- **Resolution**:
  - In `Toolbar`, conditionally render `.search-bar` only when `currentView === 'catalog' || currentView === 'my-maps'`.
  - Dynamic placeholder adapts per view: "Поиск регионов (⌘K)" for Catalog and "Поиск по моим картам (⌘K)" for My Maps.
  - Keyboard shortcut `⌘K` only triggers when the search bar is active and visible.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/i18n.js`.

---

### 23. Fix Left-Alignment in Catalog Tree (Antarctica Indentation Bug) [COMPLETED]
- **Issue**: Leaf nodes (e.g. Antarctica) lacked expander chevrons, causing text labels to align flush to the left edge and misalign with parent nodes.
- **Resolution**:
  - Added `.chevron-spacer` (14x14px with `margin-right: 6px`) rendered on leaf nodes in `RegionNode`.
  - All nodes at the same tree depth now align with pixel-perfect consistency.
- **Files Modified**: `frontend/src/AppleTheme.css`, `frontend/src/App.jsx`.

---

### 27. Change Catalog Button Label from "Build" ("Собрать") to "Download" ("Загрузить") [COMPLETED]
- **Issue**: Action buttons in the catalog read "Собрать" ("Build"), which felt like a technical step rather than acquiring a map.
- **Resolution**:
  - Standardized catalog action buttons to `{t('buttons.download')}` ("Загрузить" in Russian, "Download" in English).
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/i18n.js`.

---

### 29. Rename "Ready Maps" Tab to "My Maps" ("Мои карты") [COMPLETED]
- **Issue**: "Ready Maps" / "Готовые карты" felt dated and impersonal compared to modern Apple naming conventions ("My Maps" / "Мои карты").
- **Resolution**:
  - Renamed tab to `myMaps` ("Мои карты" / "My Maps") across `i18n.js` and `App.jsx`.
  - Added real-time filtering of local maps using the top search bar.
- **Files Modified**: `frontend/src/i18n.js`, `frontend/src/App.jsx`.

---

### 30. Background Map Downloading (Do Not Auto-Redirect to Console) [COMPLETED]
- **Issue**: Clicking to download a map immediately pulled the user away to the Console tab, breaking catalog browsing workflow.
- **Resolution**:
  - Removed `setCurrentView('console')` from `handleStartBuild`. Downloads run in the background while the user stays on their current screen.
  - Active background job status is clearly visible via the pulsing green indicator on the "Консоль сборки" sidebar item.
- **Files Modified**: `frontend/src/App.jsx`.

---

### 31. Hierarchical Nested Display in Catalog Search Results [COMPLETED]
- **Issue**: Searching for a parent region (e.g. Russia) rendered a flat list of cards without showing the parent-child relationship.
- **Resolution**:
  - In `app.go`, enhanced `SearchCatalog` with Cyrillic-to-English alias mapping (e.g. 'россия'/'рф' -> 'russia', 'беларусь' -> 'belarus', etc.) and preserved hierarchical subtrees without flattening descendants into root entries.
  - In `CatalogView`, rendered search results using `RegionNode` inside a `.catalog-tree card` with `defaultExpanded={true}`.
  - Parent countries (e.g. Russia) appear with collapsible federal districts and oblasts, each having their own "Загрузить" button.
- **Files Modified**: `app.go`, `app_test.go`, `frontend/src/App.jsx`.

---

### 32. Adjust Vertical Spacing Below "Search Results" Header [COMPLETED]
- **Issue**: Found items in the search results were crammed right beneath the "Результаты поиска" header without sufficient breathing room.
- **Resolution**:
  - Added `.search-results-container .section-title` with `margin-bottom: 16px;`, 13px font size, uppercase letter-spacing in Apple HIG style.
- **Files Modified**: `frontend/src/AppleTheme.css`.

---

### 33. Adaptive Progress Bar Text Color for Theme & Fill State [COMPLETED]
- **Issue**: In Light Theme, progress percentage text inside the progress bar was white, making it completely unreadable against the light background before the blue fill reached the numbers.
- **Resolution**:
  - Applied dynamic classes `.uncovered` (dark text against unfilled track in Light Theme) and `.covered` (white text with text-shadow once covered by blue fill).
- **Files Modified**: `frontend/src/AppleTheme.css`, `frontend/src/App.jsx`.

---

### 22 & 16. Delete Downloaded Map Action & One-Click Update in "My Maps" [COMPLETED]
- **Issue**: Users could not delete downloaded `.img` files directly from within the application, nor update them with a single click (combining Task 22 and Task 16: "Map Updating Capability from Ready Maps Tab").
- **Resolution**:
  - Implemented secure backend `DeleteOutput(path string) error` in `app.go` with 4 security checks (ensuring path is strictly within the `output/` directory, preventing path traversal, verifying `.img` extension, and deleting companion `.meta.json` files).
  - Designed an authentic Apple HIG confirmation sheet modal (`.modal-sheet`, `.modal-overlay` with backdrop blur, smooth spring scale animation, escape key listener) with destructive delete action.
  - Added companion metadata persistence (`<name>.meta.json`) recording source URL, region ID, and timestamps, with automatic fallback matching against Geofabrik catalog for existing maps.
  - Added "Update Map" ("Обновить") button on each card in "My Maps" to re-download and re-compile maps in the background without forcing tab navigation.
- **Files Modified**: `app.go`, `app_test.go`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`, `frontend/wailsjs/go/main/App.js`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/models.ts`.

---

### 26. Catalog Download Status Badge ("Downloaded") & "Update" Button [COMPLETED]
- **Issue**: The catalog did not indicate which maps were already downloaded, showing a generic "Download" button for every region.
- **Resolution**:
  - Added $O(1)$ fast set-based lookup indexing downloaded maps across region names, region IDs, and source URLs.
  - Rendered an Apple HIG frosted green status pill (`.badge-downloaded` with checkmark icon: "Скачано" / "Downloaded") next to compiled regions.
  - In-place button morphing: replaced "Загрузить" with an "Обновить" / "Update" button (`RefreshCw` icon with tinted blue style) for already downloaded maps.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`.

---

### 28. Apple-Style Circular Progress Ring in Catalog and "My Maps" [COMPLETED]
- **Issue**: Starting a download did not provide in-place visual feedback on the item itself in the catalog or in "My Maps".
- **Resolution**:
  - Created reusable `CircularProgress` component featuring an Apple App Store / Safari style SVG ring (`strokeDasharray` / `strokeDashoffset`), background track, dynamic colors, and a central square stop button that highlights red on hover.
  - **In Catalog**: The "Загрузить" / "Обновить" button morphs into the circular progress ring with live percentage or stage label; clicking the central square cancels the active job cleanly via `CancelActiveJob()`.
  - **In "My Maps"**: Immediately inserts an active card at the top of the list showing the circular progress ring, region title, format badge, stage pill, transfer speed, and remaining ETA.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`.

---

### 34. Uniform Catalog Action Buttons & Icon-Only "Downloaded" Badge [COMPLETED]
- **Issue**: "Download" and "Update" buttons in the catalog had mismatched visual widths, and the textual green "Downloaded" / "Скачано" pill (~90px wide) caused uneven button alignment across neighboring rows.
- **Resolution**:
  - Standardized small button geometry: `.btn-primary.btn-sm`, `.btn-secondary.btn-sm`, and `.btn-danger.btn-sm` are unified with `height: 28px; min-width: 92px; font-size: 12px; font-weight: 500; border-radius: var(--radius-sm); box-sizing: border-box; display: inline-flex; align-items: center; justify-content: center; white-space: nowrap;`.
  - Replaced the wide text pill with an Apple HIG compact icon-only squircle badge (`.badge-downloaded`: 28×28px matching the button height, `<CheckCircle2 size={15} />` centered, with tooltip on hover).
  - Wrapped every row's action buttons in `.catalog-action-slot` so both downloaded and non-downloaded items have identical right-alignment.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 36. Truncate Long Map Names with Tooltip in "My Maps" [COMPLETED]
- **Issue**: Long region names (e.g. "Crimean Federal District", "Northwestern Federal District") combined with format badges squeezed the card, causing action buttons ("Показать в Finder / Проводнике") to wrap awkwardly onto a second line.
- **Resolution**:
  - Applied `min-width: 0; flex: 1;` on `.output-info` and `flex-wrap: nowrap; min-width: 0;` on `.output-title-row`.
  - Applied `overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; max-width: 100%;` on `.output-title`.
  - Added HTML `title={cleanName}` and `title={activeJob.name}` attribute tooltips to display full unabbreviated names on mouse hover.
  - Enforced `flex-shrink: 0; white-space: nowrap;` on `.output-actions` and all action buttons, completely preventing unwanted multi-line wrapping.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`.

---

### 37. Quick Navigation to Build Console on Progress Indicator Click [COMPLETED]
- **Issue**: Users downloading maps in the background had to manually switch tabs to view live Splitter and Mkgmap compilation output.
- **Resolution**:
  - Made circular progress rings, percentage labels, and active download cards clickable (`.clickable-progress` with `cursor: pointer` and subtle hover state).
  - Clicking anywhere on the progress indicator calls `setCurrentView('console')`, instantly opening the build console.
  - Isolated the central square cancel button with `e.stopPropagation()`, ensuring cancelling a job cleanly stops the process without triggering an unwanted view change.
  - Added localized tooltips: `"Перейти в консоль сборки"` / `"View build console"`.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`.

---

### 24. Add Apple-Style UI Animations & Transitions [COMPLETED]
- **Issue**: The application lacked native Apple micro-interactions, such as fluid tab transitions, button active compression, and smooth tree chevron rotation.
- **Resolution**:
  - **Tab / View Transitions**: Added `@keyframes appleViewFadeSlide` (`opacity: 0 -> 1`, `transform: translateY(4px) -> translateY(0)` with Apple standard ease `cubic-bezier(0.16, 1, 0.3, 1)` in 180ms) on `.view-container`.
  - **Tree Expand / Collapse**: Smooth chevron rotation (`.chevron { transition: transform 180ms cubic-bezier(0.16, 1, 0.3, 1); } .chevron.expanded { transform: rotate(90deg); }`), eliminating SVG re-mount flicker.
  - **Button Press Feedback**: Added `:active:not(:disabled)` compression with `transform: scale(0.97)` and Apple timing curves.
  - **Card Hover Elevation**: Added subtle hover shadow lift and border transition on `.output-card`.
- **Files Modified**: `frontend/src/AppleTheme.css`, `frontend/src/App.jsx`.

---

### 12. Support Alternative Map Data Providers (Beyond Geofabrik) [COMPLETED]
- **Issue**: Relying exclusively on Geofabrik was fragile due to occasional downtime, rate limiting, and lack of city-level extracts.
- **Resolution**:
  - Created `internal/provider` package providing unified provider registry:
    - **Geofabrik**: Hierarchical regional and sub-regional extracts.
    - **OpenStreetMap France**: Mirror extracts with automatic URL translation (`provider.MapGeofabrikURLToOSMFrance`).
    - **BBBike.org**: 240+ metropolitan city extracts worldwide (`provider.BuildBBBikeCatalog`) arranged into continent/country/city hierarchy.
  - Implemented backend provider endpoints in `app.go`: `GetProviders()`, `GetCatalogByProvider(id)`, `ForceRefreshCatalogByProvider(id)`, `SearchCatalogByProvider(id, query)`.
  - Added Apple Segmented Control in Catalog header to smoothly switch providers on the fly.
  - Companion `.meta.json` records `sourceType` (`geofabrik`, `osm-france`, `bbbike`, `local`) for accurate map provenance.
- **Files Modified**: `internal/provider/provider.go`, `internal/provider/provider_test.go`, `app.go`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`.

---

### 13. System Proxy & Custom Proxy Configuration [COMPLETED]
- **Issue**: Users in restricted networks or behind proxies needed system proxy support and custom proxy overrides.
- **Resolution**:
  - Added `download.ProxyConfig` with `SetGlobalProxy` and `GetGlobalProxy` in `internal/download/downloadClient.go`.
  - Configured HTTP transport proxy resolver honoring system environment (`HTTP_PROXY`, `HTTPS_PROXY`) by default.
  - Supports custom HTTP, HTTPS, and SOCKS5 proxy URLs with validation.
  - Added proxy configuration panel in Settings tab with toggle for system proxy, custom proxy URL input, and persistence to `cache/proxy-config.json`.
- **Files Modified**: `internal/download/downloadClient.go`, `internal/download/downloadClient_test.go`, `app.go`, `app_test.go`, `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`.

---

### 25. Investigate & Resolve "Timeout Awaiting Response Headers" Error [COMPLETED]
- **Issue**: Geofabrik occasionally timed out on cold cache extracts (e.g. Benin), causing abrupt failures followed by false success reports.
- **Resolution**:
  - Increased `ResponseHeaderTimeout` from 30s to 90s in `internal/download/downloadClient.go`.
  - Implemented `DownloadWithOptions` with automatic retry loop using exponential backoff (up to 3 attempts, 2s/4s delay) and `[WARN]` logs streamed to build console.
  - Fast-fails on non-retryable 404/403 errors and user cancellations.
  - Fixed completion event logic to prevent false `convert:done` emissions on failed or cancelled downloads.
- **Files Modified**: `internal/download/downloadClient.go`, `internal/download/downloadClient_test.go`, `app.go`.

---

### 35. Multi-Map Sequential Download & Compilation Queue [COMPLETED]
- **Issue**: Starting a map build blocked all buttons across the application, preventing users from selecting multiple maps to download.
- **Resolution**:
  - Replaced button disablement with a client-side download queue state machine (`downloadQueue: Array<{ id, name, url, heapSize }>`).
  - Requesting a download/update while another map is active enqueues the job without blocking the UI.
  - Queued items display an amber `.btn-queued` button with a Clock icon and `"В очереди"` / `"Queued"` badge. Clicking removes the item from the queue.
  - Dedicated "Очередь загрузки" ("Download Queue") section in "My Maps" displaying queued cards with sequential index (`#1`, `#2`), target URL, waiting badge, individual cancel buttons, and "Очистить очередь" ("Clear Queue").
  - Auto-advances queue upon job completion or cancellation.
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/AppleTheme.css`, `frontend/src/i18n.js`.

---

### 38. Map Catalog Renaming & Instant Auto-Load on Startup [COMPLETED]
- **Issue**: Catalog navigation was labeled specifically as "Каталог Geofabrik", and opening the app required a manual tab visit to begin fetching maps.
- **Resolution**:
  - Renamed tab and headers from "Каталог Geofabrik" to **«Каталог карт»** / **"Map Catalog"** across navigation and toolbars.
  - Added background catalog auto-load immediately on application startup (`loadCatalog(false, defaultSource)` on initial mount), dramatically improving startup UX.
  - Added **"Основной источник карт"** ("Default Map Source") selector in Settings persisted to `localStorage` (`zimap_default_source`).
- **Files Modified**: `frontend/src/App.jsx`, `frontend/src/i18n.js`, `frontend/src/AppleTheme.css`.

---

### 8. System Notifications on Map Conversion Complete [COMPLETED]
- **Issue**: When map conversion finishes, user is not notified if the window is minimized or in the background.
- **Resolution**:
  - Implemented `SendNotification(title, message string) error` method on `*App` with native support for macOS (`osascript`), Windows (PowerShell balloon/toast notification), and Linux (`notify-send`).
  - Exposed `SendNotification` to Wails IPC bindings.
  - Triggered desktop notifications upon successful map compilation (`"Карта %s успешно собрана!"`) and compilation errors (`"Ошибка сборки карты %s: %s"`) in both `ConvertURL` and `ConvertLocal`.
  - Added comprehensive unit tests in `app_test.go` verifying normal and special-character notification delivery.
- **Files Modified**: `app.go`, `app_test.go`, `frontend/wailsjs/go/main/App.d.ts`, `frontend/wailsjs/go/main/App.js`.

---

### 18. Rename Application to "ZiMap Downloader" [COMPLETED]
- **Issue**: Application needed full branding alignment across window title, bundle identifier, binary name, and CLI help.
- **Resolution**:
  - Updated window title in `main.go` to `"ZiMap Downloader"`.
  - Updated CLI help and banner text in `main.go` from `".img Map"` to `"ZiMap Downloader"`.
  - Updated `wails.json`: `"name": "zimap"`, `"outputfilename": "ZiMap"`, `"productName": "ZiMap Downloader"`, `"companyName": "Ziviy"`, `"comments": "ZiMap Downloader - OpenStreetMap to .img format compiler for GPS navigators and outdoor devices"`.
  - Updated `build/darwin/Info.plist` with `CFBundleIdentifier` set to `com.ziviy.zimap`.
  - Verified `build/windows/info.json` metadata templates automatically populate with `"ZiMap Downloader"`.
- **Files Modified**: `main.go`, `wails.json`, `build/darwin/Info.plist`, `build/windows/info.json`.

---

### 21. Redraw Application Icon in Apple Squircle Style [COMPLETED]
- **Issue**: Application needed an authentic, high-resolution macOS Big Sur / Sonoma squircle icon without any trademarked logos or text.
- **Resolution**:
  - Generated a brand-neutral Apple Squircle icon (1024×1024) featuring a deep slate/charcoal frosted glass surface with subtle gold and cyan topographic contour elevation lines, a geometric mountain peak, and an orange compass navigation needle.
  - Processed with antialiased Apple superellipse squircle curve (`n=4.4`) with transparent background into `build/appicon.png` (1024×1024 RGBA).
  - Built macOS native `build/darwin/iconfile.icns` using `/usr/bin/iconutil` with full Retina resolutions (16×16 through 1024×1024).
  - Built Windows multi-resolution `build/windows/icon.ico` (16, 32, 48, 64, 128, 256 px) using PNG-compressed directory format.
  - Verified bundle icon integration during `wails build`.
- **Files Created/Modified**: `build/appicon.png`, `build/darwin/iconfile.icns`, `build/windows/icon.ico`.

---

### 50. Architectural Codebase Refactoring & Comprehensive Test Suite [COMPLETED]
- **Issue**: The application had grown monolithic (`app.go` with 1678 lines and `App.jsx` with 2550 lines), test coverage was limited, and latent parameter ordering bugs existed in converter options.
- **Resolution**:
  - **Comprehensive Test Suite (Pre-Refactor)**:
    - Added extensive Go unit test suites covering `internal/archive` (.tar.gz, path traversal, truncation, corrupt headers), `internal/setup` (path discovery, mock tool verification, environment overrides), `internal/cli` (command registry, empty args, unknown commands), and `internal/converter` (context cancellation, file format validations, detail level normalization).
    - Fixed latent parameter order bug in `internal/converter/convert.go` where `opts.MapName` and `detailLevel` were inverted in `mkgmapArguments`.
    - Added `app_test.go` with `httptest.Server` mocks for GitHub Releases updates, system info, job cancellation, precompiled BBBike downloads, and error fallbacks.
    - Added Vitest unit test suite in `frontend/` (`i18n.test.js`, `formatters.test.js`, `queue.test.js`) with 23 passing tests (100% pass rate).
  - **Modular Backend Architecture**:
    - Split `app.go` into domain-specific files in `package main`:
      - `app.go`: struct definition, lifecycle hooks (`startup`, `shutdown`), and safe `emitEvent` helper.
      - `app_catalog.go`: catalog loading, multi-tier caching (RAM + 24h disk), Russian alias mappings, and recursive tree search.
      - `app_convert.go`: pipeline execution (`ConvertURL`, `ConvertLocal`), precompiled BBBike downloads, and job cancellation.
      - `app_outputs.go`: output file listing, companion metadata, safe deletion, native file revelation, and export dialogs.
      - `app_tools.go`: Adoptium JDK / Mkgmap / Splitter status checks, tool downloads, and system info.
      - `app_updater.go`: semver version comparison and GitHub Releases checking.
      - `app_storage.go`: disk space metrics and temporary/cache cleanup.
      - `app_proxy.go`: proxy configuration management.
      - `app_notify.go`: cross-platform desktop notifications.
  - **Modular Frontend Architecture**:
    - Modularized `frontend/src/App.jsx` from 2550 lines to ~750 lines as the core state orchestrator.
    - Extracted clean components under `frontend/src/components/`:
      - `components/common/`: `CircularProgress.jsx`, `ToastNotification.jsx`
      - `components/modals/`: `DeleteConfirmationModal.jsx`, `UpdateModal.jsx`, `StorageCleanModal.jsx`
      - `components/navigation/`: `Sidebar.jsx`, `Toolbar.jsx`
      - `components/views/`: `CatalogView.jsx`, `OutputsView.jsx`, `ConsoleView.jsx`, `LocalConvertView.jsx`, `ToolsView.jsx`, `SettingsView.jsx`, `AboutView.jsx`
      - `utils/`: `formatters.js`, `queue.js`
  - **Verification**:
    - All Go tests pass with 0 errors (`go test -count=1 ./...`).
    - All Vitest tests pass with 0 errors (`npm test`).
    - Clean production builds for Vite (`npm run build`) and macOS Wails bundle (`wails build`).
- **Files Created/Modified**:
  - `app.go`, `app_catalog.go`, `app_convert.go`, `app_outputs.go`, `app_tools.go`, `app_updater.go`, `app_storage.go`, `app_proxy.go`, `app_notify.go`, `app_test.go`
  - `internal/converter/convert.go`, `internal/converter/convert_test.go`, `internal/cli/registry.go`, `internal/cli/registry_test.go`, `internal/archive/targz_test.go`, `internal/setup/paths_test.go`, `internal/setup/setup_test.go`
  - `frontend/src/App.jsx`, `frontend/src/components/*`, `frontend/src/utils/*`, `frontend/src/i18n.test.js`
  - `AGENTS.md`, `TODO.md`





