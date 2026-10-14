import React, { useRef, useEffect } from 'react';
import { Search, X } from 'lucide-react';

export const Toolbar = ({ title, currentView, searchQuery, setSearchQuery, onSearchSubmit, t }) => {
  const searchInputRef = useRef(null);
  const showSearch = currentView === 'catalog' || currentView === 'my-maps';
  const placeholder = currentView === 'catalog' ? t('search.catalogPlaceholder') : t('search.myMapsPlaceholder');

  useEffect(() => {
    const handleKeyDown = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        if (showSearch) {
          e.preventDefault();
          searchInputRef.current?.focus();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [showSearch]);

  return (
    <div className="toolbar">
      <div className="breadcrumb">{title}</div>
      {showSearch && (
        <div className="search-bar">
          <Search size={14} color="var(--text-secondary)" />
          <input 
            ref={searchInputRef}
            type="text" 
            placeholder={placeholder} 
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') onSearchSubmit();
            }}
          />
          {searchQuery && (
            <X 
              size={13} 
              className="search-clear-btn" 
              onClick={() => setSearchQuery('')} 
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Toolbar;
