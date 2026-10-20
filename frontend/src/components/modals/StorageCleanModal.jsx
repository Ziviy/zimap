import React, { useState, useEffect } from 'react';
import { Trash2, RefreshCw, AlertTriangle, CheckSquare } from 'lucide-react';
import { formatBytes } from '../../utils/formatters';

export const StorageCleanModal = ({ onConfirm, onCancel, cleaning, metrics, t }) => {
  const [clearCache, setClearCache] = useState(true);
  const [clearTemp, setClearTemp] = useState(true);
  const [clearMaps, setClearMaps] = useState(false);
  const [clearTools, setClearTools] = useState(false);
  const [clearSettings, setClearSettings] = useState(false);

  const allSelected = clearCache && clearTemp && clearMaps && clearTools && clearSettings;
  const anySelected = clearCache || clearTemp || clearMaps || clearTools || clearSettings;
  const hasDestructive = clearMaps || clearTools || clearSettings;

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && !cleaning) onCancel();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [cleaning, onCancel]);

  const handleToggleAll = () => {
    if (allSelected) {
      setClearCache(true);
      setClearTemp(true);
      setClearMaps(false);
      setClearTools(false);
      setClearSettings(false);
    } else {
      setClearCache(true);
      setClearTemp(true);
      setClearMaps(true);
      setClearTools(true);
      setClearSettings(true);
    }
  };

  return (
    <div className="modal-overlay" onClick={() => !cleaning && onCancel()}>
      <div className="modal-sheet" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 460 }}>
        <div className="modal-header">
          <div className="modal-icon-destructive">
            <Trash2 size={24} />
          </div>
          <h3>{t('storage.modalTitle')}</h3>
          <p className="modal-description">{t('storage.modalConfirm')}</p>
        </div>

        <div style={{ display: 'flex', justifyContent: 'flex-end', width: '100%', marginBottom: 4 }}>
          <button
            type="button"
            className="btn-secondary btn-sm"
            onClick={handleToggleAll}
            disabled={cleaning}
            style={{ fontSize: 11.5 }}
          >
            <CheckSquare size={12} style={{ marginRight: 5 }} />
            {allSelected ? t('storage.resetDefaultSelectionBtn') : t('storage.selectAllBtn')}
          </button>
        </div>

        <div className="clean-options-list">
          <label className="clean-option-label">
            <input 
              type="checkbox" 
              className="clean-option-checkbox"
              checked={clearCache} 
              onChange={(e) => setClearCache(e.target.checked)}
              disabled={cleaning}
            />
            <div>
              <div style={{ fontWeight: 600 }}>{t('storage.cacheLabel')} ({formatBytes(metrics?.cacheSize || 0)})</div>
              <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>{t('storage.clearCacheCheck')}</div>
            </div>
          </label>
          <label className="clean-option-label">
            <input 
              type="checkbox" 
              className="clean-option-checkbox"
              checked={clearTemp} 
              onChange={(e) => setClearTemp(e.target.checked)}
              disabled={cleaning}
            />
            <div>
              <div style={{ fontWeight: 600 }}>{t('storage.tempLabel')} ({formatBytes(metrics?.tempSize || 0)})</div>
              <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>{t('storage.clearTempCheck')}</div>
            </div>
          </label>
          <label className="clean-option-label">
            <input 
              type="checkbox" 
              className="clean-option-checkbox"
              checked={clearMaps} 
              onChange={(e) => setClearMaps(e.target.checked)}
              disabled={cleaning}
            />
            <div>
              <div style={{ fontWeight: 600, color: clearMaps ? 'var(--system-red)' : 'inherit' }}>
                {t('storage.mapsLabel')} ({formatBytes(metrics?.mapsSize || 0)})
              </div>
              <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>{t('storage.clearMapsCheck')}</div>
            </div>
          </label>
          <label className="clean-option-label">
            <input 
              type="checkbox" 
              className="clean-option-checkbox"
              checked={clearTools} 
              onChange={(e) => setClearTools(e.target.checked)}
              disabled={cleaning}
            />
            <div>
              <div style={{ fontWeight: 600, color: clearTools ? 'var(--system-red)' : 'inherit' }}>
                {t('storage.toolsLabel')} ({formatBytes(metrics?.toolsSize || 0)})
              </div>
              <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>{t('storage.clearToolsCheck')}</div>
            </div>
          </label>
          <label className="clean-option-label">
            <input 
              type="checkbox" 
              className="clean-option-checkbox"
              checked={clearSettings} 
              onChange={(e) => setClearSettings(e.target.checked)}
              disabled={cleaning}
            />
            <div>
              <div style={{ fontWeight: 600, color: clearSettings ? 'var(--system-red)' : 'inherit' }}>
                {t('storage.settingsLabel')}
              </div>
              <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>{t('storage.clearSettingsCheck')}</div>
            </div>
          </label>
        </div>

        {hasDestructive && (
          <div className="clean-destructive-warning">
            <AlertTriangle size={14} style={{ flexShrink: 0, marginTop: 1 }} />
            <span>{t('storage.destructiveWarning')}</span>
          </div>
        )}

        <div className="modal-actions">
          <button 
            className="btn-secondary modal-btn" 
            onClick={onCancel}
            disabled={cleaning}
          >
            {t('buttons.cancel')}
          </button>
          <button 
            className="btn-destructive modal-btn" 
            onClick={() => onConfirm({ clearCache, clearTemp, clearMaps, clearTools, clearSettings })}
            disabled={cleaning || !anySelected}
          >
            {cleaning ? (
              <RefreshCw size={14} className="spin" style={{ marginRight: 6 }} />
            ) : (
              <Trash2 size={14} style={{ marginRight: 6 }} />
            )}
            {t('storage.cleanConfirmBtn')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default StorageCleanModal;
