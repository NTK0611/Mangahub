/**
 * library.js — My Library page logic
 * Shows user's reading list, progress bars, quick chapter update.
 */

import {
  requireAuth,
  apiGetLibrary,
  apiGetManga,
  apiUpdateProgress,
  showToast,
} from './api.js';
import { renderUserPill } from './auth.js';

let libraryData = [];
let allManga    = [];
let activeTab   = 'all';

const STATUS_TABS = [
  { id: 'all',          label: 'All',           icon: '📚' },
  { id: 'reading',      label: 'Reading',        icon: '📖' },
  { id: 'completed',    label: 'Completed',      icon: '✅' },
  { id: 'plan_to_read', label: 'Plan to Read',   icon: '🔖' },
  { id: 'on_hold',      label: 'On Hold',        icon: '⏸' },
  { id: 'dropped',      label: 'Dropped',        icon: '🗑' },
];

export async function initLibrary() {
  if (!requireAuth()) return;

  renderUserPill();
  renderTabs();
  await loadLibrary();

  document.getElementById('btn-menu')?.addEventListener('click', () => {
    document.querySelector('.sidebar')?.classList.toggle('open');
  });
}

// ── Tabs ──────────────────────────────────────────────────────────────────
function renderTabs() {
  const wrap = document.getElementById('library-tabs');
  if (!wrap) return;

  wrap.innerHTML = STATUS_TABS.map(t => `
    <button class="library-tab${t.id === activeTab ? ' active' : ''}" data-tab="${t.id}">
      ${t.icon} ${t.label}
      <span class="tab-count" id="tab-count-${t.id}" style="margin-left:5px;font-size:0.72rem;opacity:0.7"></span>
    </button>
  `).join('');

  wrap.addEventListener('click', (e) => {
    const btn = e.target.closest('.library-tab');
    if (!btn) return;
    activeTab = btn.dataset.tab;
    wrap.querySelectorAll('.library-tab').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    renderList();
  });
}

// ── Fetch data ────────────────────────────────────────────────────────────
async function loadLibrary() {
  const listWrap = document.getElementById('library-list');
  listWrap.innerHTML = '<div class="loading-spinner"></div>';

  try {
    const [libData, mangaData] = await Promise.all([
      apiGetLibrary(),
      apiGetManga().catch(() => []),
    ]);

    libraryData = libData.data || [];
    allManga    = Array.isArray(mangaData) ? mangaData : (mangaData.data || []);

    updateCounts();
    renderList();
  } catch (err) {
    listWrap.innerHTML = `
      <div class="empty-state">
        <div class="empty-icon">⚠️</div>
        <div class="empty-title">Couldn't load library</div>
        <div class="empty-desc">${escHtml(err.message)}</div>
      </div>`;
  }
}

function updateCounts() {
  STATUS_TABS.forEach(t => {
    const el = document.getElementById(`tab-count-${t.id}`);
    if (!el) return;
    const count = t.id === 'all'
      ? libraryData.length
      : libraryData.filter(p => p.status === t.id).length;
    el.textContent = count > 0 ? `(${count})` : '';
  });

  // Summary stats
  const totalEl    = document.getElementById('stat-total');
  const readingEl  = document.getElementById('stat-reading');
  const doneEl     = document.getElementById('stat-done');

  if (totalEl)   totalEl.textContent   = libraryData.length;
  if (readingEl) readingEl.textContent = libraryData.filter(p => p.status === 'reading').length;
  if (doneEl)    doneEl.textContent    = libraryData.filter(p => p.status === 'completed').length;
}

// ── Render list ───────────────────────────────────────────────────────────
function renderList() {
  const wrap = document.getElementById('library-list');
  if (!wrap) return;

  const filtered = activeTab === 'all'
    ? libraryData
    : libraryData.filter(p => p.status === activeTab);

  if (filtered.length === 0) {
    wrap.innerHTML = `
      <div class="empty-state">
        <div class="empty-icon">${activeTab === 'all' ? '📚' : STATUS_TABS.find(t => t.id === activeTab)?.icon || '📚'}</div>
        <div class="empty-title">Nothing here yet</div>
        <div class="empty-desc">
          ${activeTab === 'all'
            ? 'Start browsing and add manga to your library!'
            : `No manga in "${STATUS_TABS.find(t => t.id === activeTab)?.label}" yet.`}
        </div>
        ${activeTab === 'all' ? '<a href="home.html" class="btn btn-primary" style="margin-top:16px">Browse Manga</a>' : ''}
      </div>`;
    return;
  }

  wrap.innerHTML = filtered.map(p => libraryItemHTML(p)).join('');
}

function libraryItemHTML(p) {
  const manga    = allManga.find(m => m.id === p.manga_id);
  const coverUrl = manga?.cover_url || '';
  const total    = p.total_chapters || manga?.total_chapters || 0;
  const curr     = p.current_chapter || 0;
  const pct      = total > 0 ? Math.min(100, Math.round((curr / total) * 100)) : 0;

  const statusColors = {
    reading:      'blue',
    completed:    'green',
    plan_to_read: '',
    on_hold:      'orange',
    dropped:      '',
  };

  return `
    <div class="library-item" onclick="goToManga('${p.manga_id}')">
      <div class="library-item-cover">
        ${coverUrl
          ? `<img src="${coverUrl}" alt="${escHtml(p.title)}" loading="lazy" onerror="this.outerHTML='<div style=display:flex;align-items:center;justify-content:center;height:100%;font-size:1.5rem>📖</div>'" />`
          : '<div style="display:flex;align-items:center;justify-content:center;height:100%;font-size:1.5rem">📖</div>'}
      </div>

      <div style="flex:1;min-width:0">
        <div class="library-item-title">${escHtml(p.title)}</div>
        <div class="library-item-author">${escHtml(p.author || 'Unknown')}</div>
        <div style="display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-bottom:8px">
          <span class="stat-chip ${statusColors[p.status] || ''}">${formatStatus(p.status)}</span>
          <span style="font-size:0.75rem;color:var(--text-muted)">Ch.${curr} / ${total || '?'}</span>
        </div>
        <div class="progress-wrap">
          <div class="progress-fill" style="width:${pct}%"></div>
        </div>
        <div style="font-size:0.72rem;color:var(--text-muted);margin-top:3px">${pct}% complete</div>
      </div>

      <div class="library-item-actions" onclick="event.stopPropagation()">
        <button class="btn btn-secondary btn-sm" onclick="openQuickUpdate('${escHtml(p.manga_id)}', ${curr}, '${p.status}', ${total})">
          +1 Ch
        </button>
        <span style="font-size:0.7rem;color:var(--text-muted)">${formatDate(p.last_updated)}</span>
      </div>
    </div>
  `;
}

// ── Quick +1 chapter update ───────────────────────────────────────────────
window.openQuickUpdate = async (mangaId, currentChapter, status, total) => {
  const newChapter = currentChapter + 1;
  const newStatus  = (total > 0 && newChapter >= total) ? 'completed' : status;

  try {
    await apiUpdateProgress(mangaId, newChapter, newStatus);
    showToast(
      'Progress saved!',
      `Chapter ${newChapter}${newStatus === 'completed' ? ' — Completed! 🎉' : ''}`,
      newStatus === 'completed' ? 'success' : 'info'
    );
    await loadLibrary();
  } catch (err) {
    showToast('Error', err.message, 'error');
  }
};

window.goToManga = (id) => {
  window.location.href = `/manga.html?id=${encodeURIComponent(id)}`;
};

// ── Helpers ───────────────────────────────────────────────────────────────
function formatStatus(s) {
  return {
    reading:      '📖 Reading',
    completed:    '✅ Completed',
    plan_to_read: '🔖 Plan to Read',
    on_hold:      '⏸ On Hold',
    dropped:      '🗑 Dropped',
  }[s] || s;
}

function formatDate(str) {
  if (!str) return '';
  try {
    return new Date(str).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  } catch (_) {
    return '';
  }
}

function escHtml(str) {
  return String(str || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}