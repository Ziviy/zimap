import React, { useState } from 'react';
import { RefreshCw, Clock, FolderOutput, FolderOpen, Trash2, Download, X } from 'lucide-react';
import CircularProgress from '../common/CircularProgress';
import DeleteConfirmationModal from '../modals/DeleteConfirmationModal';
import { formatBytes, formatDate, getRevealInfo } from '../../utils/formatters';

export const OutputsView = ({ 
  outputs, 
  searchQuery, 
  onRefreshOutputs, 
  onReveal, 
  onUpdate, 
  onDelete, 
  onExport,
  onNavigateConsole,
  jobRunning, 
  activeJob, 
  stage, 
  progress, 
  progressDetail, 
  onCancel, 
  downloadQueue = [],
  onRemoveFromQueue,
  onClearQueue,
  t 
}) => {
  const [itemToDelete, setItemToDelete] = useState(null);
  const [deleting, setDeleting] = useState(false);

  const filteredOutputs = searchQuery 
    ? outputs.filter(o => o.name.toLowerCase().includes(searchQuery.toLowerCase())) 
    : outputs;

  const handleConfirmDelete = async (item) => {
    setDeleting(true);
    try {
      if (onDelete) {
        await onDelete(item.path);
      }
      setItemToDelete(null);
    } catch (err) {
      console.error("Delete failed:", err);
    } finally {
      setDeleting(false);
    }
  };

  let metricsText = '';
  if (progressDetail && progressDetail.speedText) {
    metricsText = `${formatBytes(progressDetail.downloaded)} / ${formatBytes(progressDetail.total)} (${progressDetail.speedText})`;
    if (progressDetail.etaSeconds) {
      metricsText += ` • ${t('console.eta')}: ${progressDetail.etaSeconds}s`;
    }
  }

  const isCurrentJobActive = jobRunning && activeJob;

  return (
    <div className="view-container">
      <div className="view-header">
        <div>
          <h2>{t('header.myMaps')}</h2>
          <p className="subtitle">{t("header.myMapsSub")}</p>
        </div>
        <button className="btn-secondary" onClick={onRefreshOutputs}>
          <RefreshCw size={14} style={{ marginRight: 6 }} />
          {t('buttons.refresh')}
        </button>
      </div>

      {isCurrentJobActive && (
        <div 
          className="card output-card active-download-card clickable-progress"
          onClick={() => onNavigateConsole && onNavigateConsole()}
          title={t('console.viewConsole')}
        >
          <div className="output-card-left">
            <CircularProgress 
              progress={progress} 
              stage={stage} 
              size={34} 
              onCancel={onCancel}
              tooltip={t('console.viewConsole')}
              cancelTooltip={t('console.stopProcess')}
              onNavigateConsole={onNavigateConsole}
            />
            <div className="output-info">
              <div className="output-title-row">
                <span className="output-title" title={activeJob.name}>{activeJob.name}</span>
                <span className="format-badge">.IMG</span>
                <span className="stage-pill active">{t(`console.stages.${stage}`) || stage}</span>
              </div>
              <div className="output-meta">
                {metricsText || (stage === 'download' ? `${progress}%` : (t(`console.stages.${stage}`) || 'Processing...'))}
              </div>
            </div>
          </div>
          <div className="output-actions" onClick={(e) => e.stopPropagation()}>
            <button 
              className="btn-danger btn-sm" 
              onClick={(e) => {
                e.stopPropagation();
                onCancel();
              }}
            >
              <X size={13} style={{ marginRight: 4 }} />
              {t('buttons.cancel')}
            </button>
          </div>
        </div>
      )}

      {downloadQueue && downloadQueue.length > 0 && (
        <div className="queue-section">
          <div className="queue-header">
            <div className="queue-title-row">
              <Clock size={15} color="var(--system-orange)" />
              <span className="queue-title">{t('queue.title')}</span>
              <span className="queue-count-badge">{downloadQueue.length}</span>
            </div>
            {downloadQueue.length > 1 && (
              <button className="btn-secondary btn-sm" onClick={onClearQueue}>
                {t('queue.clear')}
              </button>
            )}
          </div>
          <div className="queue-list">
            {downloadQueue.map((item, idx) => (
              <div key={item.id || item.url || idx} className="card queue-card">
                <div className="queue-card-left">
                  <span className="queue-index">#{idx + 1}</span>
                  <div className="output-info">
                    <div className="output-title-row">
                      <span className="output-title" title={item.name}>{item.name}</span>
                      <span className="format-badge">.IMG</span>
                      <span className="queue-badge">{t('queue.waiting')}</span>
                    </div>
                    <div className="output-meta">
                      {item.url}
                    </div>
                  </div>
                </div>
                <div className="output-actions">
                  <button 
                    className="btn-secondary btn-sm" 
                    onClick={() => onRemoveFromQueue && onRemoveFromQueue(item.id || item.url)}
                    title={t('queue.removeFromQueue')}
                  >
                    <X size={13} style={{ marginRight: 4 }} />
                    {t('buttons.cancel')}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {filteredOutputs.length === 0 && !isCurrentJobActive && (!downloadQueue || downloadQueue.length === 0) ? (
        <div className="card empty-state">
          <FolderOutput size={48} color="var(--system-gray)" style={{ margin: '0 auto 12px' }} />
          <h3>{searchQuery ? `${t('search.noResults')} "${searchQuery}"` : t('myMaps.emptyTitle')}</h3>
          <p className="subtitle">{searchQuery ? '' : t('myMaps.emptySub')}</p>
        </div>
      ) : (
        <div className="card-list">
          {filteredOutputs.map(item => {
            const cleanName = item.name.replace(/\.img$/i, '');
            const isThisUpdating = jobRunning && activeJob && (
              activeJob.name.toLowerCase() === cleanName.toLowerCase() ||
              (activeJob.url && item.sourceUrl && activeJob.url === item.sourceUrl)
            );
            const isThisQueued = downloadQueue && downloadQueue.some(q => 
              q.name.toLowerCase() === cleanName.toLowerCase() ||
              (item.sourceUrl && q.url === item.sourceUrl)
            );
            const dateStr = formatDate(item.modTime || item.meta?.updatedAt || item.meta?.createdAt);
            const revealInfo = getRevealInfo(t);

            return (
              <div key={item.path} className="card output-card">
                <div className="output-info">
                  <div className="output-title-row">
                    <span className="output-title" title={cleanName}>{cleanName}</span>
                    <span className="format-badge">.IMG</span>
                  </div>
                  <div className="output-meta">
                    <span className="output-meta-size">{t('common.size')}: {formatBytes(item.size)}</span>
                    {dateStr && <><span className="meta-sep">•</span><span>{dateStr}</span></>}
                    {item.meta?.regionID && (
                      <>
                        <span className="meta-sep">•</span>
                        <span className="output-meta-slug" title={item.meta.regionID}>{item.meta.regionID}</span>
                      </>
                    )}
                  </div>
                </div>
                <div className="output-actions">
                  {isThisQueued ? (
                    <button 
                      className="btn-queued btn-sm" 
                      onClick={() => onRemoveFromQueue && onRemoveFromQueue(item.sourceUrl || cleanName)}
                      title={t('queue.removeFromQueue')}
                    >
                      <Clock size={13} style={{ marginRight: 4 }} />
                      {t('queue.queued')}
                    </button>
                  ) : (
                    <button 
                      className="btn-secondary btn-sm" 
                      title={t('myMaps.updateTooltip')}
                      onClick={() => onUpdate && onUpdate(item)}
                    >
                      <RefreshCw size={13} style={{ marginRight: 4 }} />
                      {t('myMaps.updateBtn')}
                    </button>
                  )}
                  <button 
                    className="btn-secondary btn-sm" 
                    title={t('buttons.exportTooltip') || t('myMaps.exportTooltip')}
                    onClick={() => onExport && onExport(item.path)}
                  >
                    <Download size={13} style={{ marginRight: 4 }} />
                    {t('buttons.export')}
                  </button>
                  <button 
                    className="btn-secondary btn-sm" 
                    title={revealInfo.tooltip}
                    onClick={() => onReveal(item.path)}
                  >
                    <FolderOpen size={13} style={{ marginRight: 4 }} />
                    {revealInfo.label}
                  </button>
                  <button 
                    className="btn-danger-icon" 
                    title={t('myMaps.deleteTooltip')}
                    disabled={jobRunning && isThisUpdating}
                    onClick={() => setItemToDelete(item)}
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {itemToDelete && (
        <DeleteConfirmationModal 
          item={itemToDelete}
          onConfirm={handleConfirmDelete}
          onCancel={() => setItemToDelete(null)}
          deleting={deleting}
          t={t}
        />
      )}
    </div>
  );
};

export default OutputsView;
