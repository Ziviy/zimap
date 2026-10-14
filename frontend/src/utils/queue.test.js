import { describe, it, expect } from 'vitest';
import {
  isMapQueued,
  createQueueItem,
  enqueueJob,
  dequeueJob,
  removeJobFromQueue
} from './queue.js';

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
