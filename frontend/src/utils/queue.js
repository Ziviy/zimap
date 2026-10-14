export function isMapQueued(name, queue) {
  if (!name || !Array.isArray(queue)) return false;
  const clean = name.replace(/\.img$/i, '').toLowerCase();
  return queue.some(item => item.name && item.name.replace(/\.img$/i, '').toLowerCase() === clean);
}

export function createQueueItem(url, name, heapSize = '', detailLevel = 'standard') {
  return {
    id: `${name}-${Date.now()}-${Math.random().toString(36).substr(2, 5)}`,
    url,
    name,
    heapSize,
    detailLevel,
    addedAt: Date.now()
  };
}

export function enqueueJob(queue, item) {
  if (isMapQueued(item.name, queue)) {
    return queue;
  }
  return [...queue, item];
}

export function dequeueJob(queue) {
  if (!queue || queue.length === 0) return { nextJob: null, remainingQueue: [] };
  const [nextJob, ...remainingQueue] = queue;
  return { nextJob, remainingQueue };
}

export function removeJobFromQueue(queue, jobId) {
  if (!Array.isArray(queue)) return [];
  return queue.filter(item => item.id !== jobId && item.name !== jobId);
}
