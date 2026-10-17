import { afterEach, describe, it, expect, vi } from 'vitest';
import {
  isMapQueued,
  createQueueItem,
  enqueueJob,
  dequeueJob,
  scheduleNextJob,
  removeJobFromQueue
} from './queue.js';

describe('queue scheduling', () => {
  afterEach(() => vi.useRealTimers());

  it('starts only the first job and leaves the queue untouched until the timer fires', () => {
    vi.useFakeTimers();
    const queue = [{ name: 'Germany' }, { name: 'France' }];
    const startJob = vi.fn();

    scheduleNextJob(queue, startJob);
    expect(startJob).not.toHaveBeenCalled();
    expect(queue).toHaveLength(2);

    vi.runAllTimers();
    expect(startJob).toHaveBeenCalledExactlyOnceWith(queue[0], [queue[1]]);
    expect(queue).toHaveLength(2);
  });

  it('cancels a pending start when the effect is cleaned up', () => {
    vi.useFakeTimers();
    const startJob = vi.fn();
    const cleanup = scheduleNextJob([{ name: 'Germany' }], startJob);

    cleanup();
    vi.runAllTimers();
    expect(startJob).not.toHaveBeenCalled();
  });
});

describe('queue utilities', () => {
  it('identifies if a map is queued case-insensitively and ignoring .img', () => {
    const queue = [
      { id: '1', name: 'germany' },
      { id: '2', name: 'moscow.img' }
    ];

    expect(isMapQueued('germany', queue)).toBe(true);
    expect(isMapQueued('GERMANY', queue)).toBe(true);
    expect(isMapQueued('germany.img', queue)).toBe(true);
    expect(isMapQueued('moscow', queue)).toBe(true);
    expect(isMapQueued('france', queue)).toBe(false);
    expect(isMapQueued(null, queue)).toBe(false);
    expect(isMapQueued('  ', queue)).toBe(false);
  });

  it('creates valid queue item with defaults', () => {
    const item = createQueueItem('https://example.com/map.pbf', 'austria');
    expect(item.name).toBe('austria');
    expect(item.url).toBe('https://example.com/map.pbf');
    expect(item.detailLevel).toBe('standard');
    expect(item.id).toContain('austria');
    expect(typeof item.addedAt).toBe('number');
  });

  it('enqueues job and prevents duplicates', () => {
    const initialQueue = [];
    const item1 = createQueueItem('https://example.com/1.pbf', 'italy');
    const q1 = enqueueJob(initialQueue, item1);
    expect(q1.length).toBe(1);

    // Try enqueuing same map again
    const item2 = createQueueItem('https://example.com/1.pbf', 'italy');
    const q2 = enqueueJob(q1, item2);
    expect(q2.length).toBe(1);

    // Enqueue different map
    const item3 = createQueueItem('https://example.com/2.pbf', 'spain');
    const q3 = enqueueJob(q2, item3);
    expect(q3.length).toBe(2);
  });

  it('deduplicates jobs by URL as well as normalized map name', () => {
    const item = createQueueItem('https://example.com/map.pbf', ' Germany.img ');
    const queue = [item];

    expect(enqueueJob(queue, createQueueItem(item.url, 'another name'))).toBe(queue);
    expect(enqueueJob(queue, createQueueItem('https://example.com/other.pbf', 'GERMANY'))).toBe(queue);
    expect(isMapQueued(' germany ', queue)).toBe(true);
  });

  it('dequeues next job in FIFO order', () => {
    const item1 = createQueueItem('https://example.com/1.pbf', 'italy');
    const item2 = createQueueItem('https://example.com/2.pbf', 'spain');
    const queue = [item1, item2];

    const { nextJob, remainingQueue } = dequeueJob(queue);
    expect(nextJob.name).toBe('italy');
    expect(remainingQueue.length).toBe(1);
    expect(remainingQueue[0].name).toBe('spain');

    const empty = dequeueJob([]);
    expect(empty.nextJob).toBeNull();
    expect(empty.remainingQueue).toEqual([]);
  });

  it('removes jobs by URL or normalized map name without changing the original queue', () => {
    const queue = [
      { id: 'job-1', name: 'Germany.img', url: 'https://example.com/de.pbf' },
      { id: 'job-2', name: 'France', url: 'https://example.com/fr.pbf' },
    ];

    expect(removeJobFromQueue(queue, queue[0].url)).toEqual([queue[1]]);
    expect(removeJobFromQueue(queue, ' GERMANY ')).toEqual([queue[1]]);
    expect(queue).toHaveLength(2);
  });

  it('removes job from queue by id or name', () => {
    const item1 = { id: 'job-1', name: 'italy' };
    const item2 = { id: 'job-2', name: 'spain' };
    const queue = [item1, item2];

    const updated = removeJobFromQueue(queue, 'job-1');
    expect(updated.length).toBe(1);
    expect(updated[0].name).toBe('spain');

    const updatedByName = removeJobFromQueue(queue, 'spain');
    expect(updatedByName.length).toBe(1);
    expect(updatedByName[0].name).toBe('italy');
  });
});
