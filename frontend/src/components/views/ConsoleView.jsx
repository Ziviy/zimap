import React, { useRef, useEffect } from 'react';
import { XCircle, CheckCircle2, Download, FolderOpen } from 'lucide-react';
import { formatBytes, getRevealInfo } from '../../utils/formatters';

export const ConsoleView = ({ 
  logs, 
  setLogs, 
  stage, 
  progress, 
  progressDetail, 
  jobRunning, 
  onCancel, 
  t, 
  errorMsg, 
  activeJob, 
  completedJob, 
  onExport, 
  onReveal 
}) => {
  const endRef = useRef(null);

  const stages = [
    { id: 'download', label: '1. ' + t('console.stages.download') },
    { id: 'splitter', label: '2. ' + t('console.stages.split') },
    { id: 'mkgmap', label: '3. ' + t('console.stages.compile') },
    { id: 'done', label: '4. ' + t('console.stages.done') },
  ];

  const getStageIndex = () => {
    switch(stage) {
      case 'download': return 0;
      case 'splitter': return 1;
      case 'mkgmap': return 2;
      case 'done': return 3;
      case 'error': return -1;
      default: return -1;
    }
  };

  const currentIdx = getStageIndex();
  
  // Find which stage failed if any
  let errorIdx = -1;
  if (stage === 'error') {
    const lastLog = logs.join(' ').toLowerCase();
    if (lastLog.includes('download')) errorIdx = 0;
    else if (lastLog.includes('split')) errorIdx = 1;
    else if (lastLog.includes('mkgmap')) errorIdx = 2;
    else errorIdx = 0;
  }

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  let metricsText = '';
  if (progressDetail && progressDetail.speedText) {
    metricsText = `${formatBytes(progressDetail.downloaded)} / ${formatBytes(progressDetail.total)} (${progressDetail.speedText})`;
    if (progressDetail.etaSeconds) {
      metricsText += ` • ${t('console.eta')}: ${progressDetail.etaSeconds}s`;
    }
  }

  const finishedJob = completedJob || activeJob;
  const finishedName = finishedJob?.name 
    ? `${finishedJob.name.replace(/\.img$/i, '')}.img` 
    : 'gmapsupp.img';

  return (
    <div className="view-container" style={{ display: 'flex', flexDirection: 'column' }}>
      <div className="view-header">
        <div>
          <h2>{t('header.console')}</h2>
          <p className="subtitle">{t('header.consoleSub')}</p>
        </div>
        {jobRunning && stage !== 'error' && (
          <button className="btn-danger" onClick={onCancel}>
            <XCircle size={14} style={{ marginRight: 6 }} />
            {t('console.stopProcess')}
          </button>
        )}
      </div>

      {stage === 'error' && (
        <div className="console-error-card">
          <XCircle className="icon" size={24} />
          <div className="console-error-content">
            <h4>{t('console.errorTitle')}</h4>
            <p>{errorMsg || 'An unknown error occurred.'}</p>
          </div>
        </div>
      )}

      {stage === 'done' && (
        <div className="console-success-card">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <CheckCircle2 className="icon" size={24} />
            <div>
              <h4 className="console-success-title">{t('console.doneSuccess')}</h4>
              <p className="console-success-sub">{finishedName}</p>
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <button 
              className="btn-secondary btn-sm"
              onClick={() => onExport && onExport(finishedName)}
              title={t('buttons.exportTooltip') || t('myMaps.exportTooltip')}
            >
              <Download size={13} style={{ marginRight: 4 }} />
              {t('buttons.export')}
            </button>
            <button 
              className="btn-secondary btn-sm"
              onClick={() => onReveal && onReveal(finishedName)}
              title={getRevealInfo(t).tooltip}
            >
              <FolderOpen size={13} style={{ marginRight: 4 }} />
              {getRevealInfo(t).label}
            </button>
          </div>
        </div>
      )}

      <div className="card stepper-card">
        <div className="stepper">
          {stages.map((st, i) => {
            const isError = stage === 'error' && errorIdx === i;
            const isDone = (currentIdx > i || stage === 'done') && !isError && stage !== 'error';
            const isActive = currentIdx === i && stage !== 'done' && stage !== 'error';
            return (
              <div 
                key={st.id} 
                className={`step-item ${isDone ? 'done' : ''} ${isActive ? 'active' : ''} ${isError ? 'error' : ''}`}
              >
                <div className="step-circle">
                  {isError ? <XCircle size={16} /> : (isDone ? <CheckCircle2 size={16} /> : (i + 1))}
                </div>
                <div className="step-label">{st.label}</div>
              </div>
            );
          })}
        </div>

        {progress > 0 && stage !== 'error' && (
          <div className="progress-wrapper">
            <div className="progress-bar-meta">
              <span className="progress-bar-meta-left">{stages.find(s => s.id === stage)?.label || ''}</span>
              <span className="progress-bar-meta-right">{metricsText} {metricsText ? '•' : ''} {progress}%</span>
            </div>
            <div className="progress-bar-container">
              <div className="progress-bar-fill" style={{ width: `${progress}%` }}></div>
              <span className={`progress-text ${progress >= 88 ? 'covered' : 'uncovered'}`}>{progress}%</span>
            </div>
          </div>
        )}
      </div>

      <div className="terminal-container">
        <div className="terminal-header">
          <div className="terminal-title">STDOUT & STDERR LOGS</div>
          <div className="terminal-actions">
            <button className="terminal-btn" onClick={() => navigator.clipboard.writeText(logs.join('\n'))}>
              {t('console.copyLogs')}
            </button>
            <button className="terminal-btn" onClick={() => setLogs([])}>
              {t('console.clearLogs')}
            </button>
          </div>
        </div>
        <div className="terminal">
          {logs.length === 0 ? (
            <div className="terminal-empty">{t('console.logsEmpty')}</div>
          ) : (
            logs.map((line, idx) => (
              <div key={idx} className="terminal-line">{line}</div>
            ))
          )}
          <div ref={endRef} />
        </div>
      </div>
    </div>
  );
};

export default ConsoleView;
