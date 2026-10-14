import React, { useState, useEffect } from 'react';
import { Trash2, RefreshCw } from 'lucide-react';
import { formatBytes } from '../../utils/formatters';

export const StorageCleanModal = ({ onConfirm, onCancel, cleaning, metrics, t }) => {
  const [clearCache, setClearCache] = useState(true);
  const [clearTemp, setClearTemp] = useState(true);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && !cleaning) onCancel();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [cleaning, onCancel]);

  return (
    <div className="modal-overlay" onClick={() => !cleaning && onCancel()}>
      <div className="modal-sheet" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-icon-destructive">
            <Trash2 size={24} />
          </div>
          <h3>{t('storage.modalTitle')}</h3>
          <p className="modal-description">{t('storage.modalConfirm')}</p>
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
        </div>

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
            onClick={() => onConfirm(clearCache, clearTemp)}
            disabled={cleaning || (!clearCache && !clearTemp)}
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
