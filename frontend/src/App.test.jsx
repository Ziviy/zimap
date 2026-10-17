import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { Children } from 'react';
import App from './App';
import * as WailsApp from '../wailsjs/go/main/App';

const hooks = vi.hoisted(() => ({ view: 'catalog', setState: vi.fn() }));

// Exercise the bound IPC callbacks without mounting views or running startup effects.
vi.mock('react', async (importOriginal) => ({
  ...await importOriginal(),
  useState: (initial) => [
    initial === 'catalog' ? hooks.view : typeof initial === 'function' ? initial() : initial,
    hooks.setState,
  ],
  useRef: (initial) => ({ current: initial }),
  useEffect: vi.fn(),
}));

vi.mock('../wailsjs/go/main/App', () => ({
  ConvertURL: vi.fn(),
  ConvertLocal: vi.fn(),
}));

function findAction(element, name) {
  if (!element?.props) return null;
  if (typeof element.props[name] === 'function') return element.props[name];
  for (const child of Children.toArray(element.props.children)) {
    const action = findAction(child, name);
    if (action) return action;
  }
  return null;
}

const builds = [
  ['catalog', 'onStartBuild', 'ConvertURL'],
  ['local', 'onStartLocalBuild', 'ConvertLocal'],
];

describe('conversion IPC error ownership', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.stubGlobal('localStorage', { getItem: () => null });
  });
  afterEach(() => vi.unstubAllGlobals());

  it.each(builds)('%s ignores a previous job rejection after another job starts', async (view, action, binding) => {
    hooks.view = view;
    let rejectPrevious;
    const previous = new Promise((_, reject) => { rejectPrevious = reject; });
    WailsApp[binding].mockReturnValueOnce(previous).mockResolvedValueOnce();
    const start = findAction(App(), action);

    start('first-input', 'First map', '4g');
    start('next-input', 'Next map', '4g');
    hooks.setState.mockClear();

    rejectPrevious(new Error('Previous job failed'));
    await previous.catch(() => {});
    expect(hooks.setState).not.toHaveBeenCalled();
  });

  it.each(builds)('%s still clears running state when the current job rejects', async (view, action, binding) => {
    hooks.view = view;
    let rejectCurrent;
    const current = new Promise((_, reject) => { rejectCurrent = reject; });
    WailsApp[binding].mockReturnValueOnce(current);
    const start = findAction(App(), action);

    start('current-input', 'Current map', '4g');
    hooks.setState.mockClear();

    rejectCurrent(new Error('Current job failed'));
    await current.catch(() => {});
    expect(hooks.setState).toHaveBeenCalledWith(false);
    expect(hooks.setState).toHaveBeenCalledWith(null);
  });
});
