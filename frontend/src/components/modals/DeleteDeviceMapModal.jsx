import React, { useEffect } from 'react';
import { Trash2, RefreshCw } from 'lucide-react';

export const DeleteDeviceMapModal = ({ map, onConfirm, onCancel, deleting, t }) => {
  useEffect(() => {
    if (!map) return undefined;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && !deleting) {
        onCancel();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [map, deleting, onCancel]);

  if (!map) return null;

  const displayName = map.internalName || map.fileName.replace(/\.img$/i, '');
  const confirmText = t('device.deleteConfirm')
    .replace('{name}', displayName)
    .replace('{fileName}', map.fileName);

  return (
    <div className="modal-overlay" onClick={() => !deleting && onCancel()}>
      <div className="modal-sheet" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-icon-destructive">
            <Trash2 size={24} />
          </div>
          <h3>{t('device.deleteTitle')}</h3>
          <p className="modal-description">{confirmText}</p>
          <div className="modal-file-badge">
            {map.fileName} • {map.sizeFormatted} • {map.relativeDir}
          </div>
        </div>
        <div className="modal-actions">
          <button 
            className="btn-secondary modal-btn" 
            onClick={onCancel}
            disabled={deleting}
          >
            {t('buttons.cancel')}
          </button>
          <button 
            className="btn-destructive modal-btn" 
            onClick={() => onConfirm(map)}
            disabled={deleting}
          >
            {deleting ? (
              <RefreshCw size={14} className="spin" style={{ marginRight: 6 }} />
            ) : (
              <Trash2 size={14} style={{ marginRight: 6 }} />
            )}
            {t('buttons.delete')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default DeleteDeviceMapModal;
