# Архитектурный и технический план: Задача #20 — Проверка актуальности карт и синхронизация для Garmin Device

> **Статус**: План подготовлен к реализации  
> **Версия спецификации**: 1.0.0  
> **Целевые платформы**: macOS (Apple Silicon / Intel), Windows 10/11 (x86-64), Linux  
> **Связанные компоненты**: `internal/device`, `app_device.go`, `app_outputs.go`, `GarminDeviceView.jsx`, `App.jsx`, `i18n.js`  
> **Автор**: Lead Architect / Senior Software Engineer  

---

## 1. Введение и пользовательские сценарии

### 1.1. Контекст
В рамках Задачи #19 в **ZiMap Garmin Downloader** была разработана выделенная вкладка **«Устройство Garmin»** (`GarminDeviceView`), позволяющая обнаруживать подключенные навигаторы и часы Garmin (Fenix, Epix, Forerunner, Edge, GPSMAP), считывать метаданные устройства (`GarminDevice.xml`), анализировать емкость накопителя, отображать список пользовательских карт с их внутренними заголовками и безопасно удалять карты с защитой системных файлов производителя.

Однако до сих пор сохраняется критический разрыв в пользовательском опыте:
1. **Неизвестна актуальность**: Пользователь не знает, какая версия карты установлена на часах — устарела ли она по сравнению со свежей сборкой на компьютере или совпадает с ней.
2. **Ручной перенос файлов**: Чтобы обновить карту на часах или перенести свежесобранную карту из вкладки «Мои карты», пользователю приходится вручную открывать Finder / Проводник, искать том устройства, находить папку `GARMIN/` или `GARMIN/Map/`, копировать файл и надеяться, что на устройстве хватит свободного места.
3. **Риск повреждения FAT32**: При ручном копировании без принудительного сброса буферов (`fsync`) преждевременное извлечение кабеля USB приводит к повреждению файловой таблицы FAT32 или получению поврежденной карты («битый» `.img`), из-за чего часы могут циклически перезагружаться при загрузке карт.

### 1.2. Пользовательские сценарии (User Stories)
* **Сценарий A (Проверка в один клик)**: Подключив часы, пользователь нажимает кнопку **«Проверить актуальность»** (или анализ запускается автоматически при монтировании). Система сопоставляет локальные карты из каталога `output/` с картами на устройстве и визуально раскладывает их по статусам: *«Актуальна»*, *«Доступно обновление»*, *«Не на устройстве»*, *«Только на устройстве»*.
* **Сценарий B (Точечное обновление)**: Для карты со статусом *«Доступно обновление»* пользователь нажимает **«Обновить на устройстве»**. Приложение проверяет наличие свободного места, потоково копирует файл с отображением прогресса и скорости передачи, сбрасывает буферы на диск (`fsync`), обновляет время модификации и переводит статус в *«Актуальна»*.
* **Сценарий C (Первичная загрузка)**: Для карты, скомпилированной на ПК, но отсутствующей на часах (*«Не на устройстве»*), пользователь нажимает **«Загрузить на Garmin»**, и карта безопасно переносится в целевой каталог устройства.
* **Сценарий D (Пакетная синхронизация)**: При наличии нескольких устаревших карт пользователь нажимает **«Синхронизировать всё»**, запуская последовательную очередь обновления с общим индикатором прогресса.

---

## 2. Аппаратные и файловые нюансы Garmin (Technical Constraints)

При синхронизации данных с накопителями Garmin необходимо учитывать ключевые технические особенности:

### 2.1. Файловая система FAT32 и гранулярность времени (Timestamp Resolution)
* Большинство устройств Garmin отформатированы в **FAT32** (реже exFAT на устройствах с памятью 64 ГБ+).
* В FAT32 метка времени модификации файла (`mtime`) имеет **дискретность 2 секунды** (четные секунды).
* **Следствие**: Прямое сравнение времени через `time.Equal()` или строгий оператор `After` недопустимо. Если разница во времени между файлом на ПК и на часах составляет менее 2 секунд ($\Delta t \le 2\,\text{сек}$), карты считаются синхронизированными по времени.
* **Управление метками времени (`os.Chtimes`)**: При обычном копировании файла операционная система присваивает целевому файлу текущее время (`time.Now()`). Если не скорректировать метку времени, карта на часах станет «новее», чем исходник на ПК! **Обязательное требование**: после завершения записи файла на устройство необходимо вызвать `os.Chtimes(targetPath, srcModTime, srcModTime)`, синхронизируя время устройства с временем локального билда.

### 2.2. Скорость интерфейса USB и буферизация
* Подавляющее большинство часов Garmin (включая Fenix 7/8, Epix 2) используют контроллер **USB 2.0 Full Speed / High Speed** с реальной скоростью записи от **4 до 12 МБ/с**.
* Карта размером 1.5 ГБ передается от 2 до 5 минут.
* **Требование к UX**: Необходим детальный потоковый трекинг прогресса: переданные байты, общий размер, процент, текущая скорость передачи (МБ/с) и расчетное оставшееся время (ETA).
* **Требование к отмене**: Длительное копирование должно поддерживать прерывание по инициативе пользователя (кнопка «Отмена» с контекстом `context.WithCancel`).

### 2.3. Атомарность записи и защита от сбоев
* Прямая перезапись рабочего файла `.img` опасна: если пользователь случайно заденет кабель во время записи, на устройстве останется поврежденный файл.
* **Паттерн безопасной записи**:
  1. Копирование во временный файл в том же каталоге устройства: `<targetPath>.tmp-zimap`.
  2. Вызов `file.Sync()` для принудительного сброса аппаратных кэшей контроллера флеш-памяти.
  3. Закрытие файла.
  4. Атомарное переименование `os.Rename(tempPath, targetPath)`.
  5. Установка времени модификации через `os.Chtimes`.
  6. В случае ошибки или отмены — гарантированное удаление временного файла `.tmp-zimap`.

### 2.4. Расчет свободного места (Pre-flight Capacity Check)
* Перед началом копирования система обязана проверить:
  $$\text{FreeBytes} \ge \text{MapSizeBytes} + \text{SafetyMargin}$$
  где $\text{SafetyMargin} = 20\,\text{МБ}$ (запас для служебных логов активности Garmin, треков и файлов `.fit`).
* Если карта обновляется поверх существующей (перезапись), учитывается освобождаемый размер:
  $$\text{EffectiveNeeded} = \text{NewMapSize} - \text{ExistingMapSize}$$
  *(с учетом того, что при атомарном копировании через `.tmp-zimap` на короткое время требуется свободное место для нового файла целиком, либо при жесткой нехватке места — предупреждение пользователя)*.

---

## 3. Матрица статусов и алгоритм сопоставления (Correlation Engine)

### 3.1. Статусы свежести (Freshness Statuses)

| Статус | Код | Бейдж / Цвет | Описание | Доступные действия |
| :--- | :--- | :--- | :--- | :--- |
| **Актуальна** | `up_to_date` | 🟢 Зеленый (`CheckCircle2`) | Карта на часах идентична или новее карты на ПК ($\Delta t \le 2\,\text{с}$ или на часах новее, размеры совпадают). | Показать в Finder / Проводнике, Удалить |
| **Доступно обновление** | `update_available` | 🟠 Оранжевый (`ArrowUpCircle`) | Локальная карта на ПК была пересобрана позже, чем установлена на устройстве ($t_{\text{local}} > t_{\text{device}} + 2\,\text{с}$). | **«Обновить на устройстве»**, Показать, Удалить |
| **Не на устройстве** | `not_on_device` | 🔵 Синий (`PlusCircle`) | Карта скомпилирована в `output/` на ПК, но отсутствует на подключенном Garmin. | **«Загрузить на Garmin»**, Экспорт |
| **Только на устройстве** | `only_on_device` | ⚪ Серый (`HardDrive`) | Пользовательская карта установлена на часах, но отсутствует в локальной папке `output/`. | Показать на устройстве, Удалить с часов |

### 3.2. Алгоритм сопоставления карт (Matching Strategy)

Каждая карта на ПК представлена структурой `OutputItem` (имя файла, размер, `ModTime`, `Meta` с `regionID` и `sourceURL`).  
Каждая пользовательская карта на устройстве представлена структурой `DeviceMap` (имя файла, относительный путь, `InternalName`, размер, `ModTime`).

Сопоставление выполняется в 3 этапа:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        3-Tier Matching Algorithm                       │
├────────────────────────────────────────────────────────────────────────┤
│ 1. Точное совпадение по нормализованному имени файла                  │
│    strings.EqualFold(local.Name, device.FileName)                      │
│    (например, "russia-central.img" == "russia-central.img")            │
├────────────────────────────────────────────────────────────────────────┤
│ 2. Сопоставление по внутреннему заголовку или RegionID                │
│    Если имя файла отличается, но meta.RegionID или meta.Name           │
│    совпадает с internalName (без расширения) или cleanName             │
├────────────────────────────────────────────────────────────────────────┤
│ 3. Специальная обработка контейнера "gmapsupp.img"                    │
│    Если на устройстве присутствует gmapsupp.img, сверяем его внутрен-  │
│    ний заголовок с локальными картами для установления точного родства │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Архитектура бэкенда (Go)

### 4.1. Модели данных (`internal/device/sync_types.go`)

```go
package device

import "time"

type FreshnessStatus string

const (
    StatusUpToDate        FreshnessStatus = "up_to_date"
    StatusUpdateAvailable FreshnessStatus = "update_available"
    StatusNotOnDevice     FreshnessStatus = "not_on_device"
    StatusOnlyOnDevice    FreshnessStatus = "only_on_device"
)

// MapFreshnessItem представляет результат сравнения одной карты
type MapFreshnessItem struct {
    ID             string          `json:"id"`             // Уникальный ключ карты (slug / filename)
    MapName        string          `json:"mapName"`        // Читаемое название карты
    FileName       string          `json:"fileName"`       // Имя файла на диске (.img)
    Status         FreshnessStatus `json:"status"`         // up_to_date | update_available | not_on_device | only_on_device
    
    // Данные локальной карты на ПК (nil если only_on_device)
    LocalPath      string          `json:"localPath,omitempty"`
    LocalSize      int64           `json:"localSize,omitempty"`
    LocalSizeStr   string          `json:"localSizeStr,omitempty"`
    LocalModTime   time.Time       `json:"localModTime,omitempty"`
    
    // Данные карты на устройстве Garmin (nil если not_on_device)
    DevicePath     string          `json:"devicePath,omitempty"`
    DeviceSize     int64           `json:"deviceSize,omitempty"`
    DeviceSizeStr  string          `json:"deviceSizeStr,omitempty"`
    DeviceModTime  time.Time       `json:"deviceModTime,omitempty"`
    DeviceDir      string          `json:"deviceDir,omitempty"` // "GARMIN" или "GARMIN/Map"
    InternalName   string          `json:"internalName,omitempty"`
    
    // Вычисляемые параметры синхронизации
    BytesToTransfer int64          `json:"bytesToTransfer"`
    HasEnoughSpace  bool           `json:"hasEnoughSpace"`
    StatusMessage   string         `json:"statusMessage"`
}

// DeviceFreshnessReport агрегированный отчет для фронтенда
type DeviceFreshnessReport struct {
    MountPath         string             `json:"mountPath"`
    DeviceName        string             `json:"deviceName"`
    CheckedAt         time.Time          `json:"checkedAt"`
    Items             []MapFreshnessItem `json:"items"`
    UpToDateCount     int                `json:"upToDateCount"`
    UpdateAvailableCount int             `json:"updateAvailableCount"`
    NotOnDeviceCount  int                `json:"notOnDeviceCount"`
    OnlyOnDeviceCount int                `json:"onlyOnDeviceCount"`
    TotalSyncBytes    int64              `json:"totalSyncBytes"`
    DeviceFreeBytes   uint64             `json:"deviceFreeBytes"`
    DeviceFreeStr     string             `json:"deviceFreeStr"`
}

// SyncProgressEvent событие потокового прогресса в Wails IPC
type SyncProgressEvent struct {
    MapID          string  `json:"mapId"`
    FileName       string  `json:"fileName"`
    BytesCopied    int64   `json:"bytesCopied"`
    TotalBytes     int64   `json:"totalBytes"`
    Percentage     int     `json:"percentage"`
    SpeedBytesSec  int64   `json:"speedBytesSec"`
    SpeedFormatted string  `json:"speedFormatted"`
    ETASeconds     int     `json:"etaSeconds"`
}
```

### 4.2. Алгоритм сравнения (`internal/device/freshness.go`)

```go
func CompareMaps(deviceMaps []DeviceMap, localMaps []OutputItem, deviceFreeBytes uint64) []MapFreshnessItem
```
Логика сравнения:
1. Создается карта сопоставлений по нормализованному ключу (без расширения `.img`, в нижнем регистре).
2. Для каждого `OutputItem`:
   - Поиск в `DeviceMap`.
   - Если найдено:
     - Разница времени: $\Delta t = \text{local.ModTime} - \text{device.ModTime}$.
     - Если $\Delta t > 2\,\text{секунды}$ $\rightarrow$ `StatusUpdateAvailable`.
     - Иначе $\rightarrow$ `StatusUpToDate`.
   - Если не найдено $\rightarrow$ `StatusNotOnDevice`.
3. Для оставшихся `DeviceMap`, не сопоставленных ни с одним локальным файлом:
   - Присваивается статус `StatusOnlyOnDevice`.
4. Для каждого элемента рассчитывается `HasEnoughSpace` с учетом доступной памяти устройства и запаса в 20 МБ.
5. Результаты сортируются: сначала требующие внимания (`update_available`, затем `not_on_device`, затем `up_to_date`, затем `only_on_device`).

### 4.3. Движок потоковой синхронизации (`internal/device/sync.go`)

```go
func (s *SyncManager) SyncFile(
    ctx context.Context, 
    srcPath string, 
    destPath string, 
    progressFn func(SyncProgressEvent),
) error
```

Ключевые этапы функции `SyncFile`:
1. Проверка существования и доступности `srcPath`.
2. Валидация безопасности пути `destPath` через `ValidateDestinationPath` (предотвращение выхода за пределы тома, запрет перезаписи системных файлов).
3. Проверка свободного места на устройстве.
4. Открытие исходного файла и создание временного целевого файла:
   ```go
   tempDest := destPath + ".tmp-zimap"
   ```
5. Потоковое копирование блоками по 128 КБ (`131072 bytes`) с контекстом отмены (`ctx.Done()`) и подсчетом скорости/ETA скользящим средним.
6. Вызов `tempFile.Sync()` для принудительной очистки кэша контроллера накопителя.
7. Закрытие обоих дескрипторов.
8. Атомарное перемещение `os.Rename(tempDest, destPath)`.
9. Установка исходного времени модификации:
   ```go
   _ = os.Chtimes(destPath, srcStat.ModTime(), srcStat.ModTime())
   ```
10. Удаление временного файла в блоке `defer` в случае любой ошибки или отмены.

### 4.4. Интеграция в App (`app_device.go`)

Методы, экспортируемые во фронтенд через Wails IPC:
* `CheckDeviceFreshness(mountPath string) (*device.DeviceFreshnessReport, error)`
* `SyncMapToDevice(mountPath string, fileName string) error` (запуск в фоновой горутине с эмиссией событий `device:sync_progress`, `device:sync_done`, `device:sync_error`)
* `SyncAllOutdatedMaps(mountPath string) error` (последовательная синхронизация всех карт `update_available` и `not_on_device`)
* `CancelDeviceSync() error` (прерывание текущего копирования через отмену контекста)

---

## 5. Архитектура фронтенда (React 19 & Apple HIG)

### 5.1. Обновление компонента `GarminDeviceView.jsx`

#### A. Панель управления актуальностью (Freshness Hero Toolbar)
* В блоке устройства добавляется кнопка **«Проверить актуальность»** с иконкой `RefreshCw`.
* При наличии отчета отображается сводный бейдж-бар:
  - 🟢 **X** актуально
  - 🟠 **Y** доступно обновление
  - 🔵 **Z** не на часах
  - ⚪ **W** только на часах
* Кнопка быстрой синхронизации: **«Синхронизировать всё (Y+Z)»** (`Zap` или `RefreshCw` иконка), если есть карты, требующие переноса.

#### B. Фильтры отображения (Segmented Control)
Вкладки фильтрации списка карт:
1. **Все карты** (`all`)
2. **Требуют обновления** (`updates`) — показывает только `update_available` и `not_on_device`
3. **На устройстве** (`on-device`) — `up_to_date`, `update_available`, `only_on_device`
4. **Не на устройстве** (`missing`) — `not_on_device`

#### C. Карточка карты со сравнением версий (Comparison Card)
Каждая карточка наглядно отображает состояние:
```
┌────────────────────────────────────────────────────────────────────────┐
│ 📍 Центральный ФО (russia-central.img)               [🟠 Обновление]   │
│                                                                        │
│   Компьютер: 14 окт 2026, 11:30 • 512 МБ  (Свежая сборка)              │
│   Устройство: 02 сен 2026, 18:20 • 490 МБ  (Устарела на 42 дня)        │
│                                                                        │
│   [Обновить на устройстве]  [Показать в Finder]  [Удалить с часов]     │
└────────────────────────────────────────────────────────────────────────┘
```

#### D. Интерактивный прогресс передачи (In-place Transfer Ring / Bar)
* При нажатии «Обновить на устройстве» кнопка превращается в круговой индикатор Apple HIG (`CircularProgress`) или карточка отображает анимированную полосу передачи:
  - Процент выполнения (например, `45%`).
  - Метрики: `230 МБ / 512 МБ (8.2 МБ/с) • ETA: 34 сек`.
  - Кнопка немедленной остановки (Stop / Cancel).

### 5.2. Стилистика и токены (`AppleTheme.css`)
* Использование системных CSS-переменных проекта:
  - `--system-green`, `--system-orange`, `--system-blue`, `--system-gray`.
  - Матовые фоны с эффектом размытия (`backdrop-filter: blur(20px)`).
  - Специфические классы для бейджей: `.badge-fresh-uptodate`, `.badge-fresh-update`, `.badge-fresh-missing`, `.badge-fresh-deviceonly`.

---

## 6. Локализация (`i18n.js`)

Полная поддержка двух языков (RU / EN) со 100% паритетом ключей:

```javascript
// Русская локализация
freshness: {
  checkBtn: "Проверить актуальность",
  checking: "Проверка карт...",
  syncAllBtn: "Синхронизировать всё ({count})",
  syncAllTitle: "Обновить все устаревшие и недостающие карты",
  upToDate: "Актуальна",
  updateAvailable: "Доступно обновление",
  notOnDevice: "Не на устройстве",
  onlyOnDevice: "Только на часах",
  filterAll: "Все ({count})",
  filterUpdates: "Обновления ({count})",
  filterOnDevice: "На устройстве ({count})",
  filterMissing: "Не на часах ({count})",
  onComputer: "На компьютере",
  onDevice: "На устройстве",
  syncToDevice: "Загрузить на Garmin",
  updateOnDevice: "Обновить на устройстве",
  syncing: "Передача...",
  syncSuccess: "Карта «{name}» успешно синхронизирована с устройством!",
  syncError: "Ошибка синхронизации карты «{name}»: {err}",
  insufficientSpace: "Недостаточно свободного места на устройстве. Требуется: {needed}, свободно: {free}",
  outdatedBy: "Устарела на {days} дн.",
  identicalNotice: "Версия на устройстве полностью совпадает с локальной сборкой.",
  speedEta: "{speed} • Осталось: {eta}",
}

// English localization
freshness: {
  checkBtn: "Check Freshness",
  checking: "Checking maps...",
  syncAllBtn: "Sync All ({count})",
  syncAllTitle: "Update all outdated and missing maps",
  upToDate: "Up to date",
  updateAvailable: "Update Available",
  notOnDevice: "Not on Device",
  onlyOnDevice: "Only on Device",
  filterAll: "All ({count})",
  filterUpdates: "Updates ({count})",
  filterOnDevice: "On Device ({count})",
  filterMissing: "Not on Device ({count})",
  onComputer: "On Computer",
  onDevice: "On Device",
  syncToDevice: "Copy to Garmin",
  updateOnDevice: "Update on Device",
  syncing: "Transferring...",
  syncSuccess: "Map '{name}' successfully synced to Garmin device!",
  syncError: "Failed to sync map '{name}': {err}",
  insufficientSpace: "Insufficient storage on device. Required: {needed}, available: {free}",
  outdatedBy: "Outdated by {days} days",
  identicalNotice: "Map on device is fully up to date with local build.",
  speedEta: "{speed} • ETA: {eta}",
}
```

---

## 7. Обработка граничных ситуаций и безопасность

1. **Внезапное отключение кабеля USB во время копирования**:
   - Поскольку запись ведется во временный файл `<name>.tmp-zimap`, на часах не остается поврежденного файла основной карты.
   - При следующем сканировании или очистке хранилища (Task #48) временные файлы `.tmp-zimap` автоматически обнаруживаются и удаляются.
2. **Нехватка свободного места**:
   - Проверка свободного места выполняется **до** открытия файлов на запись.
   - Если размер превышает доступный объем памяти, операция блокируется с выводом предупреждающего Toast-сообщения с точными цифрами в МБ/ГБ.
3. **Попытка перезаписи системной карты**:
   - `ValidateDestinationPath` строго сверяет целевое имя с черным списком (`IsSystemMap`). Попытка перезаписать `gmapbmap.img` или тайлы TopoActive немедленно отклоняется с ошибкой `ErrSystemMapProtected`.
4. **Конкурентные операции**:
   - Мьютекс `syncMutex` предотвращает одновременный запуск двух процессов копирования на одно и то же устройство, исключая порчу файловой системы.

---

## 8. План поэтапной реализации

### Фаза 1: Бэкенд и логика сравнения (`internal/device`)
- [ ] Создать `internal/device/sync_types.go` с типами отчетов и статусов.
- [ ] Реализовать `internal/device/freshness.go`: алгоритм сопоставления локальных и внешних карт с учетом допуска FAT32 ($\pm 2$ сек).
- [ ] Написать модульные тесты `internal/device/freshness_test.go` на все 4 статуса (`up_to_date`, `update_available`, `not_on_device`, `only_on_device`).

### Фаза 2: Движок безопасного копирования и синхронизации
- [ ] Создать `internal/device/sync.go`: безопасное потоковое копирование, расчет скорости передачи и ETA, атомарный `os.Rename`, `fsync` и `os.Chtimes`.
- [ ] Написать модульные тесты `internal/device/sync_test.go` (проверка успешного копирования, отмены контекста, очистки temp-файлов при сбое, сохранения mtime).

### Фаза 3: Интеграция с Wails API (`app_device.go`)
- [ ] Добавить методы `CheckDeviceFreshness`, `SyncMapToDevice`, `SyncAllOutdatedMaps`, `CancelDeviceSync` в `app_device.go`.
- [ ] Настроить потоковую эмиссию событий `device:sync_progress`, `device:sync_done`, `device:sync_error`.
- [ ] Написать интеграционные тесты в `app_device_test.go`.

### Фаза 4: Фронтенд UI и интеграция компонентов
- [ ] Добавить ключи локализации в `frontend/src/i18n.js` и проверить тест паритета `npm test`.
- [ ] Добавить стили карточек сравнения, бейджей актуальности и прогресс-бара копирования в `frontend/src/AppleTheme.css`.
- [ ] Расширить `GarminDeviceView.jsx`: кнопки проверки, фильтры сегментов, сравнительные карточки версий, индикаторы прогресса передачи и кнопка отмены.
- [ ] Связать состояние синхронизации в `App.jsx`.

### Фаза 5: Комплексное тестирование и верификация
- [ ] Запуск Go тестов без кэша: `go test -count=1 ./...`.
- [ ] Запуск тестов фронтенда: `cd frontend && npm test`.
- [ ] Сборка Vite фронтенда: `npm run build`.
- [ ] Сборка полного бандла: `wails build`.
- [ ] Обновление документации в `TODO.md` и `AGENTS.md`.

---

## 9. Критерии приемки (Definition of Done)

1. Кнопка **«Проверить актуальность»** в реальном времени сравнивает все карты с локальным каталогом `output/` и мгновенно выставляет статусы.
2. Карты со статусом `update_available` и `not_on_device` имеют рабочую кнопку синхронизации.
3. Процесс передачи отображает процент, скорость в МБ/с и расчетное оставшееся время.
4. После копирования дата модификации файла на часах совпадает с датой сборки на ПК, и карта сразу помечается как `up_to_date`.
5. Кнопка отмены моментально прерывает копирование и удаляет временный файл.
6. При попытке скопировать карту, превышающую свободную память часов, выводится понятное сообщение об ошибке.
7. Все Go и Vitest тесты проходят с результатом 100% success.
