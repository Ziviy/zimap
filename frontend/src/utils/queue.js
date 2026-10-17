function normalizeMapName(name) {
  return (name || '').trim().replace(/\.img$/i, '').toLowerCase();
}

export function isMapQueued(name, queue) {
  if (!name || !Array.isArray(queue)) return false;
  const clean = normalizeMapName(name);
  return clean !== '' && queue.some(item => normalizeMapName(item.name) === clean);
}

export function createQueueItem(url, name, heapSize = '', detailLevel = 'standard') {
  return {
    id: `${name}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    url,
    name,
    heapSize,
    detailLevel,
    addedAt: Date.now()
  };
}

export function enqueueJob(queue, item) {
  if (isMapQueued(item.name, queue) || (item.url && queue.some(queued => queued.url === item.url))) {
    return queue;
  }
  return [...queue, item];
}

export function dequeueJob(queue) {
  if (!queue || queue.length === 0) return { nextJob: null, remainingQueue: [] };
  const [nextJob, ...remainingQueue] = queue;
  return { nextJob, remainingQueue };
}

export function scheduleNextJob(queue, startJob) {
  const timer = setTimeout(() => {
    const { nextJob, remainingQueue } = dequeueJob(queue);
    startJob(nextJob, remainingQueue);
  }, 100);
  return () => clearTimeout(timer);
}

export function removeJobFromQueue(queue, jobId) {
  if (!Array.isArray(queue)) return [];
  const clean = normalizeMapName(jobId);
  return queue.filter(item => item.id !== jobId && item.url !== jobId && normalizeMapName(item.name) !== clean);
}
