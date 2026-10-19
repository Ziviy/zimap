import React from 'react';
import { 
  FolderTree, FolderOutput, Watch, FileUp, Terminal, Wrench, Settings, Info, Sun, Moon 
} from 'lucide-react';

export const Sidebar = ({ 
  currentView, 
  setCurrentView, 
  theme, 
  toggleTheme, 
  jobRunning, 
  toolsReady, 
  deviceConnected,
  appVersion,
  t 
}) => (
  <div className="sidebar">
    <div className="sidebar-drag-region"></div>
    
    <div className="sidebar-section" style={{ marginTop: 8 }}>
      <div className="sidebar-section-title">MAPS & CATALOG</div>
      {[
        { id: 'catalog', icon: FolderTree, label: t('nav.catalog') },
        { id: 'my-maps', icon: FolderOutput, label: t('nav.myMaps') },
        { 
          id: 'device', 
          icon: Watch, 
          label: t('nav.device'),
          badge: deviceConnected ? '●' : null,
          badgeColor: 'var(--system-green)'
        },
        { id: 'local', icon: FileUp, label: t('nav.local') },
      ].map(item => (
        <div 
          key={item.id} 
          className={`sidebar-item ${currentView === item.id ? 'active' : ''}`}
          onClick={() => setCurrentView(item.id)}
        >
          <item.icon className="icon" />
          <span>{item.label}</span>
          {item.badge && <span className="running-indicator" style={{ color: item.badgeColor, marginLeft: 'auto' }}>{item.badge}</span>}
        </div>
      ))}
    </div>

    <div className="sidebar-section" style={{ marginTop: 12 }}>
      <div className="sidebar-section-title">SYSTEM</div>
      {[
        { 
          id: 'console', 
          icon: Terminal, 
          label: t('nav.console'),
          badge: jobRunning ? '●' : null,
          badgeColor: 'var(--system-green)'
        },
        { id: 'tools', icon: Wrench, label: t('nav.tools') },
        { id: 'settings', icon: Settings, label: t('nav.settings') },
        { id: 'about', icon: Info, label: t('nav.about') }
      ].map(item => (
        <div 
          key={item.id} 
          className={`sidebar-item ${currentView === item.id ? 'active' : ''}`}
          onClick={() => setCurrentView(item.id)}
        >
          <item.icon className="icon" />
          <span>{item.label}</span>
          {item.badge && <span className="running-indicator" style={{ color: item.badgeColor, marginLeft: 'auto' }}>{item.badge}</span>}
        </div>
      ))}
    </div>
    
    <div className="sidebar-footer" style={{ marginTop: 'auto', padding: 0, borderTop: 'none' }}>
      <div className="sidebar-footer-branding" title="ZiMap Downloader">
        <div className="sidebar-footer-title">ZiMap</div>
        <div className="sidebar-footer-subtitle">Map Downloader</div>
        <div className="sidebar-footer-badge">{appVersion || '0.0.2 BETA'}</div>
        <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 8, display: 'flex', alignItems: 'center' }}>
          <span className="ready-indicator" style={{ backgroundColor: toolsReady ? 'var(--system-green)' : 'var(--system-red)' }}></span>
          {toolsReady ? t('common.ready') : t('common.needsSetup')}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-tertiary)', marginTop: 4 }}>by Ziviy</div>
      </div>

      <div className="sidebar-item" onClick={toggleTheme} style={{ margin: '0 8px 12px' }}>
        {theme === 'dark' ? <Sun className="icon" /> : <Moon className="icon" />}
        <span>{theme === 'dark' ? t('common.themeLight') : t('common.themeDark')}</span>
      </div>
    </div>
  </div>
);

export default Sidebar;
