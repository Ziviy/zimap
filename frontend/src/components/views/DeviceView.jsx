import React, { useState, useMemo } from 'react';
import { 
  Watch, 
  RefreshCw, 
  FolderOpen, 
  LogOut, 
  Trash2, 
  MapPin, 
  HardDrive, 
  AlertTriangle, 
  ShieldCheck, 
  CheckCircle2,
  ArrowUpCircle,
  PlusCircle,
  Layers,
  Compass,
  Laptop,
  Zap,
  StopCircle,
  Clock,
  Download,
  Lock,
  ChevronDown,
  ChevronUp
} from 'lucide-react';
import DeleteDeviceMapModal from '../modals/DeleteDeviceMapModal';
import { formatDate, formatBytes, formatEta, getRevealInfo } from '../../utils/formatters';

export const DeviceView = ({
  devices = [],
  usbStatus,
  loading = false,
  freshnessReport = null,
  freshnessLoading = false,
  syncState = null,
  onCheckFreshness,
  onSyncMap,
  onSyncAll,
  onCancelSync,
  onRefresh,
  onEject,
  onReveal,
  onDeleteMap,
  onNavigateMyMaps,
  onOpenURL,
  t
}) => {
  const [selectedMount, setSelectedMount] = useState(null);
  const [activeFilter, setActiveFilter] = useState('all');
  const [mapToDelete, setMapToDelete] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [ejecting, setEjecting] = useState(false);
  const [showPreinstalled, setShowPreinstalled] = useState(false);

  // Pick current device or first in list
  const activeDevice = (devices.find(d => d.mountPath === selectedMount)) || devices[0] || null;

  const handleConfirmDelete = async (map) => {
    if (!activeDevice || !onDeleteMap) return;
    setDeleting(true);
    try {
      await onDeleteMap(activeDevice.mountPath, map.filePath || map.devicePath);
      setMapToDelete(null);
    } catch (err) {
      console.error("Delete device map failed:", err);
    } finally {
      setDeleting(false);
    }
  };

  const handleEject = async () => {
    if (!activeDevice || !onEject) return;
    setEjecting(true);
    try {
      await onEject(activeDevice.mountPath, activeDevice.displayName || activeDevice.modelName);
    } finally {
      setEjecting(false);
    }
  };

  const handleTriggerFreshness = () => {
    if (activeDevice && onCheckFreshness) {
      onCheckFreshness(activeDevice.mountPath);
    }
  };

  const handleSyncSingle = (item) => {
    if (activeDevice && onSyncMap) {
      onSyncMap(activeDevice.mountPath, item.fileName);
    }
  };

  const handleSyncAllClick = () => {
    if (activeDevice && onSyncAll) {
      onSyncAll(activeDevice.mountPath);
    }
  };

  const revealInfo = getRevealInfo(t);

  // Filter items
  const filteredItems = useMemo(() => {
    if (!freshnessReport || !freshnessReport.items) return [];
    switch (activeFilter) {
      case 'updates':
        return freshnessReport.items.filter(
          item => item.status === 'update_available' || item.status === 'not_on_device'
        );
      case 'on-device':
        return freshnessReport.items.filter(
          item => item.status === 'up_to_date' || item.status === 'update_available' || item.status === 'only_on_device'
        );
      case 'missing':
        return freshnessReport.items.filter(
          item => item.status === 'not_on_device'
        );
      case 'all':
      default:
        return freshnessReport.items;
    }
  }, [freshnessReport, activeFilter]);

  const totalPendingSync = (freshnessReport?.updateAvailableCount || 0) + (freshnessReport?.notOnDeviceCount || 0);

  const formatDateTime = (dateVal) => {
    if (!dateVal) return '';
    try {
      const d = new Date(dateVal);
      if (isNaN(d.getTime())) return '';
      return d.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return '';
    }
  };

  return (
    <div className="view-container">
      {/* View Header */}
      <div className="view-header">
        <div>
          <h2>{t('header.device')}</h2>
          <p className="subtitle">{t('header.deviceSub')}</p>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          {devices.length > 1 && (
            <select 
              className="apple-select"
              value={activeDevice?.mountPath || ''}
              onChange={(e) => setSelectedMount(e.target.value)}
              style={{ fontSize: 13, height: 32 }}
            >
              {devices.map(d => (
                <option key={d.mountPath} value={d.mountPath}>
                  {d.displayName || d.modelName} ({d.volumeName})
                </option>
              ))}
            </select>
          )}
          <button 
            className="btn-secondary" 
            onClick={onRefresh} 
            disabled={loading || ejecting || syncState?.syncing}
            style={{ height: 32 }}
          >
            <RefreshCw size={14} className={loading ? "spin" : ""} style={{ marginRight: 6 }} />
            {t('buttons.refresh')}
          </button>
        </div>
      </div>

      {/* When no device is connected */}
      {!loading && !activeDevice && (
        <div className="device-empty-container">
          {usbStatus?.needsMtpNotice && (
            <div className="device-mtp-alert-card">
              <div className="device-mtp-icon">
                <AlertTriangle size={22} color="var(--system-orange)" />
              </div>
              <div className="device-mtp-content" style={{ flex: 1 }}>
                {usbStatus.mtpConflict ? (
                  <>
                    <h4>{t('device.conflictWarningTitle')}</h4>
                    <p>{t('device.conflictWarningDesc').replace('{app}', usbStatus.mtpConflict)}</p>
                  </>
                ) : (
                  <>
                    <h4>{usbStatus.isGarminMode ? t('device.garminModeNoticeTitle') : t('device.mtpNoticeTitle')}</h4>
                    <p>{usbStatus.isGarminMode ? t('device.garminModeNoticeDesc') : t('device.mtpNoticeDesc')}</p>
                  </>
                )}
                <div style={{ display: 'flex', gap: 8, marginTop: 12, flexWrap: 'wrap' }}>
                  <button 
                    className="btn-secondary btn-sm"
                    onClick={() => onOpenURL && onOpenURL('https://openmtp.ganeshrvel.com')}
                    style={{ fontSize: 12, height: 28 }}
                  >
                    <Download size={13} style={{ marginRight: 5 }} />
                    {t('device.openMtpDownload')}
                  </button>
                  {onNavigateMyMaps && (
                    <button 
                      className="btn-secondary btn-sm"
                      onClick={onNavigateMyMaps}
                      style={{ fontSize: 12, height: 28 }}
                    >
                      <FolderOpen size={13} style={{ marginRight: 5 }} />
                      {t('device.exportFromMyMaps')}
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          <div className="card device-empty-card">
            <div className="device-empty-icon-wrap">
              <Watch size={48} className="device-pulse-icon" />
            </div>
            <h3>{t('device.noDevice')}</h3>
            <p className="subtitle" style={{ maxWidth: 480, margin: '8px auto 20px' }}>
              {t('device.noDeviceSub')}
            </p>
            <button className="btn-primary" onClick={onRefresh} disabled={loading}>
              <RefreshCw size={14} className={loading ? "spin" : ""} style={{ marginRight: 6 }} />
              {t('buttons.refresh')}
            </button>
          </div>
        </div>
      )}

      {/* Loading state with no device yet */}
      {loading && !activeDevice && (
        <div className="card" style={{ padding: 48, textAlign: 'center' }}>
          <RefreshCw size={28} className="spin" style={{ color: 'var(--system-blue)', marginBottom: 12 }} />
          <div style={{ fontSize: 14, color: 'var(--text-secondary)' }}>
            {t('device.scanDevices')}
          </div>
        </div>
      )}

      {/* Connected Device Content */}
      {activeDevice && (
        <div className="device-content-grid">
          {/* Main Device Info Banner */}
          <div className="card device-hero-card">
            <div className="device-hero-top">
              <div className="device-hero-title-group">
                <div className="device-icon-badge">
                  <Watch size={28} />
                </div>
                <div>
                  <div className="device-name-row">
                    <span className="device-name">{activeDevice.displayName || activeDevice.modelName}</span>
                    {activeDevice.isMtp && (
                      <span className="device-meta-pill badge-mtp" title={t('device.mtpBadgeTooltip')}>
                        {t('device.mtpBadge')}
                      </span>
                    )}
                    <span className="device-connected-badge">
                      <span className="device-online-dot"></span>
                      {t('device.connected')}
                    </span>
                  </div>
                  <div className="device-meta-row">
                    {activeDevice.unitId && (
                      <span className="device-meta-pill">
                        {t('device.unitId')}: <strong>{activeDevice.unitId}</strong>
                      </span>
                    )}
                    {activeDevice.softwareVersion && (
                      <span className="device-meta-pill">
                        {t('device.softwareVersion')}: <strong>v{activeDevice.softwareVersion}</strong>
                      </span>
                    )}
                    <span className="device-meta-pill" title={activeDevice.mountPath}>
                      {t('device.mountPoint')}: <strong>{activeDevice.mountPath}</strong>
                    </span>
                  </div>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="device-hero-actions">
                {!activeDevice.isMtp && (
                  <button 
                    className="btn-secondary btn-sm" 
                    onClick={() => onReveal && onReveal(activeDevice.mountPath)}
                    title={revealInfo.tooltip}
                  >
                    <FolderOpen size={14} style={{ marginRight: 6 }} />
                    {revealInfo.label}
                  </button>
                )}
                <button 
                  className="btn-secondary btn-sm btn-eject" 
                  onClick={handleEject}
                  disabled={ejecting || syncState?.syncing}
                  title={activeDevice.isMtp ? t('device.disconnect') : t('device.eject')}
                >
                  {ejecting ? (
                    <RefreshCw size={14} className="spin" style={{ marginRight: 6 }} />
                  ) : (
                    <LogOut size={14} style={{ marginRight: 6 }} />
                  )}
                  {ejecting ? t('device.ejecting') : (activeDevice.isMtp ? t('device.disconnect') : t('device.eject'))}
                </button>
              </div>
            </div>

            {/* MTP Conflict Warning Banner */}
            {activeDevice.mtpConflict && (
              <div className="device-conflict-banner">
                <AlertTriangle size={16} color="var(--system-orange)" />
                <div style={{ flex: 1 }}>
                  <strong>{t('device.conflictWarningTitle')}: </strong>
                  <span>{t('device.conflictWarningDesc').replace('{app}', activeDevice.mtpConflict)}</span>
                </div>
              </div>
            )}

            {/* Storage Progress Section */}
            {activeDevice.storage && activeDevice.storage.totalBytes > 0 && (
              <div className="device-storage-section">
                <div className="device-storage-header">
                  <div className="device-storage-title">
                    <HardDrive size={15} style={{ marginRight: 6 }} />
                    <span>{t('device.storageTitle')}</span>
                  </div>
                  <div className="device-storage-stats">
                    <strong>{activeDevice.storage.formattedUsed}</strong> / {activeDevice.storage.formattedTotal}
                    <span className="device-storage-pct">
                      ({activeDevice.storage.usedPercentage.toFixed(1)}%)
                    </span>
                  </div>
                </div>

                <div className="device-storage-bar-track">
                  <div 
                    className="device-storage-bar-fill" 
                    style={{ width: `${Math.min(100, Math.max(2, activeDevice.storage.usedPercentage))}%` }}
                  ></div>
                </div>

                <div className="device-storage-legend">
                  <div className="storage-legend-item">
                    <span className="legend-dot used"></span>
                    <span>{t('device.storageUsed')}: {activeDevice.storage.formattedUsed}</span>
                  </div>
                  <div className="storage-legend-item">
                    <span className="legend-dot free"></span>
                    <span>{t('device.storageFree')}: {activeDevice.storage.formattedFree}</span>
                  </div>
                  <div className="storage-legend-item" style={{ marginLeft: 'auto', color: 'var(--text-tertiary)' }}>
                    <span>{t('device.storageTotal')}: {activeDevice.storage.formattedTotal}</span>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Freshness Toolbar (Task 20) */}
          <div className="device-freshness-toolbar">
            <div className="device-freshness-summary">
              {freshnessReport ? (
                <>
                  <span className="freshness-badge badge-fresh-uptodate" title={t('device.upToDate')}>
                    <CheckCircle2 size={13} />
                    <span>{freshnessReport.upToDateCount} {t('device.upToDate')}</span>
                  </span>
                  {freshnessReport.updateAvailableCount > 0 && (
                    <span className="freshness-badge badge-fresh-update" title={t('device.updateAvailable')}>
                      <ArrowUpCircle size={13} />
                      <span>{freshnessReport.updateAvailableCount} {t('device.updateAvailable')}</span>
                    </span>
                  )}
                  {freshnessReport.notOnDeviceCount > 0 && (
                    <span className="freshness-badge badge-fresh-missing" title={t('device.notOnDevice')}>
                      <PlusCircle size={13} />
                      <span>{freshnessReport.notOnDeviceCount} {t('device.notOnDevice')}</span>
                    </span>
                  )}
                  {freshnessReport.onlyOnDeviceCount > 0 && (
                    <span className="freshness-badge badge-fresh-deviceonly" title={t('device.onlyOnDevice')}>
                      <HardDrive size={13} />
                      <span>{freshnessReport.onlyOnDeviceCount} {t('device.onlyOnDevice')}</span>
                    </span>
                  )}
                </>
              ) : (
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, color: 'var(--text-secondary)' }}>
                  <Compass size={15} color="var(--system-blue)" />
                  <span>{t('device.checkFreshness')}</span>
                </div>
              )}
            </div>

            <div className="device-freshness-actions">
              <button 
                className="btn-secondary btn-sm"
                onClick={handleTriggerFreshness}
                disabled={freshnessLoading || syncState?.syncing}
                title={t('device.checkFreshness')}
              >
                <RefreshCw size={13} className={freshnessLoading ? "spin" : ""} style={{ marginRight: 6 }} />
                {freshnessLoading ? t('device.checkingFreshness') : t('device.checkFreshness')}
              </button>

              {totalPendingSync > 0 && (
                <button 
                  className="btn-sync-all"
                  onClick={handleSyncAllClick}
                  disabled={syncState?.syncing || freshnessLoading}
                  title={t('device.syncAllTitle')}
                >
                  <Zap size={14} />
                  {t('device.syncAllBtn').replace('{count}', totalPendingSync)}
                </button>
              )}
            </div>
          </div>

          {/* Filter Bar and Comparison List */}
          {freshnessReport && freshnessReport.items && freshnessReport.items.length > 0 ? (
            <>
              <div className="device-maps-section">
              <div className="device-filters-bar">
                <div className="segmented-control">
                  <button 
                    className={`segmented-btn ${activeFilter === 'all' ? 'active' : ''}`}
                    onClick={() => setActiveFilter('all')}
                  >
                    {t('device.filterAll').replace('{count}', freshnessReport.items.length)}
                  </button>
                  <button 
                    className={`segmented-btn ${activeFilter === 'updates' ? 'active' : ''}`}
                    onClick={() => setActiveFilter('updates')}
                  >
                    {t('device.filterUpdates').replace('{count}', totalPendingSync)}
                  </button>
                  <button 
                    className={`segmented-btn ${activeFilter === 'on-device' ? 'active' : ''}`}
                    onClick={() => setActiveFilter('on-device')}
                  >
                    {t('device.filterOnDevice').replace('{count}', 
                      (freshnessReport.upToDateCount || 0) + (freshnessReport.updateAvailableCount || 0) + (freshnessReport.onlyOnDeviceCount || 0)
                    )}
                  </button>
                  <button 
                    className={`segmented-btn ${activeFilter === 'missing' ? 'active' : ''}`}
                    onClick={() => setActiveFilter('missing')}
                  >
                    {t('device.filterMissing').replace('{count}', freshnessReport.notOnDeviceCount || 0)}
                  </button>
                </div>

                <div className="device-maps-badge-info">
                  <ShieldCheck size={14} color="var(--system-green)" style={{ marginRight: 4 }} />
                  <span>{t('device.systemMapsProtected')}</span>
                </div>
              </div>

              {/* Comparison Cards */}
              <div className="device-maps-list" style={{ marginTop: 12 }}>
                {filteredItems.map((item) => {
                  const isSyncingThis = syncState?.syncing && (syncState?.fileName === item.fileName || syncState?.mapId === item.fileName);

                  return (
                    <div 
                      key={item.id || item.fileName} 
                      className={`card device-compare-card ${isSyncingThis ? 'syncing' : ''}`}
                    >
                      <div className="device-compare-top">
                        <div className="device-compare-title-group">
                          <div className={`device-compare-icon-wrap status-${item.status.replace('_', '')} ${item.isPreinstalled ? 'status-uptodate' : ''}`}>
                            {item.isPreinstalled ? (
                              <ShieldCheck size={20} />
                            ) : (
                              <>
                                {item.status === 'update_available' && <ArrowUpCircle size={20} />}
                                {item.status === 'not_on_device' && <PlusCircle size={20} />}
                                {item.status === 'up_to_date' && <CheckCircle2 size={20} />}
                                {item.status === 'only_on_device' && <HardDrive size={20} />}
                              </>
                            )}
                          </div>
                          <div>
                            <div className="device-compare-title-row">
                              <span className="device-compare-title" title={item.mapName}>
                                {item.mapName}
                              </span>
                              <span className="format-badge">.IMG</span>
                              {item.isPreinstalled && (
                                <span className="badge-preinstalled" title={t('device.preinstalledTooltip')}>
                                  <ShieldCheck size={11} style={{ marginRight: 3, verticalAlign: -1 }} />
                                  {t('device.preinstalledTag')}
                                </span>
                              )}
                              <span className="device-meta-file">{item.fileName}</span>
                            </div>
                          </div>
                        </div>

                        {/* Status Badge */}
                        <div>
                          {item.isPreinstalled ? (
                            <span className="freshness-badge badge-fresh-preinstalled" title={t('device.preinstalledTooltip')}>
                              <ShieldCheck size={12} />
                              {t('device.preinstalledBadge')}
                            </span>
                          ) : (
                            <>
                              {item.status === 'up_to_date' && (
                                <span className="freshness-badge badge-fresh-uptodate">
                                  <CheckCircle2 size={12} />
                                  {t('device.upToDate')}
                                </span>
                              )}
                              {item.status === 'update_available' && (
                                <span className="freshness-badge badge-fresh-update">
                                  <ArrowUpCircle size={12} />
                                  {t('device.updateAvailable')}
                                </span>
                              )}
                              {item.status === 'not_on_device' && (
                                <span className="freshness-badge badge-fresh-missing">
                                  <PlusCircle size={12} />
                                  {t('device.notOnDevice')}
                                </span>
                              )}
                              {item.status === 'only_on_device' && (
                                <span className="freshness-badge badge-fresh-deviceonly">
                                  <HardDrive size={12} />
                                  {t('device.onlyOnDevice')}
                                </span>
                              )}
                            </>
                          )}
                        </div>
                      </div>

                      {/* Version Comparison Details Grid */}
                      <div className="device-compare-details">
                        {/* Computer Column */}
                        <div className="compare-col">
                          <div className="compare-col-header">
                            <Laptop size={12} />
                            <span>{t('device.onComputer')}</span>
                          </div>
                          {item.localSize > 0 ? (
                            <div className="compare-col-stats">
                              <span className="compare-size">{item.localSizeStr}</span>
                              <span className="meta-separator">•</span>
                              <span className="compare-date">{formatDateTime(item.localModTime)}</span>
                              {item.status === 'update_available' && (
                                <span className="compare-tag tag-fresh">{t('device.freshBuildNotice')}</span>
                              )}
                            </div>
                          ) : (
                            <div className="compare-col-stats">
                              <span className="compare-tag tag-none">—</span>
                            </div>
                          )}
                        </div>

                        {/* Device Column */}
                        <div className="compare-col">
                          <div className="compare-col-header">
                            <Watch size={12} />
                            <span>{t('device.onDevice')}</span>
                          </div>
                          {item.deviceSize > 0 ? (
                            <div className="compare-col-stats">
                              <span className="compare-size">{item.deviceSizeStr}</span>
                              <span className="meta-separator">•</span>
                              <span className="compare-date">{formatDateTime(item.deviceModTime)}</span>
                              {item.daysOutdated > 0 ? (
                                <span className="compare-tag tag-outdated">
                                  {t('device.outdatedBy').replace('{days}', item.daysOutdated)}
                                </span>
                              ) : item.status === 'update_available' ? (
                                <span className="compare-tag tag-outdated">
                                  {t('device.outdatedToday')}
                                </span>
                              ) : (
                                <span className="compare-tag tag-fresh">
                                  {t('device.upToDate')}
                                </span>
                              )}
                            </div>
                          ) : (
                            <div className="compare-col-stats">
                              <span className="compare-tag tag-none">{t('device.notOnDevice')}</span>
                            </div>
                          )}
                        </div>
                      </div>

                      {/* In-place Sync Progress Bar (if syncing this item) */}
                      {isSyncingThis ? (
                        <div className="card-sync-progress-wrap">
                          <div className="card-sync-progress-header">
                            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                              <RefreshCw size={13} className="spin" color="var(--system-blue)" />
                              <strong>{t('device.syncing')} {syncState.progress?.percentage || 0}%</strong>
                            </div>
                            <div className="card-sync-stats">
                              <span>
                                {syncState.progress?.speedFormatted || '...'}
                              </span>
                              {syncState.progress?.etaSeconds > 0 && (
                                <>
                                  <span className="meta-separator">•</span>
                                  <span>
                                    <Clock size={11} style={{ marginRight: 3, verticalAlign: -1 }} />
                                    ETA: {formatEta(syncState.progress.etaSeconds)}
                                  </span>
                                </>
                              )}
                              <button 
                                className="btn-secondary btn-sm" 
                                onClick={onCancelSync}
                                style={{ height: 24, padding: '0 8px', marginLeft: 8 }}
                                title={t('device.cancelSync')}
                              >
                                <StopCircle size={12} style={{ marginRight: 4 }} />
                                {t('buttons.cancel')}
                              </button>
                            </div>
                          </div>
                          <div className="card-sync-track">
                            <div 
                              className="card-sync-fill"
                              style={{ width: `${Math.max(2, syncState.progress?.percentage || 0)}%` }}
                            ></div>
                          </div>
                        </div>
                      ) : (
                        /* Normal Actions Row */
                        <div className="device-compare-actions-row">
                          <div className="device-compare-notice">
                            {item.isPreinstalled && (
                              <span style={{ color: 'var(--text-secondary)', display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                                <ShieldCheck size={13} color="var(--system-green)" />
                                {t('device.garminSignature')}
                              </span>
                            )}
                            {item.status === 'up_to_date' && !item.isPreinstalled && (
                              <span style={{ color: 'var(--text-secondary)' }}>
                                {t('device.identicalNotice')}
                              </span>
                            )}
                            {!item.hasEnoughSpace && (
                              <span style={{ color: 'var(--system-red)', display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                                <AlertTriangle size={13} />
                                {t('device.insufficientSpace')
                                  .replace('{needed}', formatBytes(item.bytesToTransfer + 20*1024*1024))
                                  .replace('{free}', activeDevice.storage?.formattedFree || '')}
                              </span>
                            )}
                          </div>

                          <div className="device-compare-btns">
                            {item.status === 'update_available' && !item.isPreinstalled && (
                              <button 
                                className="btn-secondary btn-sm btn-sync-action action-update"
                                onClick={() => handleSyncSingle(item)}
                                disabled={syncState?.syncing || !item.hasEnoughSpace}
                                title={t('device.updateOnDevice')}
                              >
                                <ArrowUpCircle size={13} style={{ marginRight: 5 }} />
                                {t('device.updateOnDevice')}
                              </button>
                            )}

                            {item.status === 'not_on_device' && (
                              <button 
                                className="btn-secondary btn-sm btn-sync-action"
                                onClick={() => handleSyncSingle(item)}
                                disabled={syncState?.syncing || !item.hasEnoughSpace}
                                title={t('device.syncToDevice')}
                              >
                                <PlusCircle size={13} style={{ marginRight: 5 }} />
                                {t('device.syncToDevice')}
                              </button>
                            )}

                            {item.devicePath && (
                              <>
                                {!activeDevice.isMtp && (
                                  <button 
                                    className="btn-secondary btn-sm"
                                    onClick={() => onReveal && onReveal(item.devicePath)}
                                    title={revealInfo.tooltip}
                                  >
                                    <FolderOpen size={13} style={{ marginRight: 5 }} />
                                    {revealInfo.label}
                                  </button>
                                )}
                                {item.isPreinstalled ? (
                                  <div 
                                    className="device-protected-badge"
                                    title={t('device.cannotDeletePreinstalled')}
                                  >
                                    <Lock size={12} style={{ marginRight: 4 }} />
                                    <span>{t('device.cannotDeletePreinstalled')}</span>
                                  </div>
                                ) : (
                                  <button 
                                    className="btn-secondary btn-sm btn-delete-device-map"
                                    onClick={() => setMapToDelete({ ...item, filePath: item.devicePath })}
                                    disabled={syncState?.syncing}
                                    title={t('device.deleteBtn')}
                                  >
                                    <Trash2 size={13} style={{ marginRight: 5 }} />
                                    {t('device.deleteBtn')}
                                  </button>
                                )}
                              </>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Collapsible Pre-installed Maps Section */}
            {((freshnessReport?.preinstalledMaps && freshnessReport.preinstalledMaps.length > 0) || 
              (activeDevice?.preinstalledMaps && activeDevice.preinstalledMaps.length > 0)) && (
              <div className="preinstalled-section">
                <div 
                  className="preinstalled-header-toggle" 
                  onClick={() => setShowPreinstalled(!showPreinstalled)}
                >
                  <div className="preinstalled-header-left">
                    <ShieldCheck size={16} color="var(--system-green)" />
                    <span>
                      {t('device.preinstalledTitle').replace('{count}', 
                        (freshnessReport?.preinstalledMaps || activeDevice?.preinstalledMaps || []).length
                      )}
                    </span>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                      {showPreinstalled ? (t('buttons.collapse') || 'Свернуть') : (t('buttons.expand') || 'Показать')}
                    </span>
                    {showPreinstalled ? <ChevronUp size={15} /> : <ChevronDown size={15} />}
                  </div>
                </div>

                {showPreinstalled && (
                  <div className="preinstalled-list">
                    {(freshnessReport?.preinstalledMaps || activeDevice?.preinstalledMaps || []).map((m) => (
                      <div key={m.fileName} className="preinstalled-item-row">
                        <div className="preinstalled-item-info">
                          <ShieldCheck size={14} color="var(--system-green)" style={{ flexShrink: 0 }} />
                          <span className="preinstalled-item-title" title={m.internalName || m.fileName}>
                            {m.internalName || m.fileName}
                          </span>
                          <span className="format-badge">.IMG</span>
                          <span className="badge-preinstalled">{t('device.preinstalledTag')}</span>
                          <span className="device-meta-file">{m.fileName}</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}>
                          <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                            {m.sizeFormatted || formatBytes(m.sizeBytes)}
                          </span>
                          <div className="device-protected-badge" title={t('device.cannotDeletePreinstalled')}>
                            <Lock size={11} style={{ marginRight: 3 }} />
                            <span>{t('device.preinstalledBadge')}</span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
          ) : (
            /* Fallback: If freshness report not loaded or empty */
            <div className="device-maps-section">
              <div className="device-maps-header">
                <div className="device-maps-title-row">
                  <Layers size={16} color="var(--system-blue)" />
                  <span className="device-maps-title">
                    {t('device.customMapsCount').replace('{count}', activeDevice.customMaps?.length || 0)}
                  </span>
                </div>
                <div className="device-maps-badge-info">
                  <ShieldCheck size={14} color="var(--system-green)" style={{ marginRight: 4 }} />
                  <span>{t('device.systemMapsProtected')}</span>
                </div>
              </div>

              {(!activeDevice.customMaps || activeDevice.customMaps.length === 0) ? (
                <div className="card device-empty-maps-card">
                  <div className="empty-maps-icon">
                    <Compass size={32} />
                  </div>
                  <h4>{t('device.noCustomMaps')}</h4>
                  <p>{t('device.noCustomMapsSub')}</p>
                </div>
              ) : (
                <div className="device-maps-list">
                  {activeDevice.customMaps.map((map) => (
                    <div key={map.filePath} className="card device-map-card">
                      <div className="device-map-left">
                        <div className="device-map-icon-wrap">
                          <MapPin size={18} />
                        </div>
                        <div className="device-map-info">
                          <div className="device-map-title-row">
                            <span className="device-map-title" title={map.internalName || map.fileName}>
                              {map.internalName || map.fileName.replace(/\.img$/i, '')}
                            </span>
                            <span className="format-badge">.IMG</span>
                          </div>
                          <div className="device-map-meta">
                            <span className="device-meta-file">{map.fileName}</span>
                            <span className="meta-separator">•</span>
                            <span>{map.sizeFormatted}</span>
                            <span className="meta-separator">•</span>
                            <span>{t('device.folder')}: {map.relativeDir}</span>
                            {map.modTime && (
                              <>
                                <span className="meta-separator">•</span>
                                <span>{t('device.modified')}: {formatDate(map.modTime)}</span>
                              </>
                            )}
                          </div>
                        </div>
                      </div>

                      <div className="device-map-actions">
                        {!activeDevice.isMtp && (
                          <button 
                            className="btn-secondary btn-sm"
                            onClick={() => onReveal && onReveal(map.filePath)}
                            title={revealInfo.tooltip}
                          >
                            <FolderOpen size={13} style={{ marginRight: 5 }} />
                            {revealInfo.label}
                          </button>
                        )}
                        <button 
                          className="btn-secondary btn-sm btn-delete-device-map"
                          onClick={() => setMapToDelete(map)}
                          title={t('device.deleteBtn')}
                        >
                          <Trash2 size={13} style={{ marginRight: 5 }} />
                          {t('device.deleteBtn')}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Delete Device Map Modal */}
      {mapToDelete && (
        <DeleteDeviceMapModal 
          map={mapToDelete}
          onConfirm={handleConfirmDelete}
          onCancel={() => !deleting && setMapToDelete(null)}
          deleting={deleting}
          t={t}
        />
      )}
    </div>
  );
};

export const GarminDeviceView = DeviceView;
export default DeviceView;
