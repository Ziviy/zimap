import React, { useState, useEffect, useRef } from 'react';
import { getI18n } from './i18n';
import './AppleTheme.css';

import * as WailsApp from '../wailsjs/go/main/App';
import * as WailsRuntime from '../wailsjs/runtime/runtime';
import appIconUrl from './assets/appicon.png';

import { formatBytes, openExternalURL } from './utils/formatters';
import { createQueueItem, enqueueJob, scheduleNextJob, isMapQueued, removeJobFromQueue } from './utils/queue';

import Sidebar from './components/navigation/Sidebar';
import Toolbar from './components/navigation/Toolbar';
import CatalogView from './components/views/CatalogView';
import LocalConvertView from './components/views/LocalConvertView';
import ConsoleView from './components/views/ConsoleView';
import ToolsView from './components/views/ToolsView';
import OutputsView from './components/views/OutputsView';
import DeviceView from './components/views/DeviceView';
import SettingsView from './components/views/SettingsView';
import AboutView from './components/views/AboutView';

import UpdateModal from './components/modals/UpdateModal';
import StorageCleanModal from './components/modals/StorageCleanModal';
import ToastNotification from './components/common/ToastNotification';

function App() {
  const [currentView, setCurrentView] = useState('catalog');
  const [theme, setTheme] = useState(localStorage.getItem('zimap_theme') || 'light');
  const [searchQuery, setSearchQuery] = useState('');
  
  const [lang, setLang] = useState(localStorage.getItem('zimap_language') || 'ru');
  const [heap, setHeap] = useState(localStorage.getItem('zimap_heap') || '4');
  const t = getI18n(lang);

  // Providers & Default Source state (Task 12 & Task 38)
  const [defaultSource, setDefaultSource] = useState(localStorage.getItem('zimap_default_source') || 'geofabrik');
  const [activeProvider, setActiveProvider] = useState(localStorage.getItem('zimap_default_source') || 'geofabrik');
  // Map detail level state (Task 45): default detail preset for all local compilations
  const [detailLevel, setDetailLevel] = useState(localStorage.getItem('zimap_detail') || 'standard');
  const [providers, setProviders] = useState([
    { id: 'geofabrik', name: 'Geofabrik' },
    { id: 'osm-france', name: 'OpenStreetMap France' },
    { id: 'bbbike', name: 'BBBike (Города)' }
  ]);

  // Download Queue state (Task 35)
  const [downloadQueue, setDownloadQueue] = useState([]);

  // Job & Progress state
  const [jobRunning, setJobRunning] = useState(false);
  const [activeJob, setActiveJob] = useState(null); // { name: string, url: string, isLocal: boolean }
  const [completedJob, setCompletedJob] = useState(null);
  const [stage, setStage] = useState('');
  const [progress, setProgress] = useState(0);
  const [progressDetail, setProgressDetail] = useState(null);
  const [logs, setLogs] = useState([]);
  const [errorMsg, setErrorMsg] = useState('');

  // Tools state
  const [toolsStatus, setToolsStatus] = useState({ java: {}, mkgmap: {}, splitter: {}, installed: false });
  const [checkingTools, setCheckingTools] = useState(false);
  const [toolsUpdating, setToolsUpdating] = useState(false);
  const [toolsLogs, setToolsLogs] = useState([]);

  // Outputs state
  const [outputs, setOutputs] = useState([]);

  // Garmin Device state (Task 19)
  const [devices, setDevices] = useState([]);
  const [usbStatus, setUsbStatus] = useState(null);
  const [devicesLoading, setDevicesLoading] = useState(false);

  // Catalog State (lifted up to not refetch on view switch)
  const [catalog, setCatalog] = useState(null);
  const [catalogLoading, setCatalogLoading] = useState(false);
  
  // Notifications state (Task 8)
  const [notificationsEnabled, setNotificationsEnabled] = useState(
    () => localStorage.getItem('zimap_notifications_enabled') !== 'false'
  );

  const handleSetNotificationsEnabled = (enabled) => {
    setNotificationsEnabled(enabled);
    localStorage.setItem('zimap_notifications_enabled', enabled ? 'true' : 'false');
    if (enabled && window.Notification && Notification.permission === 'default') {
      Notification.requestPermission().catch(() => {});
    }
  };

  // Toast state (Tasks 17, 48, 42)
  const [toast, setToast] = useState(null);
  const toastTimerRef = useRef(null);

  const showToast = (message, type = 'info', duration = 3500) => {
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    setToast({ message, type });
    toastTimerRef.current = setTimeout(() => {
      setToast(null);
      toastTimerRef.current = null;
    }, duration);
  };

  // Self-update state (Task 17 & In-App Updater)
  const [appVersion, setAppVersion] = useState('0.0.1 BETA');
  const [checkingUpdate, setCheckingUpdate] = useState(false);
  const [updateInfo, setUpdateInfo] = useState(null);
  const [updateModalOpen, setUpdateModalOpen] = useState(false);
  const [updateStatus, setUpdateStatus] = useState({
    stage: 'idle',
    progress: 0,
    metrics: null,
    error: null,
  });

  useEffect(() => {
    if (WailsApp.GetAppVersion) {
      WailsApp.GetAppVersion().then(v => {
        if (v) setAppVersion(v);
      }).catch(err => console.warn('GetAppVersion error:', err));
    }
  }, []);

  const handleCheckUpdate = async () => {
    setCheckingUpdate(true);
    try {
      let info = null;
      if (WailsApp.CheckForUpdate) {
        info = await WailsApp.CheckForUpdate();
      }
      if (info && info.isNewer) {
        setUpdateInfo(info);
        setUpdateStatus({ stage: 'idle', progress: 0, metrics: null, error: null });
        setUpdateModalOpen(true);
      } else {
        const ver = info?.currentVersion || appVersion || '0.0.1 BETA';
        showToast(`${t('update.upToDate')} (${ver})`, 'success');
      }
    } catch (err) {
      console.error('Check update error:', err);
      showToast(`${t('update.checkError')}: ${err?.message || err}`, 'error');
    } finally {
      setCheckingUpdate(false);
    }
  };

  const handleSimulateUpdate = async () => {
    try {
      let info = null;
      if (WailsApp.SimulateUpdateCheck) {
        info = await WailsApp.SimulateUpdateCheck();
      }
      if (info) {
        setUpdateInfo(info);
        setUpdateStatus({ stage: 'idle', progress: 0, metrics: null, error: null });
        setUpdateModalOpen(true);
        showToast(t('update.simulateToast'), 'info');
      }
    } catch (err) {
      console.error('Simulate update error:', err);
      showToast(`Simulation error: ${err?.message || err}`, 'error');
    }
  };

  const handleStartUpdate = async (assetURL) => {
    setUpdateStatus({ stage: 'download', progress: 0, metrics: null, error: null });
    try {
      if (WailsApp.InstallUpdate) {
        await WailsApp.InstallUpdate(assetURL || 'simulate');
      }
    } catch (err) {
      console.error('InstallUpdate error:', err);
      setUpdateStatus(prev => ({ ...prev, stage: 'error', error: err?.message || String(err) }));
    }
  };

  const handleCancelUpdate = async () => {
    try {
      if (WailsApp.CancelUpdate) {
        await WailsApp.CancelUpdate();
      }
      setUpdateStatus(prev => ({ ...prev, stage: 'idle' }));
      showToast(t('update.cancelled'), 'warning');
    } catch (err) {
      console.warn('CancelUpdate error:', err);
    }
  };

  const handleRestartApp = async () => {
    try {
      if (WailsApp.RestartApp) {
        await WailsApp.RestartApp();
      }
    } catch (err) {
      console.error('RestartApp error:', err);
      showToast(`Restart error: ${err?.message || err}`, 'error');
    }
  };

  const handleDownloadUpdate = (url) => {
    setUpdateModalOpen(false);
    openExternalURL(url || 'https://github.com/Ziviy/zimap/releases');
  };

  // Storage Metrics & Cleaning state (Task 48)
  const [storageMetrics, setStorageMetrics] = useState({ cacheSize: 0, tempSize: 0, mapsSize: 0, totalSize: 0 });
  const [loadingMetrics, setLoadingMetrics] = useState(false);
  const [cleanModalOpen, setCleanModalOpen] = useState(false);
  const [cleaningStorage, setCleaningStorage] = useState(false);

  const refreshStorageMetrics = async () => {
    setLoadingMetrics(true);
    try {
      if (WailsApp.GetStorageMetrics) {
        const m = await WailsApp.GetStorageMetrics();
        if (m) setStorageMetrics(m);
      }
    } catch (err) {
      console.error('GetStorageMetrics error:', err);
    } finally {
      setLoadingMetrics(false);
    }
  };

  const handleConfirmCleanStorage = async (clearCache, clearTemp) => {
    setCleaningStorage(true);
    try {
      let freed = 0;
      if (WailsApp.ClearTempAndCacheFiles) {
        freed = await WailsApp.ClearTempAndCacheFiles(clearCache, clearTemp);
      }
      setCleanModalOpen(false);
      await refreshStorageMetrics();

      if (freed > 0) {
        showToast(t('storage.freedSuccess').replace('{size}', formatBytes(freed)), 'success');
      } else {
        showToast(t('storage.alreadyClean'), 'info');
      }

      if (clearCache) {
        loadCatalog(true, activeProvider);
      }
      if (clearTemp) {
        refreshOutputs();
      }
    } catch (err) {
      console.error('ClearTempAndCacheFiles error:', err);
      showToast(`${t('storage.cleanError')}: ${err?.message || err}`, 'error');
    } finally {
      setCleaningStorage(false);
    }
  };

  const activeJobRef = useRef(activeJob);
  useEffect(() => {
    activeJobRef.current = activeJob;
  }, [activeJob]);

  const langRef = useRef(lang);
  useEffect(() => {
    langRef.current = lang;
  }, [lang]);
  
  useEffect(() => {
    localStorage.setItem('zimap_language', lang);
  }, [lang]);

  useEffect(() => {
    localStorage.setItem('zimap_heap', heap);
  }, [heap]);

  useEffect(() => {
    localStorage.setItem('zimap_theme', theme);
  }, [theme]);

  useEffect(() => {
    localStorage.setItem('zimap_default_source', defaultSource);
  }, [defaultSource]);

  useEffect(() => {
    localStorage.setItem('zimap_detail', detailLevel);
  }, [detailLevel]);

  const toggleTheme = (newTheme) => {
    if (newTheme) {
      setTheme(newTheme);
    } else {
      setTheme(prev => prev === 'light' ? 'dark' : 'light');
    }
  };

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  const refreshOutputs = async () => {
    try {
      if (WailsApp.ListOutputs) {
        const list = await WailsApp.ListOutputs();
        setOutputs(list || []);
      }
    } catch (e) {
      console.error(e);
    }
  };

  // Device Freshness & Sync state (Task 20)
  const [deviceFreshnessReport, setDeviceFreshnessReport] = useState(null);
  const [deviceFreshnessLoading, setDeviceFreshnessLoading] = useState(false);
  const [deviceSyncState, setDeviceSyncState] = useState({
    syncing: false,
    mapId: '',
    fileName: '',
    progress: null,
  });

  const checkDeviceFreshness = async (mountPath) => {
    let targetMount = mountPath;
    if (!targetMount) {
      if (devices.length > 0) {
        targetMount = devices[0].mountPath;
      } else {
        return;
      }
    }
    setDeviceFreshnessLoading(true);
    try {
      if (WailsApp.CheckDeviceFreshness) {
        const report = await WailsApp.CheckDeviceFreshness(targetMount);
        setDeviceFreshnessReport(report);
      }
    } catch (err) {
      console.error("CheckDeviceFreshness error:", err);
      showToast(t('device.syncError').replace('{name}', '').replace('{err}', err?.message || err), 'error');
    } finally {
      setDeviceFreshnessLoading(false);
    }
  };

  const handleSyncMapToDevice = async (mountPath, fileName) => {
    try {
      if (WailsApp.SyncMapToDevice) {
        await WailsApp.SyncMapToDevice(mountPath, fileName);
      }
    } catch (err) {
      console.error("SyncMapToDevice error:", err);
      showToast(t('device.syncError').replace('{name}', fileName).replace('{err}', err?.message || err), 'error');
    }
  };

  const handleSyncAllOutdatedMaps = async (mountPath) => {
    try {
      if (WailsApp.SyncAllOutdatedMaps) {
        await WailsApp.SyncAllOutdatedMaps(mountPath);
      }
    } catch (err) {
      console.error("SyncAllOutdatedMaps error:", err);
      showToast(t('device.syncError').replace('{name}', 'All').replace('{err}', err?.message || err), 'error');
    }
  };

  const handleCancelDeviceSync = async () => {
    try {
      if (WailsApp.CancelDeviceSync) {
        await WailsApp.CancelDeviceSync();
      }
    } catch (err) {
      console.error("CancelDeviceSync error:", err);
    }
  };

  const loadDevices = async () => {
    setDevicesLoading(true);
    try {
      let devs = [];
      if (WailsApp.GetConnectedDevices) {
        devs = await WailsApp.GetConnectedDevices();
      } else if (WailsApp.GetConnectedGarminDevices) {
        devs = await WailsApp.GetConnectedGarminDevices();
      }
      setDevices(devs || []);
      if (WailsApp.GetDeviceUSBStatus) {
        const usb = await WailsApp.GetDeviceUSBStatus();
        setUsbStatus(usb);
      } else if (WailsApp.GetGarminUSBStatus) {
        const usb = await WailsApp.GetGarminUSBStatus();
        setUsbStatus(usb);
      }
      if (devs && devs.length > 0) {
        checkDeviceFreshness(devs[0].mountPath);
      } else {
        setDeviceFreshnessReport(null);
      }
    } catch (err) {
      console.error("Failed to scan devices:", err);
    } finally {
      setDevicesLoading(false);
    }
  };

  useEffect(() => {
    if (currentView === 'device') {
      loadDevices();
    }
  }, [currentView]);

  const handleEjectDevice = async (mountPath, deviceName) => {
    try {
      if (WailsApp.EjectDevice) {
        await WailsApp.EjectDevice(mountPath);
      } else if (WailsApp.EjectGarminDevice) {
        await WailsApp.EjectGarminDevice(mountPath);
      }
      showToast(t('device.ejectedToast').replace('{name}', deviceName || 'Device'), 'success');
      setDeviceFreshnessReport(null);
      await loadDevices();
    } catch (err) {
      console.error("Failed to eject device:", err);
      showToast(t('device.ejectError').replace('{err}', err?.message || err), 'error');
    }
  };

  const handleDeleteDeviceMap = async (mountPath, filePath) => {
    try {
      if (WailsApp.DeleteDeviceMap) {
        await WailsApp.DeleteDeviceMap(mountPath, filePath);
      }
      const fileName = filePath.split(/[/\\]/).pop();
      showToast(t('device.deleteSuccess').replace('{name}', fileName), 'success');
      await loadDevices();
      if (mountPath) {
        await checkDeviceFreshness(mountPath);
      }
    } catch (err) {
      console.error("Failed to delete device map:", err);
      showToast(t('device.deleteError').replace('{err}', err?.message || err), 'error');
      throw err;
    }
  };

  const handleRevealDevicePath = async (filePath) => {
    try {
      if (WailsApp.RevealDevicePath) {
        await WailsApp.RevealDevicePath(filePath);
      } else if (WailsApp.RevealFile) {
        await WailsApp.RevealFile(filePath);
      }
    } catch (err) {
      console.error("Failed to reveal device path:", err);
    }
  };

  const refreshTools = async (showFeedback = false) => {
    setCheckingTools(true);
    try {
      if (WailsApp.CheckTools) {
        const res = await WailsApp.CheckTools();
        const status = res || { java: {}, mkgmap: {}, splitter: {}, installed: false };
        setToolsStatus(status);
        if (showFeedback) {
          if (status.installed) {
            showToast(t('tools.allReadyToast'), 'success');
          } else {
            showToast(t('tools.needsSetupToast'), 'error');
          }
        }
      }
    } catch (e) {
      console.error(e);
      if (showFeedback) {
        showToast(e?.message || 'Error checking tools', 'error');
      }
    } finally {
      setCheckingTools(false);
    }
  };

  const loadCatalog = async (forceRefresh = false, providerId = activeProvider) => {
    setCatalogLoading(true);
    try {
      let data;
      if (forceRefresh && WailsApp.ForceRefreshCatalogByProvider) {
        data = await WailsApp.ForceRefreshCatalogByProvider(providerId);
      } else if (forceRefresh && WailsApp.ForceRefreshCatalog) {
        data = await WailsApp.ForceRefreshCatalog();
      } else if (WailsApp.GetCatalogByProvider) {
        data = await WailsApp.GetCatalogByProvider(providerId);
      } else if (WailsApp.GetCatalog) {
        data = await WailsApp.GetCatalog();
      }
      setCatalog(data);
    } catch (err) {
      console.error("Failed to load catalog:", err);
    } finally {
      setCatalogLoading(false);
    }
  };

  // Initial loads and event subscriptions
  useEffect(() => {
    refreshTools();
    refreshOutputs();
    refreshStorageMetrics();
    loadDevices();

    // Auto-load catalog immediately on startup from default source (Task 38)
    loadCatalog(false, defaultSource);

    if (WailsApp.GetProviders) {
      WailsApp.GetProviders().then(list => {
        if (list && list.length > 0) setProviders(list);
      }).catch(console.error);
    }

    // Event listeners
    if (WailsRuntime.EventsOn) {
      WailsRuntime.EventsOn("convert:stage", (st) => {
        setStage(st);
        setLogs(prev => [...prev, `[STAGE] -> ${st}`]);
      });

      WailsRuntime.EventsOn("convert:progress", (pct) => {
        setProgress(pct);
      });
      
      WailsRuntime.EventsOn("convert:progress_detail", (detail) => {
        setProgressDetail(detail);
      });

      WailsRuntime.EventsOn("convert:log", (line) => {
        setLogs(prev => [...prev, line]);
      });

      WailsRuntime.EventsOn("convert:done", () => {
        const completedJobInfo = activeJobRef.current;
        setCompletedJob(completedJobInfo);
        const completedJobName = completedJobInfo?.name;
        setJobRunning(false);
        setActiveJob(null);
        setStage('done');
        setProgress(100);
        setLogs(prev => [...prev, `[SUCCESS] Сборка карты завершена успешно!`]);
        refreshOutputs();
        refreshStorageMetrics();

        if (localStorage.getItem('zimap_notifications_enabled') !== 'false') {
          const currentLang = langRef.current || 'ru';
          const title = 'ZiMap Downloader';
          const body = completedJobName
            ? `${completedJobName}: ${currentLang === 'ru' ? 'Сборка карты завершена успешно!' : 'Map compiled successfully!'}`
            : (currentLang === 'ru' ? 'Сборка карты завершена успешно!' : 'Map compiled successfully!');

          if (typeof WailsApp.SendNotification === 'function') {
            WailsApp.SendNotification(title, body).catch(err => console.warn('SendNotification error:', err));
          }
          if (window.Notification) {
            if (Notification.permission === 'granted') {
              try {
                new Notification(title, { body, icon: appIconUrl });
              } catch (e) {
                console.warn('Web Notification error:', e);
              }
            } else if (Notification.permission === 'default') {
              Notification.requestPermission().then(permission => {
                if (permission === 'granted') {
                  try {
                    new Notification(title, { body, icon: appIconUrl });
                  } catch (e) {
                    console.warn('Web Notification error:', e);
                  }
                }
              }).catch(console.warn);
            }
          }
        }
      });

      WailsRuntime.EventsOn("convert:error", (errMsg) => {
        const failedJobName = activeJobRef.current?.name;
        setJobRunning(false);
        setActiveJob(null);
        setStage('error');
        setErrorMsg(errMsg);
        setLogs(prev => [...prev, `[ERROR] ${errMsg}`]);

        if (localStorage.getItem('zimap_notifications_enabled') !== 'false') {
          const currentLang = langRef.current || 'ru';
          const title = 'ZiMap Downloader';
          const body = failedJobName
            ? `${failedJobName}: ${errMsg || (currentLang === 'ru' ? 'Ошибка компиляции карты' : 'Map compilation failed')}`
            : (errMsg || (currentLang === 'ru' ? 'Ошибка компиляции карты' : 'Map compilation failed'));

          if (typeof WailsApp.SendNotification === 'function') {
            WailsApp.SendNotification(title, body).catch(err => console.warn('SendNotification error:', err));
          }
          if (window.Notification) {
            if (Notification.permission === 'granted') {
              try {
                new Notification(title, { body, icon: appIconUrl });
              } catch (e) {
                console.warn('Web Notification error:', e);
              }
            } else if (Notification.permission === 'default') {
              Notification.requestPermission().then(permission => {
                if (permission === 'granted') {
                  try {
                    new Notification(title, { body, icon: appIconUrl });
                  } catch (e) {
                    console.warn('Web Notification error:', e);
                  }
                }
              }).catch(console.warn);
            }
          }
        }
      });

      WailsRuntime.EventsOn("tools:log", (line) => {
        setToolsLogs(prev => [...prev, line]);
      });

      // Device Sync Events (Task 20)
      WailsRuntime.EventsOn("device:sync_start", (data) => {
        setDeviceSyncState({
          syncing: true,
          mapId: data?.mapId || data?.fileName || '',
          fileName: data?.fileName || '',
          progress: null,
        });
      });

      WailsRuntime.EventsOn("device:sync_progress", (data) => {
        setDeviceSyncState(prev => ({
          ...prev,
          syncing: true,
          mapId: data?.mapId || prev.mapId,
          fileName: data?.fileName || prev.fileName,
          progress: data,
        }));
      });

      WailsRuntime.EventsOn("device:sync_done", async (data) => {
        setDeviceSyncState({ syncing: false, mapId: '', fileName: '', progress: null });
        const name = data?.mapName || data?.fileName || '';
        showToast(t('device.syncSuccess').replace('{name}', name), 'success');
        let devs = [];
        if (WailsApp.GetConnectedDevices) {
          devs = await WailsApp.GetConnectedDevices();
        } else if (WailsApp.GetConnectedGarminDevices) {
          devs = await WailsApp.GetConnectedGarminDevices();
        }
        setDevices(devs || []);
        if (devs && devs.length > 0) {
          checkDeviceFreshness(devs[0].mountPath);
        }
      });

      WailsRuntime.EventsOn("device:sync_all_done", async (data) => {
        setDeviceSyncState({ syncing: false, mapId: '', fileName: '', progress: null });
        showToast(t('device.syncAllSuccess').replace('{count}', data?.total || ''), 'success');
        let devs = [];
        if (WailsApp.GetConnectedDevices) {
          devs = await WailsApp.GetConnectedDevices();
        } else if (WailsApp.GetConnectedGarminDevices) {
          devs = await WailsApp.GetConnectedGarminDevices();
        }
        setDevices(devs || []);
        if (devs && devs.length > 0) {
          checkDeviceFreshness(devs[0].mountPath);
        }
      });

      WailsRuntime.EventsOn("device:sync_error", (data) => {
        setDeviceSyncState({ syncing: false, mapId: '', fileName: '', progress: null });
        const name = data?.fileName || '';
        const errMsg = data?.error || '';
        showToast(t('device.syncError').replace('{name}', name).replace('{err}', errMsg), 'error');
      });

      WailsRuntime.EventsOn("device:sync_cancelled", () => {
        setDeviceSyncState({ syncing: false, mapId: '', fileName: '', progress: null });
        showToast(t('device.syncCancelled'), 'warning');
      });

      WailsRuntime.EventsOn("update:stage", (st) => {
        setUpdateStatus(prev => ({ ...prev, stage: st }));
      });

      WailsRuntime.EventsOn("update:progress", (pct) => {
        setUpdateStatus(prev => ({ ...prev, progress: pct }));
      });

      WailsRuntime.EventsOn("update:metrics", (m) => {
        setUpdateStatus(prev => ({ ...prev, metrics: m }));
      });

      WailsRuntime.EventsOn("update:done", () => {
        setUpdateStatus(prev => ({ ...prev, stage: 'ready', progress: 100 }));
      });

      WailsRuntime.EventsOn("update:error", (errMsg) => {
        setUpdateStatus(prev => ({ ...prev, stage: 'error', error: errMsg }));
      });
    }

    return () => {
      if (WailsRuntime.EventsOff) {
        WailsRuntime.EventsOff("convert:stage");
        WailsRuntime.EventsOff("convert:progress");
        WailsRuntime.EventsOff("convert:progress_detail");
        WailsRuntime.EventsOff("convert:log");
        WailsRuntime.EventsOff("convert:done");
        WailsRuntime.EventsOff("convert:error");
        WailsRuntime.EventsOff("tools:log");
        WailsRuntime.EventsOff("device:sync_start");
        WailsRuntime.EventsOff("device:sync_progress");
        WailsRuntime.EventsOff("device:sync_done");
        WailsRuntime.EventsOff("device:sync_all_done");
        WailsRuntime.EventsOff("device:sync_error");
        WailsRuntime.EventsOff("device:sync_cancelled");
        WailsRuntime.EventsOff("update:stage");
        WailsRuntime.EventsOff("update:progress");
        WailsRuntime.EventsOff("update:metrics");
        WailsRuntime.EventsOff("update:done");
        WailsRuntime.EventsOff("update:error");
      }
    };
  }, []);

  const handleSelectProvider = (providerId) => {
    setActiveProvider(providerId);
    loadCatalog(false, providerId);
  };

  const handleExportMap = async (pathOrName) => {
    try {
      if (WailsApp.ExportMap) {
        const exported = await WailsApp.ExportMap(pathOrName);
        if (exported) {
          showToast(t('buttons.exportSuccess').replace('{path}', exported), 'success');
        }
      }
    } catch (e) {
      console.error(e);
      showToast(e?.message || 'Export error', 'error');
    }
  };

  const handleUpdateTools = async () => {
    setToolsUpdating(true);
    setToolsLogs(['Запуск загрузки и обновления инструментов...']);
    try {
      if (WailsApp.UpdateTools) {
        await WailsApp.UpdateTools();
        setToolsLogs(prev => [...prev, 'Инструменты успешно обновлены!']);
        refreshTools();
      }
    } catch (e) {
      setToolsLogs(prev => [...prev, `Ошибка обновления: ${e}`]);
    } finally {
      setToolsUpdating(false);
    }
  };

  const executeBuild = (url, name, heapSize = '', detail = '') => {
    const job = { name, url, isLocal: false };
    activeJobRef.current = job;
    setActiveJob(job);
    setCompletedJob(null);
    setJobRunning(true);
    setStage('download');
    setProgress(0);
    setProgressDetail(null);
    setErrorMsg('');
    setLogs([`Запуск загрузки региона ${name}...`, `URL: ${url}`]);
    
    if (WailsApp.ConvertURL) {
      WailsApp.ConvertURL(url, name, heapSize, detail || detailLevel).catch(err => {
        if (activeJobRef.current !== job) return;
        setLogs(prev => [...prev, `[ERROR] ${err}`]);
        setJobRunning(false);
        setActiveJob(null);
      });
    }
  };

  const handleStartBuild = (url, name, heapSize = '', detail = '') => {
    if (jobRunning) {
      setDownloadQueue(prev => enqueueJob(prev, createQueueItem(url, name, heapSize, detail || detailLevel)));
      return;
    }
    executeBuild(url, name, heapSize, detail);
  };

  const handleRemoveFromQueue = (identifier) => {
    setDownloadQueue(prev => removeJobFromQueue(prev, identifier));
  };

  const handleClearQueue = () => {
    setDownloadQueue([]);
  };

  const isRegionQueued = (region) => {
    if (!region || !region.pbfUrl) return false;
    return downloadQueue.some(q => q.url === region.pbfUrl) || isMapQueued(region.name, downloadQueue);
  };

  // Auto-advance download queue (Task 35)
  useEffect(() => {
    if (!jobRunning && downloadQueue.length > 0) {
      return scheduleNextJob(downloadQueue, (nextJob, remainingQueue) => {
        executeBuild(nextJob.url, nextJob.name, nextJob.heapSize, nextJob.detailLevel);
        setDownloadQueue(remainingQueue);
      });
    }
  }, [jobRunning, downloadQueue]);

  const handleStartLocalBuild = (filePath, name, heapVal, detail = 'standard') => {
    const job = { name, url: filePath, isLocal: true };
    activeJobRef.current = job;
    setActiveJob(job);
    setCompletedJob(null);
    setJobRunning(true);
    setStage('splitter');
    setProgress(0);
    setLogs([`Запуск локальной конвертации ${name}...`, `Файл: ${filePath}`, `Память JVM: ${heapVal}`, `Уровень детализации: ${detail}`]);
    setCurrentView('console');

    if (WailsApp.ConvertLocal) {
      WailsApp.ConvertLocal(filePath, name, heapVal, detail).catch(err => {
        if (activeJobRef.current !== job) return;
        setLogs(prev => [...prev, `[ERROR] ${err}`]);
        setJobRunning(false);
        setActiveJob(null);
      });
    }
  };

  const handleUpdateMap = (item) => {
    let url = item.sourceUrl || item.meta?.sourceURL;
    const cleanName = item.name.replace(/\.img$/i, '');
    if (!url && catalog?.roots) {
      const findInNodes = (nodes) => {
        for (const n of nodes) {
          if (n.pbfUrl && (n.name.toLowerCase() === cleanName.toLowerCase() || n.id.toLowerCase() === cleanName.toLowerCase())) {
            return n.pbfUrl;
          }
          if (n.children && n.children.length > 0) {
            const found = findInNodes(n.children);
            if (found) return found;
          }
        }
        return null;
      };
      url = findInNodes(catalog.roots);
    }
    if (url) {
      handleStartBuild(url, cleanName);
    } else {
      setLogs(prev => [...prev, `[WARN] ${t('myMaps.sourceNotFound')} (${item.name})`]);
    }
  };

  const handleDeleteMap = async (path) => {
    if (WailsApp.DeleteOutput) {
      await WailsApp.DeleteOutput(path);
      await refreshOutputs();
    }
  };

  const handleCancel = async () => {
    if (WailsApp.CancelActiveJob) {
      await WailsApp.CancelActiveJob();
      setJobRunning(false);
      setActiveJob(null);
      setLogs(prev => [...prev, '[CANCELLED] Процесс был отменён пользователем']);
    }
  };

  const handleReveal = (path) => {
    if (WailsApp.RevealFile) {
      WailsApp.RevealFile(path);
    }
  };

  const handleNavigateConsole = () => {
    setCurrentView('console');
  };

  return (
    <div className="app-shell">
      <Sidebar 
        currentView={currentView} 
        setCurrentView={setCurrentView} 
        theme={theme}
        toggleTheme={() => toggleTheme()}
        jobRunning={jobRunning}
        toolsReady={toolsStatus.installed}
        deviceConnected={devices.length > 0}
        appVersion={appVersion}
        t={t}
      />
      <div className="main-content">
        <Toolbar 
          title={currentView === 'my-maps' ? t('nav.myMaps') : t('nav.' + currentView)}
          currentView={currentView}
          searchQuery={searchQuery}
          setSearchQuery={setSearchQuery}
          onSearchSubmit={() => {
            if (currentView !== 'catalog' && currentView !== 'my-maps') {
              setCurrentView('catalog');
            }
          }}
          t={t}
        />
        
        {currentView === 'catalog' && (
          <CatalogView 
            onStartBuild={handleStartBuild} 
            onCancel={handleCancel}
            onNavigateConsole={handleNavigateConsole}
            searchQuery={searchQuery}
            catalog={catalog}
            loading={catalogLoading}
            loadCatalog={loadCatalog}
            outputs={outputs}
            activeJob={activeJob}
            jobRunning={jobRunning}
            stage={stage}
            progress={progress}
            activeProvider={activeProvider}
            onSelectProvider={handleSelectProvider}
            providers={providers}
            isRegionQueued={isRegionQueued}
            onRemoveFromQueue={handleRemoveFromQueue}
            t={t}
          />
        )}
        
        {currentView === 'local' && (
          <LocalConvertView 
            onStartLocalBuild={handleStartLocalBuild} 
            t={t}
            detailLevel={detailLevel}
            setDetailLevel={setDetailLevel}
          />
        )}

        {currentView === 'console' && (
          <ConsoleView 
            logs={logs}
            setLogs={setLogs}
            stage={stage}
            progress={progress}
            progressDetail={progressDetail}
            jobRunning={jobRunning}
            onCancel={handleCancel}
            t={t}
            errorMsg={errorMsg}
            activeJob={activeJob}
            completedJob={completedJob}
            onExport={handleExportMap}
            onReveal={handleReveal}
          />
        )}

        {currentView === 'tools' && (
          <ToolsView 
            toolsStatus={toolsStatus}
            onRefreshTools={refreshTools}
            checkingTools={checkingTools}
            onUpdateTools={handleUpdateTools}
            toolsUpdating={toolsUpdating}
            toolsLogs={toolsLogs}
            t={t}
          />
        )}

        {(currentView === 'my-maps' || currentView === 'outputs') && (
          <OutputsView 
            outputs={outputs}
            searchQuery={searchQuery}
            onRefreshOutputs={refreshOutputs}
            onReveal={handleReveal}
            onUpdate={handleUpdateMap}
            onDelete={handleDeleteMap}
            onExport={handleExportMap}
            onNavigateConsole={handleNavigateConsole}
            jobRunning={jobRunning}
            activeJob={activeJob}
            stage={stage}
            progress={progress}
            progressDetail={progressDetail}
            onCancel={handleCancel}
            downloadQueue={downloadQueue}
            onRemoveFromQueue={handleRemoveFromQueue}
            onClearQueue={handleClearQueue}
            t={t}
          />
        )}

        {currentView === 'device' && (
          <DeviceView 
            devices={devices}
            usbStatus={usbStatus}
            loading={devicesLoading}
            freshnessReport={deviceFreshnessReport}
            freshnessLoading={deviceFreshnessLoading}
            syncState={deviceSyncState}
            onCheckFreshness={checkDeviceFreshness}
            onSyncMap={handleSyncMapToDevice}
            onSyncAll={handleSyncAllOutdatedMaps}
            onCancelSync={handleCancelDeviceSync}
            onRefresh={loadDevices}
            onEject={handleEjectDevice}
            onReveal={handleRevealDevicePath}
            onDeleteMap={handleDeleteDeviceMap}
            onNavigateMyMaps={() => setCurrentView('my-maps')}
            onOpenURL={openExternalURL}
            t={t}
          />
        )}
        
        {currentView === 'settings' && (
          <SettingsView
            t={t}
            lang={lang}
            setLang={setLang}
            theme={theme}
            toggleTheme={toggleTheme}
            heap={heap}
            setHeap={setHeap}
            defaultSource={defaultSource}
            setDefaultSource={setDefaultSource}
            detailLevel={detailLevel}
            setDetailLevel={setDetailLevel}
            providers={providers}
            notificationsEnabled={notificationsEnabled}
            setNotificationsEnabled={handleSetNotificationsEnabled}
            storageMetrics={storageMetrics}
            loadingMetrics={loadingMetrics}
            onRefreshMetrics={refreshStorageMetrics}
            onOpenCleanModal={() => setCleanModalOpen(true)}
            checkingUpdate={checkingUpdate}
            onCheckUpdate={handleCheckUpdate}
            onSimulateUpdate={handleSimulateUpdate}
            appVersion={appVersion}
          />
        )}

        {currentView === 'about' && (
          <AboutView 
            t={t} 
            checkingUpdate={checkingUpdate}
            onCheckUpdate={handleCheckUpdate}
            appVersion={appVersion}
          />
        )}
      </div>

      {updateModalOpen && (
        <UpdateModal 
          updateInfo={updateInfo}
          updateStatus={updateStatus}
          onStartUpdate={handleStartUpdate}
          onCancelUpdate={handleCancelUpdate}
          onRestartApp={handleRestartApp}
          onDownload={handleDownloadUpdate}
          onClose={() => setUpdateModalOpen(false)}
          appVersion={appVersion}
          t={t}
        />
      )}

      {cleanModalOpen && (
        <StorageCleanModal 
          onConfirm={handleConfirmCleanStorage}
          onCancel={() => setCleanModalOpen(false)}
          cleaning={cleaningStorage}
          metrics={storageMetrics}
          t={t}
        />
      )}

      <ToastNotification toast={toast} />
    </div>
  );
}

export default App;
