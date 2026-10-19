# Архитектурный и технический план: Задача #52 — Автоматизация сборки релизов через GitHub Actions и интеграция с In-App Updater

> **Статус**: План утвержден к реализации  
> **Версия спецификации**: 1.0.0  
> **Целевые платформы**: macOS (Apple Silicon arm64 & Intel amd64), Windows (x86-64)  
> **Связанные компоненты**: `.github/workflows/release.yml`, `app_updater.go`, `app.go`, `wails.json`, `Makefile`, `cmd/zimap-mtp`  
> **Автор**: Lead Architect / Senior Software Engineer  

---

## 1. Контекст, цели и пользовательские сценарии

### 1.1. Контекст
В приложении **ZiMap Downloader** уже реализован полнофункциональный механизм самообновления ([`app_updater.go`](../../app_updater.go)), включающий:
* Проверку новых версий через GitHub REST API (`/repos/Ziviy/zimap/releases/latest`);
* Семантическое сравнение версий (`isVersionNewer`, `parseSemVer`);
* Интеллектуальный скоринг и выбор подходящего бинарного ассета (`scoreAsset`);
* Потоковую фоновую загрузку с метриками скорости, процентов и расчетного времени (ETA);
* Автоматическую распаковку архива во временный каталог;
* Безопасную атомарную замену бандла `.app` на macOS с очисткой атрибута карантина Gatekeeper (`xattr -rd com.apple.quarantine`) и замену `.exe` на Windows;
* Перезапуск обновленного приложения в один клик (`RestartApp`).

Для завершения полного цикла доставки продукта (CD) необходимо настроить полностью автоматизированную облачную сборку и публикацию релизов на GitHub при создании git-тега.

### 1.2. Цели задачи
1. Создать воспроизводимый, независимый от локальной среды пайплайн GitHub Actions (`.github/workflows/release.yml`).
2. Обеспечить матричную сборку под две ключевые платформы:
   * **macOS**: `arm64` (Apple Silicon M1..M4) и `amd64` (Intel), с обязательным встраиванием CLI-сайдкара `zimap-mtp` в `Contents/MacOS/`.
   * **Windows**: `amd64` (64-битный исполняемый файл).
3. Обеспечить 100% совместимость имен генерируемых архивов с алгоритмом подбора ассетов в `app_updater.go`.
4. Автоматизировать генерацию Release Notes на основе коммитов и pull request'ов.

---

## 2. Анализ архитектуры и совместимость с In-App Updater

### 2.1. Требования GitHub API и видимость репозитория
* **Публичность репозитория**:  
  Клиентский код в [`app_updater.go`](../../app_updater.go#L224-L241) выполняет неавторизованный GET-запрос к `https://api.github.com/repos/Ziviy/zimap/releases/latest`. Если репозиторий является приватным (Private), GitHub возвращает код `404 Not Found`. Для работы клиентского автообновления репозиторий должен быть **Public**.
* **Статус релиза (Draft / Pre-release)**:  
  Эндпоинт `/releases/latest` в GitHub REST API возвращает **только** опубликованные, стабильные релизы. Если релиз помечен как «Draft» или «Pre-release», API вернет `404`. Пайплайн обязан публиковать релизы с флагами `draft: false` и `prerelease: false`.

### 2.2. Матрица именования артефактов и скоринг (`scoreAsset`)
Функция `scoreAsset(name, goos, goarch)` в `app_updater.go` ранжирует доступные ассеты релиза по баллам:

| ОС клиента | Архитектура | Целевое имя ассета в релизе | Баллы скоринга | Содержимое архива |
| :--- | :--- | :--- | :--- | :--- |
| **macOS** | `arm64` (Apple Silicon) | `ZiMap-darwin-arm64.zip` | **25** (+10 darwin, +10 arm64, +5 zip) | `ZiMap.app` (включая `Contents/MacOS/zimap-mtp`) |
| **macOS** | `amd64` (Intel) | `ZiMap-darwin-amd64.zip` | **25** (+10 darwin, +10 amd64, +5 zip) | `ZiMap.app` (включая `Contents/MacOS/zimap-mtp`) |
| **Windows** | `amd64` (x86-64) | `ZiMap-windows-amd64.zip` | **25** (+10 windows, +10 amd64, +5 zip) | `ZiMap.exe` |

> [!IMPORTANT]
> **Особенности архивации macOS:**
> Бандл `.app` должен архивироваться исключительно утилитой `/usr/bin/ditto` с флагами `-c -k --keepParent`. Обычный Linux `zip` или Python `shutil` повреждает файловые права macOS (Unix permissions `0755`), символические ссылки Frameworks и внутренние ресурсы бандла. На клиенте распаковка также выполняется через `ditto -xk`.

---

## 3. Архитектура рабочего процесса GitHub Actions

### 3.1. Структурная схема конвейера CI/CD

```
 [Git Tag Push: v0.0.2] or [Manual Workflow Dispatch]
                         │
        ┌────────────────┴────────────────┐
        ▼                                 ▼
┌───────────────────────┐       ┌───────────────────────┐
│   Job: build-macos    │       │  Job: build-windows   │
│   Runner: macos-14    │       │  Runner: windows-2022 │
│   (Matrix: arm64/x64) │       │  (Target: amd64)      │
├───────────────────────┤       ├───────────────────────┤
│ 1. Go 1.25 + Node 22  │       │ 1. Go 1.25 + Node 22  │
│ 2. Install Wails v2   │       │ 2. Install Wails v2   │
│ 3. Build zimap-mtp    │       │ 3. npm ci && build    │
│ 4. wails build darwin │       │ 4. wails build win    │
│ 5. Embed zimap-mtp    │       │ 5. Compress-Archive   │
│ 6. ditto zip bundle   │       │                       │
└───────────┬───────────┘       └───────────┬───────────┘
            │                               │
            └───────────────┬───────────────┘
                            ▼
              ┌───────────────────────────┐
              │   Job: publish-release    │
              │   Runner: ubuntu-latest   │
              ├───────────────────────────┤
              │ 1. Download all artifacts │
              │ 2. Generate Release Notes │
              │ 3. softprops/action-gh-   │
              │    release (v2)           │
              │ 4. Publish GitHub Release │
              └───────────────────────────┘
```

---

## 4. Спецификация файла `.github/workflows/release.yml`

```yaml
name: Release ZiMap

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      tag_name:
        description: 'Release tag name (e.g. v0.0.2)'
        required: false
        default: ''

permissions:
  contents: write

jobs:
  build-macos:
    name: Build macOS (${{ matrix.arch }})
    runs-on: macos-14
    strategy:
      fail-fast: false
      matrix:
        arch: [arm64, amd64]
    steps:
      - name: Checkout Source Code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go Runtime
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true

      - name: Setup Node.js Runtime
        uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install Wails CLI
        run: go install github.com/wailsapp/wails/v2/cmd/wails@latest

      - name: Install Frontend Dependencies
        run: cd frontend && npm ci

      - name: Compile MTP CLI Sidecar (zimap-mtp)
        run: |
          mkdir -p tools/mtp
          CGO_ENABLED=0 GOOS=darwin GOARCH=${{ matrix.arch }} go build -ldflags="-s -w" -o tools/mtp/zimap-mtp ./cmd/zimap-mtp
          chmod +x tools/mtp/zimap-mtp

      - name: Build macOS Application Bundle
        run: |
          wails build -platform darwin/${{ matrix.arch }} -clean
          
          # Поиск бандла и встраивание CLI-сайдкара
          for app in build/bin/ZiMap.app build/bin/zimap.app; do
            if [ -d "$app/Contents/MacOS" ]; then
              echo "Embedding zimap-mtp into $app/Contents/MacOS/..."
              cp tools/mtp/zimap-mtp "$app/Contents/MacOS/zimap-mtp"
              chmod +x "$app/Contents/MacOS/zimap-mtp"
            fi
          done

      - name: Archive macOS Bundle with Ditto
        run: |
          APP_PATH=""
          if [ -d "build/bin/ZiMap.app" ]; then APP_PATH="build/bin/ZiMap.app"; fi
          if [ -d "build/bin/zimap.app" ]; then APP_PATH="build/bin/zimap.app"; fi
          if [ -z "$APP_PATH" ]; then
            echo "Error: Application bundle not found in build/bin!"
            exit 1
          fi
          ditto -c -k --keepParent "$APP_PATH" "ZiMap-darwin-${{ matrix.arch }}.zip"

      - name: Upload macOS Artifact
        uses: actions/upload-artifact@v4
        with:
          name: macos-${{ matrix.arch }}
          path: ZiMap-darwin-${{ matrix.arch }}.zip
          retention-days: 1

  build-windows:
    name: Build Windows (amd64)
    runs-on: windows-2022
    steps:
      - name: Checkout Source Code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go Runtime
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true

      - name: Setup Node.js Runtime
        uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install Wails CLI
        run: go install github.com/wailsapp/wails/v2/cmd/wails@latest

      - name: Install Frontend Dependencies
        run: cd frontend && npm ci

      - name: Build Windows Executable
        run: wails build -platform windows/amd64 -clean

      - name: Package Windows ZIP Archive
        shell: pwsh
        run: |
          $exePath = "build/bin/ZiMap.exe"
          if (-not (Test-Path $exePath)) {
            $exePath = "build/bin/zimap.exe"
          }
          if (-not (Test-Path $exePath)) {
            $exePath = "build/bin/garmin-map.exe"
          }
          if (-not (Test-Path $exePath)) {
            Write-Error "Windows executable not found in build/bin!"
            exit 1
          }
          Compress-Archive -Path $exePath -DestinationPath ZiMap-windows-amd64.zip

      - name: Upload Windows Artifact
        uses: actions/upload-artifact@v4
        with:
          name: windows-amd64
          path: ZiMap-windows-amd64.zip
          retention-days: 1

  publish-release:
    name: Publish GitHub Release
    needs: [build-macos, build-windows]
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Source Code
        uses: actions/checkout@v4

      - name: Download All Release Artifacts
        uses: actions/download-artifact@v4
        with:
          path: release-assets
          merge-multiple: true

      - name: Determine Release Tag
        id: release_tag
        run: |
          if [ "${{ github.event_name }}" = "workflow_dispatch" ] && [ -n "${{ github.event.inputs.tag_name }}" ]; then
            echo "tag=${{ github.event.inputs.tag_name }}" >> $GITHUB_OUTPUT
          else
            echo "tag=${{ github.ref_name }}" >> $GITHUB_OUTPUT
          fi

      - name: Create and Publish Release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ steps.release_tag.outputs.tag }}
          name: ZiMap ${{ steps.release_tag.outputs.tag }}
          draft: false
          prerelease: false
          generate_release_notes: true
          files: |
            release-assets/ZiMap-darwin-arm64.zip
            release-assets/ZiMap-darwin-amd64.zip
            release-assets/ZiMap-windows-amd64.zip
```

---

## 5. Пошаговый план внедрения (Фазы и задачи)

### Фаза 1: Подготовка конфигурации и скрипта сборки
- [ ] **1.1.** Создать каталог `.github/workflows/` в корне проекта.
- [ ] **1.2.** Зафиксировать спецификацию рабочего процесса в `.github/workflows/release.yml`.
- [ ] **1.3.** Добавить в `Makefile` цель `make release-local` для локального тестирования упаковки дистрибутивов перед отправкой тегов.
- [ ] **1.4.** Проверить работу `scripts/build-mtp.sh` при кросс-компиляции `GOARCH=amd64` и `GOARCH=arm64`.

### Фаза 2: Настройка окружения GitHub
- [ ] **2.1.** Проверить и изменить видимость репозитория на **Public** в GitHub Settings.
- [ ] **2.2.** Включить опцию **«Read and write permissions»** в `Settings` → `Actions` → `General` → `Workflow permissions`.
- [ ] **2.3.** Проверить права доступа токена `GITHUB_TOKEN` к созданию Releases.

### Фаза 3: Тестирование сквозного пайплайна
- [ ] **3.1.** Создать тестовый релиз-тег (например, `v0.0.2-test` или `v0.0.2`).
- [ ] **3.2.** Отследить выполнение workflow в интерфейсе GitHub Actions:
  - Успешная сборка macOS arm64;
  - Успешная сборка macOS amd64;
  - Успешная сборка Windows amd64;
  - Корректная публикация релиза с прикреплением трех `.zip` файлов.
- [ ] **3.3.** Запустить предыдущую версию ZiMap (`0.0.1 BETA`) и нажать **«Проверить обновления»**.
- [ ] **3.4.** Проверить автоматическое скачивание, распаковку, атомарную замену бандла и успешный перезапуск.

---

## 6. Регламент выпуска новых версий (Памятка разработчику)

Для выпуска новой версии приложения разработчику потребуется выполнить следующие действия:

```bash
# 1. Синхронизировать номер версии в app.go и wails.json (например, 0.0.2)
# 2. Зафиксировать изменения в git
git add app.go wails.json
git commit -m "Bump version to v0.0.2"
git push origin main

# 3. Установить и отправить тег
git tag v0.0.2
git push origin v0.0.2
```

После выполнения `git push origin v0.0.2`:
1. GitHub Actions автоматически запустит конвейер сборки под все платформы.
2. В течение ~4–6 минут соберет бинарники, упакует архивы и опубликует релиз `v0.0.2`.
3. Все запущенные клиентские копии ZiMap при следующем поиске обновлений предложат пользователям обновиться в 1 клик.
