import { describe, it, expect } from 'vitest';
import {
  formatBytes,
  formatDate,
  formatSpeed,
  formatEta,
  getRevealInfo,
  isValidExternalURL
} from './formatters.js';

describe('formatBytes', () => {
  it('formats zero or negative bytes as 0 B', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(-10)).toBe('0 B');
    expect(formatBytes(null)).toBe('0 B');
    expect(formatBytes(undefined)).toBe('0 B');
  });

  it('formats bytes, kilobytes, megabytes and gigabytes accurately', () => {
    expect(formatBytes(500)).toBe('500 B');
    expect(formatBytes(1024)).toBe('1 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(1024 * 1024)).toBe('1 MB');
    expect(formatBytes(1024 * 1024 * 1024 * 2.5)).toBe('2.5 GB');
  });
});

describe('formatDate', () => {
  it('handles empty or invalid dates gracefully', () => {
    expect(formatDate(null)).toBe('');
    expect(formatDate(undefined)).toBe('');
    expect(formatDate('invalid-date')).toBe('');
  });

  it('formats valid ISO date string', () => {
    const formatted = formatDate('2026-09-01T12:00:00Z');
    expect(formatted).not.toBe('');
    expect(formatted).toContain('2026');
  });
});

describe('formatSpeed', () => {
  it('handles zero or negative speeds', () => {
    expect(formatSpeed(0)).toBe('0 B/s');
    expect(formatSpeed(-5)).toBe('0 B/s');
  });

  it('formats B/s, KB/s, and MB/s correctly', () => {
    expect(formatSpeed(500)).toBe('500 B/s');
    expect(formatSpeed(1024)).toBe('1.0 KB/s');
    expect(formatSpeed(1536)).toBe('1.5 KB/s');
    expect(formatSpeed(1024 * 1024 * 3.2)).toBe('3.2 MB/s');
  });
});

describe('formatEta', () => {
  it('formats empty for zero or negative seconds', () => {
    expect(formatEta(0)).toBe('');
    expect(formatEta(-10)).toBe('');
  });

  it('formats seconds, minutes and hours', () => {
    expect(formatEta(45)).toBe('45s');
    expect(formatEta(90)).toBe('1m 30s');
    expect(formatEta(3600)).toBe('1h 0m');
    expect(formatEta(3665)).toBe('1h 1m');
  });
});

describe('getRevealInfo', () => {
  const mockT = (key) => {
    const dict = {
      'buttons.revealFinder': 'В Finder',
      'buttons.revealExplorer': 'В проводнике',
      'buttons.revealFolder': 'Показать в папке',
      'buttons.revealMacTooltip': 'Показать файл в Finder',
      'buttons.revealWinTooltip': 'Показать файл в Проводнике',
      'buttons.revealTooltip': 'Показать файл на диске'
    };
    return dict[key] || key;
  };

  it('returns Finder info for macOS', () => {
    const info = getRevealInfo(mockT, 'darwin');
    expect(info.label).toBe('В Finder');
    expect(info.tooltip).toBe('Показать файл в Finder');
  });

  it('returns Explorer info for Windows', () => {
    const info = getRevealInfo(mockT, 'windows');
    expect(info.label).toBe('В проводнике');
    expect(info.tooltip).toBe('Показать файл в Проводнике');
  });

  it('returns generic folder info for Linux/other', () => {
    const info = getRevealInfo(mockT, 'linux');
    expect(info.label).toBe('Показать в папке');
    expect(info.tooltip).toBe('Показать файл на диске');
  });
});

describe('isValidExternalURL security check', () => {
  it('accepts valid http and https URLs', () => {
    expect(isValidExternalURL('https://geofabrik.de')).toBe(true);
    expect(isValidExternalURL('http://download.bbbike.org/osm/')).toBe(true);
  });

  it('rejects unsafe protocols and malicious payloads', () => {
    expect(isValidExternalURL('javascript:alert(1)')).toBe(false);
    expect(isValidExternalURL('data:text/html,<script>evil()</script>')).toBe(false);
    expect(isValidExternalURL('file:///etc/passwd')).toBe(false);
    expect(isValidExternalURL('')).toBe(false);
    expect(isValidExternalURL(null)).toBe(false);
  });
});
