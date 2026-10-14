import { describe, it, expect } from 'vitest';
import { translations, getI18n } from './i18n.js';

describe('i18n translations system', () => {
  it('contains both ru and en translations', () => {
    expect(translations).toHaveProperty('ru');
    expect(translations).toHaveProperty('en');
  });

  function getKeys(obj, prefix = '') {
    let result = [];
    for (const key of Object.keys(obj)) {
      const fullPath = prefix ? `${prefix}.${key}` : key;
      if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
        result = result.concat(getKeys(obj[key], fullPath));
      } else {
        result.push(fullPath);
      }
    }
    return result;
  }

  it('has identical structure and keys between ru and en', () => {
    const ruKeys = getKeys(translations.ru).sort();
    const enKeys = getKeys(translations.en).sort();

    const missingInEn = ruKeys.filter((k) => !enKeys.includes(k));
    const missingInRu = enKeys.filter((k) => !ruKeys.includes(k));

    expect(missingInEn, `Keys missing in English: ${missingInEn.join(', ')}`).toEqual([]);
    expect(missingInRu, `Keys missing in Russian: ${missingInRu.join(', ')}`).toEqual([]);
  });

  it('correctly resolves nested keys using getI18n', () => {
    const tRu = getI18n('ru');
    const tEn = getI18n('en');

    expect(tRu('nav.catalog')).toBe('Каталог карт');
    expect(tEn('nav.catalog')).toBe('Map Catalog');

    expect(tRu('buttons.download')).toBe('Загрузить');
    expect(tEn('buttons.download')).toBe('Download');
  });

  it('returns key string as fallback when key does not exist', () => {
    const t = getI18n('en');
    expect(t('nonexistent.key')).toBe('nonexistent.key');
    expect(t('nav.doesNotExist')).toBe('nav.doesNotExist');
  });

  it('falls back to English when an unknown language is passed', () => {
    const t = getI18n('fr');
    expect(t('nav.catalog')).toBe('Map Catalog');
  });
});
