import React, { useState, useEffect, useMemo } from 'react';
import { ChevronRight, Download, RefreshCw, CheckCircle2, Clock } from 'lucide-react';
import CircularProgress from '../common/CircularProgress';
import * as WailsApp from '../../../wailsjs/go/main/App';

export const RegionNode = ({ 
  region, 
  level = 0, 
  onBuild, 
  onCancel,
  onNavigateConsole,
  isRegionDownloaded, 
  isRegionDownloading, 
  isRegionQueued,
  onRemoveFromQueue,
  stage, 
  progress, 
  jobRunning,
  t, 
  defaultExpanded = level < 1 
}) => {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const hasChildren = region.children && region.children.length > 0;
  const isPbf = !!region.pbfUrl;

  const isDownloaded = isPbf && isRegionDownloaded && isRegionDownloaded(region);
  const isDownloading = isPbf && isRegionDownloading && isRegionDownloading(region);
  const isQueued = isPbf && isRegionQueued && isRegionQueued(region);

  return (
    <div className="tree-node" style={{ paddingLeft: level > 0 ? 18 : 0 }}>
      <div className="tree-item-row">
        <div className="tree-item-info" onClick={() => hasChildren && setExpanded(!expanded)}>
          {hasChildren ? (
            <ChevronRight size={14} className={`chevron ${expanded ? 'expanded' : ''}`} />
          ) : (
            <span className="chevron-spacer" />
          )}
          <span className="tree-item-name">{region.name}</span>
          {isPbf && <span className="pbf-tag">PBF</span>}
        </div>
        {isPbf && (
          isDownloading ? (
            <div 
              className="catalog-action-slot clickable-progress"
              onClick={() => onNavigateConsole && onNavigateConsole()}
              title={t('console.viewConsole')}
              role="button"
              tabIndex={0}
            >
              <span className="catalog-progress-label">
                {stage === 'download' ? `${progress}%` : (t(`console.stages.${stage}`) || stage)}
              </span>
              <CircularProgress 
                progress={progress} 
                stage={stage} 
                size={26}
                onCancel={onCancel}
                tooltip={t('console.viewConsole')}
                cancelTooltip={t('console.stopProcess')}
                onNavigateConsole={onNavigateConsole}
              />
            </div>
          ) : isQueued ? (
            <div className="catalog-action-slot">
              <button 
                className="btn-queued btn-sm"
                onClick={() => onRemoveFromQueue && onRemoveFromQueue(region.pbfUrl || region.name)}
                title={t('queue.removeFromQueue')}
              >
                <Clock size={12} style={{ marginRight: 4 }} />
                {t('queue.queued')}
              </button>
            </div>
          ) : isDownloaded ? (
            <div className="catalog-action-slot">
              <span className="badge-downloaded" title={t('catalog.downloaded')}>
                <CheckCircle2 size={15} />
              </span>
              <button 
                className="btn-secondary btn-sm btn-update"
                onClick={() => onBuild(region.pbfUrl, region.name)}
                title={t('myMaps.updateTooltip')}
              >
                <RefreshCw size={12} style={{ marginRight: 4 }} />
                {t('buttons.update')}
              </button>
            </div>
          ) : (
            <div className="catalog-action-slot">
              <button 
                className="btn-primary btn-sm"
                onClick={() => onBuild(region.pbfUrl, region.name)}
              >
                <Download size={13} style={{ marginRight: 4 }} />
                {t('buttons.download')}
              </button>
            </div>
          )
        )}
      </div>
      {hasChildren && expanded && (
        <div className="tree-children">
          {region.children.map(child => (
            <RegionNode 
              key={child.id} 
              region={child} 
              level={level + 1} 
              onBuild={onBuild} 
              onCancel={onCancel}
              onNavigateConsole={onNavigateConsole}
              isRegionDownloaded={isRegionDownloaded}
              isRegionDownloading={isRegionDownloading}
              isRegionQueued={isRegionQueued}
              onRemoveFromQueue={onRemoveFromQueue}
              stage={stage}
              progress={progress}
              jobRunning={jobRunning}
              t={t} 
            />
          ))}
        </div>
      )}
    </div>
  );
};

export const CatalogView = ({ 
  onStartBuild, 
  onCancel,
  onNavigateConsole,
  searchQuery, 
  catalog, 
  loading, 
  loadCatalog, 
  outputs, 
  activeJob, 
  jobRunning, 
  stage, 
  progress, 
  activeProvider = 'geofabrik',
  onSelectProvider,
  providers = [],
  isRegionQueued,
  onRemoveFromQueue,
  t 
}) => {
  const [searchResults, setSearchResults] = useState(null);
  const [searching, setSearching] = useState(false);
  const [bbbikeStyle, setBbbikeStyle] = useState(localStorage.getItem('zimap_bbbike_style') || 'pbf');

  const downloadedMapIndex = useMemo(() => {
    const names = new Set();
    const urls = new Set();
    const ids = new Set();
    for (const item of (outputs || [])) {
      const clean = item.name.replace(/\.img$/i, '').trim().toLowerCase();
      names.add(clean);
      if (item.sourceUrl) urls.add(item.sourceUrl);
      if (item.meta?.sourceURL) urls.add(item.meta.sourceURL);
      if (item.meta?.regionID) ids.add(item.meta.regionID.toLowerCase());
      if (item.meta?.name) names.add(item.meta.name.trim().toLowerCase());
    }
    return { names, urls, ids };
  }, [outputs]);

  const isRegionDownloaded = (region) => {
    if (!region || !region.pbfUrl) return false;
    const nameLower = region.name.trim().toLowerCase();
    const idLower = region.id ? region.id.trim().toLowerCase() : '';
    return downloadedMapIndex.names.has(nameLower) ||
           downloadedMapIndex.ids.has(idLower) ||
           downloadedMapIndex.urls.has(region.pbfUrl);
  };

  const isRegionDownloading = (region) => {
    if (!jobRunning || !activeJob) return false;
    const nameLower = region.name.trim().toLowerCase();
    const activeNameLower = (activeJob.name || '').trim().toLowerCase();
    return (activeNameLower === nameLower) || (activeJob.url && activeJob.url === region.pbfUrl);
  };

  useEffect(() => {
    if (!catalog && !loading) {
      loadCatalog(false, activeProvider);
    }
  }, [activeProvider]);

  useEffect(() => {
    if (!searchQuery.trim()) {
      setSearchResults(null);
      return;
    }
    const timer = setTimeout(async () => {
      setSearching(true);
      try {
        if (WailsApp.SearchCatalogByProvider) {
          const res = await WailsApp.SearchCatalogByProvider(activeProvider, searchQuery);
          setSearchResults(res);
        } else if (WailsApp.SearchCatalog) {
          const res = await WailsApp.SearchCatalog(searchQuery);
          setSearchResults(res);
        }
      } catch (err) {
        console.error("Search failed:", err);
      } finally {
        setSearching(false);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [searchQuery, activeProvider]);

  const handleRefresh = () => {
    loadCatalog(true, activeProvider);
  };

  // Task 45: resolve the selected BBBike pre-compiled Garmin style into its download URL.
  // Style "pbf" keeps the raw .osm.pbf extract which is compiled locally with the
  // user's default detail level.
  const handleBuildWithStyle = (url, name) => {
    if (activeProvider === 'bbbike' && bbbikeStyle !== 'pbf' && WailsApp.BuildBBBikeStyleURL) {
      WailsApp.BuildBBBikeStyleURL(name, bbbikeStyle)
        .then(styleUrl => onStartBuild(styleUrl, name))
        .catch(() => onStartBuild(url, name));
    } else {
      onStartBuild(url, name);
    }
  };

  const handleStyleSelect = (style) => {
    setBbbikeStyle(style);
    localStorage.setItem('zimap_bbbike_style', style);
  };

  return (
    <div className="view-container">
      <div className="view-header">
        <div>
          <h2>{t('header.catalog')}</h2>
          <p className="subtitle">{t("header.catalogSub")}</p>
        </div>
        <button className="btn-secondary" onClick={handleRefresh} disabled={loading}>
          <RefreshCw size={14} className={loading ? 'spin' : ''} style={{ marginRight: 6 }} />
          {t('buttons.refresh')}
        </button>
      </div>

      <div className="provider-bar">
        <span className="provider-label">{t('providers.source')}</span>
        <div className="segmented-control">
          {providers.map(p => (
            <button
              key={p.id}
              className={activeProvider === p.id ? 'active' : ''}
              onClick={() => onSelectProvider && onSelectProvider(p.id)}
            >
              {t(`providers.${p.id === 'osm-france' ? 'osmFrance' : p.id}`) || p.name}
            </button>
          ))}
        </div>
      </div>

      {activeProvider === 'bbbike' && (
        <div className="provider-bar bbbike-style-bar">
          <span className="provider-label">{t('bbbikeStyles.label')}</span>
          <div className="segmented-control">
            {['pbf', 'onroad', 'ontrail', 'opentopo', 'osm'].map(styleId => (
              <button
                key={styleId}
                className={bbbikeStyle === styleId ? 'active' : ''}
                onClick={() => handleStyleSelect(styleId)}
              >
                {t(`bbbikeStyles.${styleId}`) || styleId}
              </button>
            ))}
          </div>
        </div>
      )}

      {loading && !catalog ? (
        <div className="loading-state" style={{ textAlign: 'center', marginTop: 40 }}>
          <RefreshCw size={24} className="spin" style={{ marginBottom: 12, color: 'var(--system-blue)' }} />
          <p>Loading {providers.find(p => p.id === activeProvider)?.name || 'catalog'}...</p>
        </div>
      ) : searchResults !== null ? (
        <div className="search-results-container">
          <div className="section-title">
            {t('search.results')} ({searchResults.length})
            {searching && <RefreshCw size={13} className="spin" style={{ marginLeft: 8, verticalAlign: '-2px' }} />}
          </div>
          {searchResults.length === 0 ? (
            <div className="empty-state">{t("search.noResults")} "{searchQuery}"</div>
          ) : (
            <div className="catalog-tree card">
              {searchResults.map(r => (
                <RegionNode 
                  key={r.id} 
                  region={r} 
                  onBuild={handleBuildWithStyle} 
                  onCancel={onCancel}
                  onNavigateConsole={onNavigateConsole}
                  isRegionDownloaded={isRegionDownloaded}
                  isRegionDownloading={isRegionDownloading}
                  isRegionQueued={isRegionQueued}
                  onRemoveFromQueue={onRemoveFromQueue}
                  stage={stage}
                  progress={progress}
                  jobRunning={jobRunning}
                  t={t} 
                  defaultExpanded={true} 
                />
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="catalog-tree card">
          {catalog?.roots?.map(root => (
            <RegionNode 
              key={root.id} 
              region={root} 
              onBuild={handleBuildWithStyle} 
              onCancel={onCancel}
              onNavigateConsole={onNavigateConsole}
              isRegionDownloaded={isRegionDownloaded}
              isRegionDownloading={isRegionDownloading}
              isRegionQueued={isRegionQueued}
              onRemoveFromQueue={onRemoveFromQueue}
              stage={stage}
              progress={progress}
              jobRunning={jobRunning}
              t={t} 
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default CatalogView;
