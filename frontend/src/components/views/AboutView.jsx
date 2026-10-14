import React from 'react';
import { RefreshCw } from 'lucide-react';
import appIconUrl from '../../assets/appicon.png';
import { openExternalURL } from '../../utils/formatters';

export const AboutView = ({ t, checkingUpdate, onCheckUpdate, appVersion }) => {
  const openLink = (url) => {
    openExternalURL(url);
  };

  return (
    <div className="view-container">
      <div className="about-container">
        <div className="about-hero">
          <div className="about-app-icon-wrapper">
            <img src={appIconUrl} alt="ZiMap" className="about-app-icon" />
          </div>
          <h1 style={{ margin: '12px 0 2px', fontSize: 30, letterSpacing: '-0.5px', fontWeight: 700 }}>
            {t('about.title') || 'ZiMap'}
          </h1>
          <div className="about-subtitle">
            {t('about.subtitle') || 'Map Downloader'}
          </div>
          <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Version {appVersion || '0.0.1 BETA'} (Build 2026.1)</div>
          <div style={{ display: 'flex', gap: 10, marginTop: 12, flexWrap: 'wrap', justifyContent: 'center' }}>
            <button className="btn-secondary" onClick={() => openLink('https://github.com/Ziviy')}>
              {t('about.author') || 'Author: Ziviy'}
            </button>
            <button className="btn-secondary" onClick={onCheckUpdate} disabled={checkingUpdate}>
              <RefreshCw size={13} className={checkingUpdate ? 'spin' : ''} style={{ marginRight: 6 }} />
              {checkingUpdate ? t('update.checking') : t('update.check')}
            </button>
          </div>
        </div>

        <div className="about-banner">
          {t('about.banner') || 'FOR ALL OUTDOOR & GPS ENTHUSIASTS'}
        </div>

        <div className="about-card">
          <h3 style={{ marginTop: 0 }}>{t('header.about')}</h3>
          <p style={{ color: 'var(--text-secondary)', lineHeight: 1.6 }}>
            {t('about.description')}
          </p>
          
          <h4 style={{ marginTop: 24, marginBottom: 12 }}>{t('about.technologies') || 'Technologies used'}</h4>
          <div className="about-links" style={{ flexWrap: 'wrap' }}>
            <span className="about-tech-badge">OpenStreetMap</span>
            <span className="about-tech-badge">Geofabrik</span>
            <span className="about-tech-badge">Mkgmap</span>
            <span className="about-tech-badge">Splitter</span>
            <span className="about-tech-badge">Wails v2</span>
            <span className="about-tech-badge">React</span>
          </div>

          {t('about.disclaimer') && (
            <p style={{ marginTop: 24, fontSize: 11, color: 'var(--text-tertiary, #888)', lineHeight: 1.5, borderTop: '1px solid var(--border-subtle, rgba(255,255,255,0.08))', paddingTop: 14 }}>
              {t('about.disclaimer')}
            </p>
          )}
        </div>
      </div>
    </div>
  );
};

export default AboutView;
