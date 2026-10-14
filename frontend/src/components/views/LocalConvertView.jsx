import React, { useState } from 'react';
import { FileUp, FolderOpen, Play } from 'lucide-react';
import * as WailsApp from '../../../wailsjs/go/main/App';

export const LocalConvertView = ({ onStartLocalBuild, t, detailLevel = 'standard', setDetailLevel }) => {
  const [filePath, setFilePath] = useState('');
  const [mapName, setMapName] = useState('');
  const [heap, setHeap] = useState(4);
  const [dragging, setDragging] = useState(false);

  const handleBrowse = async () => {
    try {
      if (WailsApp.SelectLocalPBFDialog) {
        const selected = await WailsApp.SelectLocalPBFDialog();
        if (selected) {
          setFilePath(selected);
          const base = selected.split(/[/\\]/).pop().replace(/\.(osm\.pbf|pbf|osm\.gz|osm\.bz2|osm)$/i, '');
          if (!mapName) setMapName(base);
        }
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    setDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const file = e.dataTransfer.files[0];
      setFilePath(file.path || file.name);
      const base = file.name.replace(/\.(osm\.pbf|pbf|osm\.gz|osm\.bz2|osm)$/i, '');
      if (!mapName) setMapName(base);
    }
  };

  return (
    <div className="view-container">
      <div className="view-header">
        <div>
          <h2>{t('header.local')}</h2>
          <p className="subtitle">{t("header.localSub")}</p>
        </div>
      </div>

      <div 
        className={`dropzone card ${dragging ? 'active' : ''}`}
        onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
      >
        <FileUp size={44} color="var(--system-blue)" style={{ margin: '0 auto 12px' }} />
        <h3>{t('local.dropTitle')}</h3>
        <p className="subtitle">{t('local.dropSubtitle')}</p>
        <button className="btn-secondary" style={{ marginTop: 12 }} onClick={handleBrowse}>
          <FolderOpen size={14} style={{ marginRight: 6 }} />
          {t('buttons.browse')}
        </button>
      </div>

      {filePath && (
        <div className="card file-selected-card">
          <div className="file-info">
            <span className="file-label">{t('local.selectedFile')}:</span>
            <span className="file-path">{filePath}</span>
          </div>
        </div>
      )}

      <div className="card form-card">
        <div className="form-group">
          <label>{t('local.mapNameLabel')}</label>
          <input 
            type="text" 
            className="text-input" 
            placeholder={t('local.mapNamePlaceholder')} 
            value={mapName} 
            onChange={e => setMapName(e.target.value)}
          />
        </div>

        <div className="form-group">
          <div className="slider-header">
            <label>{t('settings.heap')}: <strong>{heap} GB</strong></label>
            <span className="subtitle">{t('settings.heapSub')}</span>
          </div>
          <input 
            type="range" 
            min="2" 
            max="32" 
            step="2"
            value={heap} 
            onChange={e => setHeap(Number(e.target.value))}
            className="apple-slider"
          />
        </div>

        <div className="form-group">
          <div className="slider-header">
            <label>{t('local.detailLabel')}</label>
            <span className="subtitle">{t('local.detailSub')}</span>
          </div>
          <div className="segmented-control">
            {['standard', 'high', 'topo'].map(level => (
              <button
                key={level}
                className={detailLevel === level ? 'active' : ''}
                onClick={() => setDetailLevel && setDetailLevel(level)}
              >
                {t(`detailLevels.${level}`) || level}
              </button>
            ))}
          </div>
        </div>

        <button 
          className="btn-primary btn-large" 
          disabled={!filePath}
          onClick={() => onStartLocalBuild(filePath, mapName || 'map', heap + 'g', detailLevel)}
        >
          <Play size={16} style={{ marginRight: 8 }} />
          {t('local.convertBtn')}
        </button>
      </div>
    </div>
  );
};

export default LocalConvertView;
