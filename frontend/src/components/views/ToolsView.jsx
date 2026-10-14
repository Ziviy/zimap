import React from 'react';
import { RefreshCw, CheckCircle2, XCircle } from 'lucide-react';

export const ToolsView = ({ toolsStatus, onRefreshTools, checkingTools, onUpdateTools, toolsUpdating, toolsLogs, t }) => {
  const tools = [
    {
      name: 'Adoptium Eclipse Temurin JDK 21',
      desc: 'Java Runtime Environment для работы Splitter и Mkgmap',
      status: toolsStatus?.java || {},
      key: 'java'
    },
    {
      name: 'Mkgmap (r4924)',
      desc: 'Компилятор векторных карт OpenStreetMap в формат .img для GPS-навигаторов',
      status: toolsStatus?.mkgmap || {},
      key: 'mkgmap'
    },
    {
      name: 'Splitter (r654)',
      desc: 'Утилита нарезки полигонов OSM на оптимизированные тайлы',
      status: toolsStatus?.splitter || {},
      key: 'splitter'
    }
  ];

  return (
    <div className="view-container">
      <div className="view-header">
        <div>
          <h2>{t('header.tools')}</h2>
          <p className="subtitle">{t("header.toolsSub")}</p>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button className="btn-secondary" onClick={() => onRefreshTools(true)} disabled={checkingTools}>
            <RefreshCw size={14} className={checkingTools ? 'spin' : ''} style={{ marginRight: 6 }} />
            {t('tools.check')}
          </button>
          <button className="btn-primary" onClick={onUpdateTools} disabled={toolsUpdating}>
            <RefreshCw size={14} className={toolsUpdating ? 'spin' : ''} style={{ marginRight: 6 }} />
            {toolsUpdating ? t('tools.updating') : t('tools.update')}
          </button>
        </div>
      </div>

      <div className="card-list">
        {tools.map(tool => (
          <div key={tool.key} className="card tool-card" style={{ flexDirection: 'column', alignItems: 'flex-start' }}>
            <div className="tool-info" style={{ width: '100%' }}>
              <div className="tool-header">
                <h3>{tool.name}</h3>
                <span className={`status-pill ${tool.status.installed ? 'success' : 'error'}`}>
                  {tool.status.installed ? (
                    <><CheckCircle2 size={13} style={{ marginRight: 4 }} /> {t('tools.installed')}</>
                  ) : (
                    <><XCircle size={13} style={{ marginRight: 4 }} /> {t('tools.notInstalled')}</>
                  )}
                </span>
                {tool.status.installed && (
                  <span className={`tool-badge ${tool.status.embedded ? 'embedded' : 'system'}`}>
                    {tool.status.embedded ? t('tools.embedded') : t('tools.system')}
                  </span>
                )}
                {!tool.status.installed && (
                  <span className="tool-badge missing">{t('tools.notInstalled')}</span>
                )}
              </div>
              <p className="subtitle">{tool.desc}</p>
              {tool.status.path && (
                <div className="tool-path">{tool.status.path}</div>
              )}
            </div>
          </div>
        ))}
      </div>

      {toolsLogs && toolsLogs.length > 0 && (
        <div className="terminal-container" style={{ marginTop: 20 }}>
          <div className="terminal-header">
            <div className="terminal-title">TOOLS UPDATE LOG</div>
          </div>
          <div className="terminal" style={{ height: 180 }}>
            {toolsLogs.map((l, i) => (
              <div key={i} className="terminal-line">{l}</div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default ToolsView;
