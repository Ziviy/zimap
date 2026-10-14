import React, { useEffect } from 'react';
import { Trash2, RefreshCw } from 'lucide-react';
import { formatBytes } from '../../utils/formatters';

export const DeleteConfirmationModal = ({ item, onConfirm, onCancel, deleting, t }) => {
  useEffect(() => {
    if (!item) return undefined;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && !deleting) {
        onCancel();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [item, deleting, onCancel]);

  if (!item) return null;

  const cleanName = item.name.replace(/\.img$/i, '');
  const confirmText = t('myMaps.deleteConfirm').replace('{name}', cleanName);

  return (
    <div className="modal-overlay" onClick={() => !deleting && onCancel()}>
      <div className="modal-sheet" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-icon-destructive">
            <Trash2 size={24} />
          </div>
          <h3>{t('myMaps.deleteTitle')}</h3>
          <p className="modal-description">{confirmText}</p>
          <div className="modal-file-badge">
            {item.name} • {formatBytes(item.size)}
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
            onClick={() => onConfirm(item)}
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

export default DeleteConfirmationModal;
