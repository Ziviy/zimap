import React, { useEffect } from 'react';
import { 
  Sparkles, Download, ExternalLink, RefreshCw, CheckCircle2, AlertCircle, RotateCcw 
} from 'lucide-react';
import { formatBytes } from '../../utils/formatters';

export const UpdateModal = ({ 
  updateInfo, 
  updateStatus = { stage: 'idle', progress: 0, metrics: null, error: null },
  onStartUpdate,
  onCancelUpdate,
  onRestartApp,
  onDownload, 
  onClose, 
  appVersion,
  t 
}) => {
  const stage = updateStatus.stage || 'idle';
  const isBusy = stage === 'download' || stage === 'extract' || stage === 'install';

  useEffect(() => {
    if (!updateInfo) return undefined;
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && !isBusy) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [updateInfo, onClose, isBusy]);

  if (!updateInfo) return null;

  return (
    <div className="modal-overlay" onClick={isBusy ? undefined : onClose}>
      <div className="modal-sheet update-modal-sheet" onClick={(e) => e.stopPropagation()}>
        
        {/* Header */}
        <div className="modal-header">
          <div className={`modal-icon-accent ${stage === 'ready' ? 'success' : stage === 'error' ? 'error' : ''}`}>
            {stage === 'ready' ? (
              <CheckCircle2 size={24} style={{ color: 'var(--system-green)' }} />
            ) : stage === 'error' ? (
              <AlertCircle size={24} style={{ color: 'var(--system-red)' }} />
            ) : isBusy ? (
              <RefreshCw size={24} className="spin" style={{ color: 'var(--system-blue)' }} />
            ) : (
              <Sparkles size={24} />
            )}
          </div>
          
          <h3>
            {stage === 'ready' 
              ? (t('update.readyTitle') || 'Обновление готово к установке!') 
              : stage === 'error'
              ? (t('update.checkError') || 'Ошибка обновления')
              : (t('update.modalTitle') || 'Доступна новая версия ZiMap')}
          </h3>

          <div className="update-badge-row">
            <span className="version-pill">
              {t('update.currentVersion')}: {updateInfo.currentVersion || appVersion || '0.0.3 BETA'}
            </span>
            <span style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>→</span>
            <span className="version-pill new">
              {t('update.latestVersion')}: {updateInfo.version}
            </span>
          </div>

          {updateInfo.assetName && stage === 'idle' && (
            <div className="update-asset-tag">
              <span>{updateInfo.assetName}</span>
              {updateInfo.assetSize > 0 && (
                <span className="asset-size">• {formatBytes(updateInfo.assetSize)}</span>
              )}
            </div>
          )}
        </div>

        {/* Body content based on stage */}
        {stage === 'idle' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-secondary)' }}>
              {t('update.releaseNotes')}
            </span>
            <div className="release-notes-box">
              {updateInfo.releaseNotes || 'No release notes provided.'}
            </div>
          </div>
        )}

        {isBusy && (
          <div className="update-progress-card">
            <div className="update-progress-header">
              <span className="update-stage-title">
                {stage === 'download' 
                  ? (t('update.downloading') || 'Загрузка обновления...') 
                  : stage === 'extract'
                  ? (t('update.extracting') || 'Распаковка и проверка...')
                  : (t('update.installing') || 'Установка обновления...')}
              </span>
              <span className="update-progress-pct">{updateStatus.progress}%</span>
            </div>

            <div className="update-progress-bar-wrap">
              <div 
                className="update-progress-bar-fill" 
                style={{ width: `${Math.min(100, Math.max(0, updateStatus.progress))}%` }} 
              />
            </div>

            {updateStatus.metrics && (
              <div className="update-progress-stats">
                <span>{updateStatus.metrics.downloaded} / {updateStatus.metrics.total}</span>
                {updateStatus.metrics.speed && <span>• {updateStatus.metrics.speed}</span>}
                {updateStatus.metrics.eta && <span>• ETA: {updateStatus.metrics.eta}</span>}
              </div>
            )}
          </div>
        )}

        {stage === 'ready' && (
          <div className="update-ready-box">
            <p style={{ margin: 0, fontSize: 13, lineHeight: 1.5, color: 'var(--text-secondary)' }}>
              {t('update.readyDesc') || 'Новая версия успешно загружена и установлена. Перезапустите приложение, чтобы применить изменения.'}
            </p>
          </div>
        )}

        {stage === 'error' && (
          <div className="update-error-box">
            <p style={{ margin: 0, fontSize: 13, color: 'var(--system-red)', lineHeight: 1.4 }}>
              {updateStatus.error || 'An unexpected error occurred during update.'}
            </p>
          </div>
        )}

        {/* Modal Actions */}
        <div className="modal-actions">
          {stage === 'idle' && (
            <>
              <button className="btn-secondary modal-btn" onClick={onClose}>
                {t('buttons.cancel') || 'Отмена'}
              </button>
              {updateInfo.downloadURL && (
                <button 
                  className="btn-secondary modal-btn" 
                  onClick={() => onDownload(updateInfo.downloadURL)}
                  title={t('update.openInBrowser')}
                >
                  <ExternalLink size={13} style={{ marginRight: 6 }} />
                  {t('update.openInBrowser') || 'В браузере'}
                </button>
              )}
              {updateInfo.canAutoUpdate && onStartUpdate && (
                <button 
                  className="btn-primary modal-btn" 
                  onClick={() => onStartUpdate(updateInfo.assetURL || 'simulate')}
                >
                  <Download size={14} style={{ marginRight: 6 }} />
                  {t('update.updateNow') || 'Обновить сейчас'}
                </button>
              )}
            </>
          )}

          {isBusy && (
            <button className="btn-secondary modal-btn" onClick={onCancelUpdate}>
              {t('update.cancelUpdate') || 'Отменить'}
            </button>
          )}

          {stage === 'ready' && (
            <>
              <button className="btn-secondary modal-btn" onClick={onClose}>
                {t('update.restartLater') || 'Позже'}
              </button>
              <button className="btn-primary modal-btn" onClick={onRestartApp}>
                <RotateCcw size={14} style={{ marginRight: 6 }} />
                {t('update.restartNow') || 'Перезапустить сейчас'}
              </button>
            </>
          )}

          {stage === 'error' && (
            <>
              <button className="btn-secondary modal-btn" onClick={onClose}>
                {t('update.close') || 'Закрыть'}
              </button>
              {updateInfo.downloadURL && (
                <button 
                  className="btn-secondary modal-btn" 
                  onClick={() => onDownload(updateInfo.downloadURL)}
                >
                  <ExternalLink size={13} style={{ marginRight: 6 }} />
                  {t('update.openInBrowser') || 'В браузере'}
                </button>
              )}
              {onStartUpdate && (
                <button 
                  className="btn-primary modal-btn" 
                  onClick={() => onStartUpdate(updateInfo.assetURL || 'simulate')}
                >
                  <RefreshCw size={13} style={{ marginRight: 6 }} />
                  {t('update.retry') || 'Повторить'}
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default UpdateModal;
