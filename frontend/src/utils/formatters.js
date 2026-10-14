export function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function formatDate(dateVal) {
  if (!dateVal) return '';
  try {
    const d = new Date(dateVal);
    if (isNaN(d.getTime())) return '';
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  } catch {
    return '';
  }
}

export function formatSpeed(bytesPerSec) {
  if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s';
  if (bytesPerSec >= 1024 * 1024) {
    return (bytesPerSec / (1024 * 1024)).toFixed(1) + ' MB/s';
  } else if (bytesPerSec >= 1024) {
    return (bytesPerSec / 1024).toFixed(1) + ' KB/s';
  }
  return Math.round(bytesPerSec) + ' B/s';
}

export function formatEta(seconds) {
  if (!seconds || seconds <= 0) return '';
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  if (mins < 60) return `${mins}m ${secs}s`;
  const hours = Math.floor(mins / 60);
  const remainingMins = mins % 60;
  return `${hours}h ${remainingMins}m`;
}

export const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.userAgent || navigator.platform);
export const isWin = typeof navigator !== 'undefined' && /Win/.test(navigator.userAgent || navigator.platform);

export function getRevealInfo(t, platformOverride) {
  const currentIsMac = platformOverride === 'darwin' || (platformOverride === undefined && isMac);
  const currentIsWin = platformOverride === 'windows' || (platformOverride === undefined && isWin);

  if (currentIsMac) {
    return {
      label: t('buttons.revealFinder') || 'В Finder',
      tooltip: t('buttons.revealMacTooltip') || 'Показать файл в Finder'
    };
  }
  if (currentIsWin) {
    return {
      label: t('buttons.revealExplorer') || 'В проводнике',
      tooltip: t('buttons.revealWinTooltip') || 'Показать файл в Проводнике'
    };
  }
  return {
    label: t('buttons.revealFolder') || t('buttons.reveal') || 'Показать в папке',
    tooltip: t('buttons.revealTooltip') || 'Показать файл на диске'
  };
}

// Validates an external URL before handing it to the OS browser.
// Only absolute http(s) URLs are accepted, blocking javascript:/data:/file: injection.
export function isValidExternalURL(target) {
  if (!target || typeof target !== 'string') return false;
  try {
    const parsed = new URL(target, 'http://localhost');
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}

export const openExternalURL = (target) => {
  if (!isValidExternalURL(target)) return;
  const safeTarget = target;
  if (typeof window !== 'undefined') {
    if (window.go?.main?.App?.OpenURL) {
      window.go.main.App.OpenURL(safeTarget);
    } else if (window.runtime?.BrowserOpenURL) {
      window.runtime.BrowserOpenURL(safeTarget);
    }
  }
};
