import React, { useState, useEffect } from 'react';
import { Bell, CheckCircle2, AlertCircle, RefreshCw, Trash2, Sparkles, FolderOpen } from 'lucide-react';
import { formatBytes } from '../../utils/formatters';
import * as WailsApp from '../../../wailsjs/go/main/App';

export const SettingsView = ({ 
  t, 
  lang, 
  setLang, 
  theme, 
  toggleTheme, 
  heap, 
  setHeap,
  defaultSource = 'geofabrik',
  setDefaultSource,
  detailLevel = 'standard',
  setDetailLevel,
  providers = [],
  notificationsEnabled,
  setNotificationsEnabled,
  storageMetrics,
  loadingMetrics,
  onRefreshMetrics,
  onOpenCleanModal,
  checkingUpdate,
  onCheckUpdate,
  onSimulateUpdate,
  appVersion
}) => {
  const [useSystemProxy, setUseSystemProxy] = useState(true);
  const [customProxy, setCustomProxy] = useState('');
  const [proxyStatus, setProxyStatus] = useState(null); // 'success' | 'error' | null
  const [savingProxy, setSavingProxy] = useState(false);
  const [testNotificationSent, setTestNotificationSent] = useState(false);

  const handleTestNotification = async () => {
    const title = t('settings.testTitle') || 'ZiMap Downloader';
    const body = t('settings.testBody') || 'Проверка системных уведомлений прошла успешно!';

    try {
      if (WailsApp.SendNotification) {
        await WailsApp.SendNotification(title, body);
      }
    } catch (err) {
      console.warn('Native notification error:', err);
    }

    if (window.Notification) {
      if (Notification.permission === 'granted') {
        try {
          new Notification(title, { body });
        } catch (e) {
          console.warn('Web Notification error:', e);
        }
      } else if (Notification.permission === 'default') {
        Notification.requestPermission().then(permission => {
          if (permission === 'granted') {
            try {
              new Notification(title, { body });
            } catch (e) {
              console.warn('Web Notification error:', e);
            }
          }
        }).catch(console.warn);
      }
    }

    setTestNotificationSent(true);
    setTimeout(() => setTestNotificationSent(false), 3000);
  };

  useEffect(() => {
    if (WailsApp.GetProxy) {
      WailsApp.GetProxy().then(cfg => {
        if (cfg) {
          setUseSystemProxy(cfg.useSystemProxy !== false);
          setCustomProxy(cfg.customProxy || '');
        }
      }).catch(console.error);
    }
  }, []);

  const handleSaveProxy = async () => {
    setSavingProxy(true);
    try {
      if (WailsApp.SetProxy) {
        await WailsApp.SetProxy(useSystemProxy, customProxy);
        setProxyStatus('success');
        setTimeout(() => setProxyStatus(null), 3000);
      }
    } catch (err) {
      console.error(err);
      setProxyStatus('error');
      setTimeout(() => setProxyStatus(null), 4000);
    } finally {
      setSavingProxy(false);
    }
  };

  return (
    <div className="view-container">
      <div className="view-header">
        <div>
          <h2>{t('header.settings')}</h2>
          <p className="subtitle">{t('header.settingsSub')}</p>
        </div>
      </div>
      
      <div className="settings-container">
        <div className="settings-section">
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.language')}</span>
            </div>
            <div className="settings-value">
              <div className="segmented-control">
                <button className={lang === 'ru' ? 'active' : ''} onClick={() => setLang('ru')}>Русский</button>
                <button className={lang === 'en' ? 'active' : ''} onClick={() => setLang('en')}>English</button>
              </div>
            </div>
          </div>
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.theme')}</span>
            </div>
            <div className="settings-value">
              <div className="segmented-control">
                <button className={theme === 'light' ? 'active' : ''} onClick={() => toggleTheme('light')}>{t('settings.themeLight')}</button>
                <button className={theme === 'dark' ? 'active' : ''} onClick={() => toggleTheme('dark')}>{t('settings.themeDark')}</button>
              </div>
            </div>
          </div>
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.notifications')}</span>
              <span className="settings-label-subtitle">{t('settings.notificationsSub')}</span>
            </div>
            <div className="settings-value" style={{ flexDirection: 'column', alignItems: 'flex-end', gap: 8 }}>
              <div className="segmented-control">
                <button 
                  className={notificationsEnabled ? 'active' : ''} 
                  onClick={() => setNotificationsEnabled && setNotificationsEnabled(true)}
                >
                  {lang === 'ru' ? 'Вкл' : 'Enabled'}
                </button>
                <button 
                  className={!notificationsEnabled ? 'active' : ''} 
                  onClick={() => setNotificationsEnabled && setNotificationsEnabled(false)}
                >
                  {lang === 'ru' ? 'Выкл' : 'Disabled'}
                </button>
              </div>
              <button 
                className="btn-secondary btn-sm" 
                onClick={handleTestNotification}
                disabled={testNotificationSent}
                title={t('settings.testNotification')}
              >
                <Bell size={13} style={{ marginRight: 6 }} />
                {testNotificationSent ? t('settings.testNotificationSent') : t('settings.testNotification')}
              </button>
              {testNotificationSent && (
                <div className="settings-status-msg success">
                  <CheckCircle2 size={13} style={{ marginRight: 4 }} /> {t('settings.testNotificationSent')}
                </div>
              )}
            </div>
          </div>
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.heap')}</span>
              <span className="settings-label-subtitle">{t('settings.heapSub')}</span>
            </div>
            <div className="settings-value">
              <select className="text-input" value={heap} onChange={(e) => setHeap(e.target.value)}>
                <option value="2">2 GB</option>
                <option value="4">4 GB</option>
                <option value="8">8 GB</option>
                <option value="16">16 GB</option>
                <option value="32">32 GB</option>
              </select>
            </div>
          </div>
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.defaultSource')}</span>
              <span className="settings-label-subtitle">{t('settings.defaultSourceSub')}</span>
            </div>
            <div className="settings-value">
              <select 
                className="text-input" 
                value={defaultSource} 
                onChange={(e) => setDefaultSource && setDefaultSource(e.target.value)}
              >
                {providers.map(p => (
                  <option key={p.id} value={p.id}>
                    {t(`providers.${p.id === 'osm-france' ? 'osmFrance' : p.id}`) || p.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('detailLevels.label')}</span>
              <span className="settings-label-subtitle">{t('detailLevels.standardSub')}</span>
            </div>
            <div className="settings-value">
              <select 
                className="text-input" 
                value={detailLevel} 
                onChange={(e) => setDetailLevel && setDetailLevel(e.target.value)}
              >
                <option value="standard">{t('detailLevels.standard')}</option>
                <option value="high">{t('detailLevels.high')}</option>
                <option value="topo">{t('detailLevels.topo')}</option>
              </select>
            </div>
          </div>
        </div>

        <div className="settings-section">
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.useSystemProxy')}</span>
              <span className="settings-label-subtitle">{t('settings.useSystemProxySub')}</span>
            </div>
            <div className="settings-value">
              <label className="settings-checkbox-label">
                <input 
                  type="checkbox" 
                  className="settings-checkbox" 
                  checked={useSystemProxy} 
                  onChange={(e) => setUseSystemProxy(e.target.checked)} 
                />
                <span>{useSystemProxy ? (lang === 'ru' ? 'Включено' : 'Enabled') : (lang === 'ru' ? 'Выключено' : 'Disabled')}</span>
              </label>
            </div>
          </div>
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.customProxy')}</span>
              <span className="settings-label-subtitle">{t('settings.customProxyPlaceholder')}</span>
            </div>
            <div className="settings-value" style={{ flexDirection: 'column', alignItems: 'flex-end', width: '100%', maxWidth: 360 }}>
              <div className="settings-input-row">
                <input 
                  type="text" 
                  className="text-input" 
                  placeholder="http://127.0.0.1:8080" 
                  value={customProxy} 
                  disabled={useSystemProxy}
                  onChange={(e) => setCustomProxy(e.target.value)} 
                />
                <button 
                  className="btn-primary btn-sm" 
                  onClick={handleSaveProxy}
                  disabled={savingProxy}
                >
                  {savingProxy ? <RefreshCw size={12} className="spin" /> : t('settings.proxySave')}
                </button>
              </div>
              {proxyStatus === 'success' && (
                <div className="settings-status-msg success">
                  <CheckCircle2 size={13} style={{ marginRight: 4 }} /> {t('settings.proxySaved')}
                </div>
              )}
              {proxyStatus === 'error' && (
                <div className="settings-status-msg error">
                  <AlertCircle size={13} style={{ marginRight: 4 }} /> {t('settings.proxyError')}
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="settings-section">
          <div className="settings-row" style={{ alignItems: 'flex-start' }}>
            <div className="settings-label">
              <span className="settings-label-title">{t('storage.title')}</span>
              <span className="settings-label-subtitle">{t('storage.subtitle')}</span>
            </div>
            <div className="settings-value" style={{ flexDirection: 'column', alignItems: 'stretch', width: '100%', maxWidth: 460 }}>
              <div className="storage-metrics-grid">
                <div className="storage-metric-card">
                  <span className="storage-metric-lbl">{t('storage.cacheLabel')}</span>
                  <span className="storage-metric-val">{formatBytes(storageMetrics?.cacheSize || 0)}</span>
                  <span className="storage-metric-desc">{t('storage.cacheDesc')}</span>
                </div>
                <div className="storage-metric-card">
                  <span className="storage-metric-lbl">{t('storage.tempLabel')}</span>
                  <span className="storage-metric-val">{formatBytes(storageMetrics?.tempSize || 0)}</span>
                  <span className="storage-metric-desc">{t('storage.tempDesc')}</span>
                </div>
                <div className="storage-metric-card">
                  <span className="storage-metric-lbl">{t('storage.mapsLabel')}</span>
                  <span className="storage-metric-val">{formatBytes(storageMetrics?.mapsSize || 0)}</span>
                  <span className="storage-metric-desc">{t('storage.mapsDesc')}</span>
                </div>
                <div className="storage-metric-card">
                  <span className="storage-metric-lbl">{t('storage.toolsLabel')}</span>
                  <span className="storage-metric-val">{formatBytes(storageMetrics?.toolsSize || 0)}</span>
                  <span className="storage-metric-desc">{t('storage.toolsDesc')}</span>
                </div>
              </div>
              {storageMetrics?.dataPath && (
                <div className="storage-datapath-row" title={storageMetrics.dataPath}>
                  <span className="storage-datapath-lbl">{t('storage.dataPathLabel')}</span>
                  <code className="storage-datapath-val">{storageMetrics.dataPath}</code>
                </div>
              )}
              <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 4, flexWrap: 'wrap' }}>
                <button
                  className="btn-secondary btn-sm"
                  onClick={() => WailsApp.RevealDataDirectory && WailsApp.RevealDataDirectory()}
                  title={storageMetrics?.dataPath || t('storage.openDataDirBtn')}
                >
                  <FolderOpen size={12} style={{ marginRight: 4 }} />
                  {t('storage.openDataDirBtn')}
                </button>
                <button 
                  className="btn-secondary btn-sm"
                  onClick={onRefreshMetrics}
                  disabled={loadingMetrics}
                  title={t('storage.recalculate')}
                >
                  <RefreshCw size={12} className={loadingMetrics ? 'spin' : ''} style={{ marginRight: 4 }} />
                  {t('storage.recalculate')}
                </button>
                <button 
                  className="btn-secondary btn-sm"
                  onClick={onOpenCleanModal}
                  style={{ color: 'var(--system-red)' }}
                  title={t('storage.cleanBtn')}
                >
                  <Trash2 size={12} style={{ marginRight: 4 }} />
                  {t('storage.cleanBtn')}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div className="settings-section">
          <div className="settings-row">
            <div className="settings-label">
              <span className="settings-label-title">{t('settings.aboutInfo')}</span>
              <span className="settings-label-subtitle">ZiMap Downloader {appVersion || '0.0.4'} • by Ziviy</span>
            </div>
            <div className="settings-value" style={{ display: 'flex', gap: 8 }}>
              {onSimulateUpdate && (
                <button 
                  className="btn-secondary btn-sm"
                  onClick={onSimulateUpdate}
                  title={t('update.simulateTitle')}
                >
                  <Sparkles size={12} style={{ marginRight: 6 }} />
                  {t('update.simulateBtn')}
                </button>
              )}
              <button 
                className="btn-secondary btn-sm"
                onClick={onCheckUpdate}
                disabled={checkingUpdate}
              >
                <RefreshCw size={12} className={checkingUpdate ? 'spin' : ''} style={{ marginRight: 6 }} />
                {checkingUpdate ? t('update.checking') : t('update.check')}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SettingsView;
