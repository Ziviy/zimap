# ZiMap architecture and module guide

This is the source-oriented entry point for agents and maintainers. Read [AGENTS.md](../AGENTS.md) for contribution rules, use this guide to choose the relevant files, and read only the implementation and tests needed for the task. [TODO.md](../TODO.md) tracks planned work; [plans/](plans/) contains historical implementation plans, not authoritative descriptions of current behavior.

**Maintenance rule:** update this guide when module ownership, public bindings, events, paths, or build requirements change. Source and configuration files win if this guide becomes stale. There is no need to re-audit the whole repository for an ordinary localized change.

## 1. Quick task-to-file map

| Task | Start here | Connected code / tests |
| --- | --- | --- |
| Catalog, providers, region search | `app_catalog.go`, `internal/provider/provider.go` | `internal/geofabrik/catalog.go`, `CatalogView.jsx`, catalog/provider tests |
| Download or conversion failure | `app_convert.go`, `internal/converter/convert.go` | `internal/download/downloadClient.go`, `internal/archive/`, `ConsoleView.jsx`, conversion/download tests |
| Queue order, duplicate downloads, progress state | `frontend/src/App.jsx`, `frontend/src/utils/queue.js` | `queue.test.js`, `CatalogView.jsx`, `OutputsView.jsx`, backend completion events |
| List, update, delete, or export local maps | `app_outputs.go`, `OutputsView.jsx` | `app_outputs_test.go`, `app_test.go`, `app_convert.go`, `DeleteConfirmationModal.jsx` |
| Device detection, protected maps, freshness | `app_device.go`, `internal/device/detector.go`, `freshness.go`, `filter.go` | `DeviceView.jsx`, `app_device_test.go`, device/freshness tests |
| Transfer or cancellation on a device | `internal/device/sync.go`, `app_device.go` | `sync_types.go`, `sync_test.go`; for MTP also `mtp_service.go` and `cmd/zimap-mtp/main.go` |
| Tools, Java, heap size, or data locations | `internal/setup/paths.go`, `tools.go`, `app_tools.go` | `internal/converter/convert.go`, `SettingsView.jsx`, setup tests |
| Self-update or release packaging | `app_updater.go`, `.github/workflows/release.yml` | `scripts/build-mtp.sh`, `Makefile`, `UpdateModal.jsx`, updater tests in `app_test.go` |
| Proxy or cache cleanup | `app_proxy.go`, `app_storage.go` | `internal/download/downloadClient.go`, `app_catalog.go`, `StorageCleanModal.jsx`, `app_test.go` |
| UI styling, text, or navigation | `frontend/src/AppleTheme.css`, `i18n.js`, `components/navigation/` | relevant view, `i18n.test.js`, `utils/formatters.js` |

Backend paths are repository-relative. View/component basenames above live under `frontend/src/components/`; internal device basenames live under `internal/device/`.

## 2. System shape and entry points

ZiMap has two front doors to the same local Go libraries, plus a separate USB helper:

```text
main.go
  ├─ download / convert / update ──> internal/cli ──┐
  ├─ --help / -h / help: print help, no GUI         │
  └─ no arguments: Wails desktop                  │
       embedded frontend/dist                    │
       React App.jsx                             │
          ├─ generated Wails Promise bindings    │
          └─ Wails event subscriptions           │
                    ↕                            │
             Go App service (app_*.go)            │
                    └────────────────────────────┤
                                                ↓
             internal/{download,converter,setup,archive,
                       geofabrik,provider,device}
                        ├─ HTTP downloads / GitHub releases
                        ├─ Java → splitter.jar → mkgmap.jar
                        ├─ mounted device filesystem
                        └─ MTPService → zimap-mtp → libusb / MTP device
```

- [main.go](../main.go) is the packaged desktop/CLI entry point. It embeds `frontend/dist` with `go:embed`, dispatches the three recognized CLI commands before Wails starts, handles help directly, and binds `*App` to the desktop. Unrecognized arguments currently fall through to GUI startup.
- [cmd/main.go](../cmd/main.go) is an additional CLI-only entry point. It calls the same command registry; it is not the desktop bootstrap.
- [cmd/zimap-mtp/main.go](../cmd/zimap-mtp/main.go) is a separately built executable, not a Wails binding or another React backend.
- [frontend/src/main.jsx](../frontend/src/main.jsx) mounts `App` inside React `StrictMode`. Navigation is conditional rendering via `currentView`, not a router.
- The UI talks to Go through Wails IPC, not a custom REST API. No database, remote job service, or persistent backend queue is involved.

## 3. Go application service modules

All root `app*.go` files belong to **one `package main` and one `App` service**, not independently instantiated services. Exported `App` methods form the desktop API. Common integration tests are in [app_test.go](../app_test.go), with separate output and device tests.

| File | Responsibility / main entry points |
| --- | --- |
| [app.go](../app.go) | `App`, `NewApp`, Wails lifecycle/context, `emitEvent`, shared progress DTO. Owns catalog caches, conversion cancel function, device services, and update state. `NewApp` loads proxy configuration. `startup` saves the Wails context; `domReady` and `shutdown` are currently empty. |
| [app_catalog.go](../app_catalog.go) | `RegionData`/`CatalogData`; provider enumeration, memory/disk catalog caching, refresh, hierarchical search and Russian aliases. Legacy `GetCatalog`/`SearchCatalog` methods select Geofabrik. |
| [app_convert.go](../app_convert.go) | `ConvertURL`, `ConvertLocal`, `CancelActiveJob`; manages the conversion busy slot, contexts, work directories, progress, metadata and completion. BBBike style ZIPs bypass Java through `downloadPrecompiledBBBike`. `emitDownloadProgress` is shared by both download paths. |
| [app_outputs.go](../app_outputs.go) | `OutputItem`/`MapMetadata`; list maps, resolve catalog provenance, read/write companion metadata, delete, reveal and export. `outputMapPath` validates against an `os.Root`; delete/open operations stay relative to that root. `copyFile`/`copyOpenFile` stream, guard same-file aliases, sync and report close errors. |
| [app_device.go](../app_device.go) | Desktop adapter for detection, protected deletion, eject/reveal, freshness, single/batch sync and cancellation. Converts `OutputItem` into device-local map DTOs. Routes `mtp://` paths to MTP. `GetConnectedGarminDevices`, `GetGarminUSBStatus` and `EjectGarminDevice` are compatibility aliases. |
| [app_tools.go](../app_tools.go) | `CheckTools`, `UpdateTools`, system/version information and browser opening. Converts setup detection into GUI status. Tool-update stdout is forwarded as `tools:log`. |
| [app_updater.go](../app_updater.go) | Release DTOs, version comparison, platform/architecture asset matching, `CheckForUpdate`, asynchronous `InstallUpdate`, cancellation, simulation, extraction, replacement and restart. Also contains directory-copy helpers and backend byte/ETA formatting. |
| [app_storage.go](../app_storage.go) | `GetStorageMetrics`, `CleanStorage`, `ClearTempAndCacheFiles`, `RevealDataDirectory`: account for maps/cache/tools/workspaces and remove selected storage categories (cache, temp, maps, tools, settings), removing empty root directories on a full clean; blocks cleanup while a conversion slot is occupied. |
| [app_proxy.go](../app_proxy.go) | `GetProxy`/`SetProxy`: validate HTTP/HTTPS/SOCKS5 proxy URLs, persist JSON, update the shared downloader configuration. |
| [app_notify.go](../app_notify.go), `app_notify_darwin.go`, `app_notify_other.go` | Native notifications via `UNUserNotificationCenter` (macOS `.app` bundle, cgo) with AppleScript fallback, PowerShell (Windows), or `notify-send` (Linux). OS delivery is separate from browser notifications in the frontend. |

### State and execution ownership

- `App.jobMutex` protects `cancelJob`, the backend's single conversion busy slot. The **React queue** serializes requested downloads; it is not stored in Go or persisted across restarts.
- `catalogMutex` protects catalog cache objects/timestamps. `updateMutex` protects `updateRunning`/`updateCancel`. `device.SyncManager` has its own transfer lock and cancellation state. These are separate workflows, not one global scheduler.
- **Do not assume every binding returns a start acknowledgement.** `ConvertURL`, `ConvertLocal` and `UpdateTools` currently execute their main work before returning; the frontend handles their Promises and streamed events. `InstallUpdate`, `SyncMapToDevice` and `SyncAllOutdatedMaps` launch background work. Check the caller contract before changing synchronous work into a goroutine.
- `emitEvent` needs the actual Wails lifecycle context. Some unit tests avoid GUI work with a nil context; a plain `context.Background()` is not a substitute for a complete Wails runtime context.
- Heap settings and the tool-update stdout redirect are process-global. Avoid adding parallel jobs/tests around them without addressing that ownership. Window shutdown does not currently coordinate cancellation of all workflows.

## 4. Internal Go packages

These packages are reusable by CLI and desktop. Keep Wails imports out of them.

| Package / files | What belongs here |
| --- | --- |
| [internal/cli/](../internal/cli/) | `registry.go`/`command.go`: command dispatch; `download.go`: URL or interactive catalog download; `region_selector.go`: terminal hierarchy/confirmation; `convert.go`: local conversion and `--detail`; `update.go`: tools. `map_output.go` owns CLI workspaces, naming and final publication. Tests include command, selector, conversion and publication behavior. |
| [internal/converter/convert.go](../internal/converter/convert.go) | `Run`, `RunNamed`, `RunWithContext`; supported OSM input validation, detail presets, Java heap arguments, subprocess execution, log callbacks and splitter/mkgmap argument construction. Produces **`gmapsupp.img` in the caller's work/output directory**; the caller chooses the final filename. Tests use controlled tools/fixtures, not a full real-world map build. |
| [internal/download/](../internal/download/) | `downloadClient.go`: HTTP client, proxy transport, retry/backoff, cancellation, transfer metrics, staged download publication and HEAD content length. `New(nil)` has a bounded overall timeout; `NewStreaming()` has connection/header bounds but no blanket body deadline. `downloadWorkflow.go` provides small CLI/tool download entry points. Tests cover HTTP failure, retries, cancellation, existing-target preservation and directory rejection. |
| [internal/geofabrik/catalog.go](../internal/geofabrik/catalog.go) | Fetch/parse the Geofabrik no-geometry JSON index, validate IDs, and expose sorted roots/children/parents. `IndexURL` is the source URL. Does not own the desktop cache. |
| [internal/provider/provider.go](../internal/provider/provider.go) | Provider IDs/info, France URL mapping, static BBBike catalog, city aliases and BBBike style URL normalization/detection. Provider selection is not a plugin loader. |
| [internal/setup/](../internal/setup/) | `paths.go`: data-root resolution; `setup.go`: aggregate tool checks/updates; `tools.go`: managed Java/JAR paths, system Java fallback, downloads, distribution validation, staged installation and OS/architecture mapping. `Tools` declares current versions/URLs. A managed Java 21 runtime is stored under `tools/jdk` even though the download URL requests a JRE. |
| [internal/archive/](../internal/archive/) | Extension dispatch in `archive.go`; ZIP, TAR and gzip-compressed TAR extraction. Shared path/symlink validation lives in `tar.go`. Rejects traversal and unsupported archive links/types; preserve these checks when changing extraction. |
| [internal/device/](../internal/device/) | See the device submodules below. No React/Wails dependency; mounted filesystem and MTP adapters share public device/freshness/progress DTOs. |

### Device submodules

| Files in `internal/device/` | Responsibility |
| --- | --- |
| `device.go`, `storage.go`, `sync_types.go` | Device/XML/map/storage models; freshness statuses/reports and sync progress JSON shapes. |
| `detector.go`, `detector_darwin.go`, `detector_windows.go`, `detector_linux.go` | Enumerate platform mount candidates, inspect `GARMIN`/`GarminDevice.xml`, list custom and preinstalled maps, and also discover MTP devices. |
| `storage_unix.go`, `storage_windows.go` | OS-specific capacity/free-space calls. |
| `img_parser.go` | Read internal `.img` titles from the binary header. |
| `filter.go` | System/OEM filename protection, signature/license detection, friendly factory-map titles and mounted-device deletion validation. This is distinct from local-output validation in `app_outputs.go`. |
| `freshness.go` | Correlate maps by filename, region/title and internal name; FAT32 two-second timestamp tolerance; 20 MiB storage reserve; summary report. Status strings: `up_to_date`, `update_available`, `not_on_device`, `only_on_device`. |
| `sync.go` | Resolve destinations, stream mounted-file transfers, sync/close files, replace targets, preserve timestamps; `SyncManager` serializes/cancels mounted and MTP transfers. `Sync`/`SyncMTP` are synchronous; `StartSync`/`StartSyncMTP` are asynchronous wrappers. |
| `mtp_detector.go` | `mtp://storage-<id>` identity, sidecar detection results → `GarminDevice`, custom/preinstalled separation. These paths are identifiers, not filesystem paths. |
| `mtp_service.go` | Find/run the helper, parse JSON, translate upload progress into `SyncProgressEvent`, and cancel active uploads. Resolution order: explicit service override → `ZIMAP_MTP_BIN` → executable/bundle locations → workspace `tools/mtp` → `PATH`. |
| `usb_diagnostics.go` | USB presence and competing-process diagnostics when a watch is not mounted or its interface is busy. |

## 5. Frontend and desktop boundary

[frontend/src/App.jsx](../frontend/src/App.jsx) is the UI orchestrator: preferences, selected view/provider, catalog/output/device data, queue, active/completed jobs, progress/logs, tools, storage, updater state and event subscriptions. Backend actions and refreshed results are passed down as props. Do not move job lifecycle into a screen that disappears when navigating away.

| Frontend module | Responsibility |
| --- | --- |
| `components/navigation/Sidebar.jsx`, `Toolbar.jsx` | View navigation, global indicators, contextual catalog/My Maps search, theme control and search shortcut. |
| `components/views/CatalogView.jsx` | Provider/style selection, nested region tree/search, downloaded/queued/active controls. |
| `components/views/LocalConvertView.jsx` | Local-file selection/drop UI, map name, heap and detail preset. |
| `components/views/ConsoleView.jsx` | Conversion logs, stage stepper, metrics, error and completed-map actions. |
| `components/views/OutputsView.jsx` | My Maps filtering, active/queued cards, update/export/reveal/delete actions. |
| `components/views/DeviceView.jsx` | Device presentation, custom/factory map distinction, freshness filters, sync and deletion UI. |
| `components/views/ToolsView.jsx`, `SettingsView.jsx`, `AboutView.jsx` | Tool status/logs; persisted preferences, proxy/storage/update controls; project/version information. |
| `components/modals/` | Delete-map/device-map confirmations, storage cleanup and updater sheets. Keep hook calls before conditional early returns. |
| `components/common/` | Shared circular build progress and toast presentation. |
| `utils/queue.js` | Queue naming/URL deduplication, item creation/removal, FIFO and cancellable delayed start. The queue is removed from only when a job actually starts; effect cleanup cancels pending timers. Covered by `queue.test.js`. |
| `utils/formatters.js` | Size/date/speed/ETA formatting, platform reveal labels and validated external URL opening. Covered by `formatters.test.js`. |
| `i18n.js`, `i18n.test.js` | RU/EN dictionaries and key parity. Add translations in both languages. |
| `AppleTheme.css` | Active design tokens, layout, vibrancy, themes, animations, responsive behavior and Wails drag regions. Do not assume leftover scaffold CSS is imported. |

### Bindings and data shapes

- [frontend/wailsjs/go/main/App.js](../frontend/wailsjs/go/main/App.js) and [App.d.ts](../frontend/wailsjs/go/main/App.d.ts) are Wails method bindings; [models.ts](../frontend/wailsjs/go/models.ts) contains generated model shapes. Regenerate/review them when Go method signatures or DTOs change. Do not add application logic to generated files.
- [frontend/wailsjs/runtime/](../frontend/wailsjs/runtime/) supplies event/dialog/runtime APIs. `App.jsx` registers and cleans up the central event listeners.
- JSON spelling is part of the API: a region uses **`pbfUrl`**, an `OutputItem` uses **`sourceUrl`**, but `MapMetadata` uses **`sourceURL`** and **`regionID`**. Do not normalize these silently on only one side.

### Main event contracts

| Events | Payload / meaning |
| --- | --- |
| `convert:stage` | Stage string. Java pipeline uses `download`, `splitter`, `mkgmap`; BBBike also uses `compile` and `finalize`; failures use `error`. The frontend sets its own `done` stage on completion. Do not rename these to a single assumed stage vocabulary without updating consumers. |
| `convert:progress`, `convert:log`, `convert:error` | Integer percentage, log string, error string respectively. Compilation progress is stage-oriented, not a guaranteed smooth percentage. |
| `convert:progress_detail` | `downloaded`, `total`, `speedBps`, `speedText`, `etaSeconds` from `ProgressDetail` in `app.go`. |
| `convert:done` | Success signal without a required payload. Only after successful publication; the conversion busy slot is released before this signal. The UI uses its active-job ref to identify the finished job. |
| `tools:log` | Captured tool-update stdout chunks, not necessarily complete lines. |
| `device:sync_start`, `device:sync_done`, `device:sync_cancelled`, `device:sync_error` | Map identifiers (`mapId`, `fileName`); errors add `error`; batch events can add index/count/name fields. |
| `device:sync_progress` | `SyncProgressEvent`: `mapId`, `fileName`, `bytesCopied`, `totalBytes`, `percentage`, `speedBytesSec`, `speedFormatted`, `etaSeconds`. |
| `device:sync_all_start`, `device:sync_all_done` | Batch `total`; individual map events still occur. |
| `update:stage`, `update:progress`, `update:log`, `update:error` | Stage string, integer percentage, log string, error string. Stages include download/extract/install/ready/error/cancelled. |
| `update:metrics`, `update:done` | Metrics use display strings `speed`, `downloaded`, `total`, `eta`. Real-install completion includes `ready`/`target`; consumers should not require that payload for simulation. |

## 6. Main workflows

### Catalog and search

1. `App.jsx` loads the default provider on startup; `CatalogView` requests backend search.
2. `GetCatalogByProvider` uses a 24-hour memory cache, then provider-named JSON disk caches for Geofabrik/France. BBBike builds a static in-code catalog.
3. France reuses the Geofabrik hierarchy with mapped extract URLs; it is not a separately parsed France catalog. Search aliases and hierarchy preservation are in `app_catalog.go`.
4. Refresh fetches the catalog and separately downloads the disk-cache payload. These paths currently use the default HTTP client; they do **not** all share the downloader's proxy/timeout policy.

### Download and local conversion

1. React starts a request immediately or appends it to the in-memory queue. A cancellable 100 ms timer starts the next queued job after the running job ends; a queued item stays in state until that timer fires.
2. `ConvertURL` reserves the backend busy slot, creates a context and downloads into an output-directory workspace. `ConvertLocal` starts with an existing input file. Supported inputs: `.osm.pbf`, `.pbf`, `.osm`, `.osm.gz`, `.osm.bz2`.
3. `runConversion` creates a `.zimap-<name>-*` workspace. `RunWithContext` validates inputs/tools/detail, runs Java splitter into `.splitter`, requires `template.args`, runs mkgmap and verifies `gmapsupp.img` exists.
4. The GUI wrapper publishes `<name>.img`, writes `<name>.meta.json` and completes through its shared defer. BBBike style archives instead download/extract/find `gmapsupp.img` and publish directly, without Java. Metadata-write failures are warnings, not map-build failures.
5. Errors propagate and must not emit success. Download cancellation is checked before publishing the staged file. Do not replace completion guards with an unconditional deferred success event.
6. CLI commands call the same downloader/converter but use `internal/cli/map_output.go` for staging/final naming; CLI and GUI do **not** share a single publication/metadata implementation.

### Local output management

`ListOutputs` finds `.img` files and reads `<name>.meta.json` or the older `<name>.img.meta.json`. For legacy maps without metadata it may fetch/consult the catalog and persist inferred metadata: it is not strictly a read-only/no-network operation. Delete/export validate local output paths through `os.Root`, including symlink containment; export then uses a native Save dialog. The shared copy helper handles same-file/hard-link aliases without truncating the source.

### Mounted and MTP device sync

Detection creates a shared device model from either a real mount or sidecar JSON. Freshness compares local outputs with custom device maps; batch sync selects missing/outdated items and stops on error/cancellation. Mounted transfers stage `<target>.tmp-zimap`, flush/close and replace, then attempt to preserve the source timestamp. There is an unlink-before-retry fallback on some rename failures: **not every platform path is an atomic replacement**. MTP uploads use the helper protocol instead, not this filesystem staging algorithm. Keep factory-map protection in both delete/upload routing paths.

The sidecar supports `detect`, `list-maps`, `upload`, `delete`, `check-conflicts` and `--mock`. Upload uses newline-delimited JSON progress/result messages. The macOS bundle contains `zimap-mtp` beside the main executable; native USB libraries stay in this helper rather than the GUI's device adapter.

### Application and tool updates

Application self-update checks GitHub releases via the REST API (honoring `GITHUB_TOKEN`/`GH_TOKEN` if present) and automatically falls back to GitHub web release endpoints (`/releases/latest` redirect, `/releases.atom`, and `/releases/expanded_assets/<tag>`) when unauthenticated REST requests hit HTTP 403/429 IP rate limits. It matches an OS/architecture asset, downloads/extracts to a temporary workspace, locates the payload and replaces the bundle/executable with backup handling. macOS extraction prefers `ditto` to preserve bundles; restart launches the new app and quits the current process. Asset names in release CI must stay compatible with `scoreAsset`.

Tool updates are separate: `setup.Tools` declares Java/mkgmap/splitter artifacts; staged downloads are validated before replacing installations. `CheckTools` reports missing/invalid tools; it does not install them. The GUI calls `UpdateTools` or the CLI calls `update` to install/repair them.

## 7. Storage, preferences and environment

There is no database. Maps/metadata, cache/proxy JSON and managed binaries are filesystem state; preferences are browser `localStorage`; jobs/caches also have in-memory state.

[internal/setup/paths.go](../internal/setup/paths.go) resolves the mutable data root (`ProjectRoot` / `DataDirectory`) in this order:

1. `ZIMAP_HOME`, then legacy `GARMIN_MAP_HOME` (converted to an absolute path).
2. Portable directory `<exeDir>/zimap-data` when a `.portable` marker file or `zimap-data/` directory exists alongside the executable (outside macOS `.app` bundles).
3. OS-standard private application data directory:
   - macOS (`darwin`): `~/Library/Application Support/ZiMap`
   - Windows (`windows`): `%LOCALAPPDATA%\ZiMap` (fallback `%APPDATA%\ZiMap` or `<exeDir>\zimap-data`)
   - Linux / Unix: `$XDG_DATA_HOME/ZiMap` or `~/.local/share/ZiMap` (fallback `os.UserConfigDir()/ZiMap`)
4. Working directory fallback if OS directories cannot be determined.

Relative to that root: `output/` holds maps/metadata/workspaces, `cache/` holds provider indexes and `proxy-config.json`, and `tools/` holds managed runtimes/JARs. Managed Java is preferred over `PATH`; mkgmap/splitter resolve from configured distributions. Tool paths/version URLs are declared in `internal/setup/tools.go`, not Wails configuration.

| Setting | Meaning |
| --- | --- |
| `ZIMAP_JAVA_HEAP`, legacy `GARMIN_MAP_JAVA_HEAP` | Validated JVM heap override such as `8g`/`4096m`; otherwise `-XX:MaxRAMPercentage=50.0`. GUI heap input is applied through the process environment. |
| `ZIMAP_MTP_BIN` | Override sidecar executable discovery. |
| `ZIMAP_MOCK_GARMIN_DIR` | Add a fake mounted-device directory to detection. |
| `ZIMAP_MOCK_GARMIN_MTP=1` | Request sidecar mock mode; still needs a built helper to execute it. |
| `ZIMAP_UPDATE_URL` | Override the GitHub releases endpoint for checking updates. |
| `HTTP_PROXY` / `HTTPS_PROXY` | System-proxy behavior in the shared downloader; custom settings are saved via `SetProxy`. Do not assume every direct `http.Get` follows custom configuration. |

Frontend preference keys: `zimap_theme`, `zimap_language`, `zimap_heap`, `zimap_default_source`, `zimap_detail`, `zimap_notifications_enabled`, and `zimap_bbbike_style` (catalog). Download queues are not saved to these keys.

## 8. Verification and packaging

### Toolchain and routine checks

- Go requirement: [go.mod](../go.mod), currently **1.26.5+**. This is also needed for the standard-library filesystem APIs used by output validation.
- Use **Node 22.x, at least 22.12**, to match release CI and the current Vite/Vitest engines. [frontend/package.json](../frontend/package.json) declares scripts; `package-lock.json` locks the frontend dependency graph.
- Build `frontend/dist` before building/testing the root Go package if assets are absent: `main.go` embeds this directory. Do not build frontend assets concurrently with a Go build that is reading them.

```bash
# From repository root; install frontend dependencies if necessary.
npm --prefix frontend ci
npm --prefix frontend test
npm --prefix frontend run lint
npm --prefix frontend run build

go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go run . --help
```

**Native dependency trap:** `go test ./...` also compiles `cmd/zimap-mtp`, which uses CGO/libusb. On Debian/Ubuntu, install `pkg-config` and `libusb-1.0-0-dev` first (or supply equivalent compiler/pkg-config/library paths). A missing `libusb-1.0.pc` is an environment/build dependency failure, not a downloader test failure. A native Wails GUI build additionally needs its platform SDK/webview dependencies.

**MTP coverage trap:** tests in `internal/device/mtp_service_test.go` skip helper integration cases if `tools/mtp/zimap-mtp` is absent. A passing `go test ./...` does not prove those mock-helper tests ran. Build the helper first when working on MTP, inspect skips with `go test -v`, and separately test real hardware. Mock mode is not evidence of USB transfer compatibility.

Test ownership: `app_test.go` covers catalog/output/storage/proxy/updater/BBBike integration; `app_outputs_test.go` covers output symlink boundaries and copy aliases; `app_device_test.go` covers the desktop device adapter. Each `internal/` package keeps its own tests. Frontend tests cover dictionary parity, utility/queue behavior and stale IPC-error callback guards (`App.test.jsx`), **not a mounted native GUI or complete browser interaction flow**. Use isolated `ZIMAP_HOME` and mock device directories for destructive-path tests; environment/global-state tests should not be made parallel casually.

### Build/release files

- [wails.json](../wails.json): app output name/metadata and frontend install/build/dev commands. `wails dev` runs hot reload; `wails build` embeds production frontend assets.
- [Makefile](../Makefile): `make build-mtp`, `make test`, `make build`, `make release-local`. Backend tests under `make test` build the helper first; the convenience packaging flow is macOS-oriented.
- [scripts/build-mtp.sh](../scripts/build-mtp.sh): builds the helper for the target OS/architecture. On macOS it builds/links static libusb, writes `tools/mtp/zimap-mtp`, and copies it into an existing app bundle.
- [.github/workflows/release.yml](../.github/workflows/release.yml): macOS arm64/amd64 and Windows amd64 builds, ZIP packaging, GitHub publishing. Windows packaging currently contains the main EXE; macOS packaging includes the MTP helper. This workflow builds releases; it does not replace running the unit/race suites.
- Expected release assets: `ZiMap-macos-arm64.zip`, `ZiMap-macos-amd64.zip`, `ZiMap-windows-amd64.zip`. Preserve this naming contract with `app_updater.go`.
- Current version identity is split across `CurrentAppVersion` in `app.go`, `info.productVersion` in `wails.json`, the version badge in `README.md`, frontend version fallbacks (`App.jsx`, `Sidebar.jsx`, `UpdateModal.jsx`, `AboutView.jsx`, `SettingsView.jsx`), and version tests in `app_test.go`. Update all of them together when bumping the version.

## 9. Keep these boundaries intact

1. Keep CLI usable without starting Wails; do not put GUI dependencies into shared conversion/download/device code.
2. Keep success, error and cancellation distinct. A 100% progress callback alone is not proof of successful file publication.
3. Local output validation, archive extraction validation, device protection, downloader publication and updater replacement are **different** safety boundaries. Reusing a generic path/copy helper must not remove their specific validation or rollback behavior.
4. Preserve system-map protection, timestamp tolerance, storage reserve, safe-eject handling and both mounted/MTP branches when changing device code.
5. Preserve RU/EN parity, external-URL validation, HIG theme tokens and interactive/drag-region boundaries.
6. `.img` is the implemented output format. `.gmapi` and private standalone storage are backlog items, not existing abstractions to extend blindly.
7. Check the working tree before changing files. Existing uncommitted work may belong to the user or an earlier agent; do not reset it merely to get a clean diff.
