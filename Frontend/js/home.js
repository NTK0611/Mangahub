/**
 * home.js — Browse & Discover page logic
 * Handles manga grid, genre filtering, search, and "continue reading" section.
 */

import {
  requireAuth,
  apiGetManga,
  apiGetLibrary,
  showToast,
  getUser,
} from './api.js';
import { renderUserPill, logout } from './auth.js';

const GENRES = ['All', 'Shounen', 'Shoujo', 'Seinen', 'Josei', 'Isekai', 'Horror', 'Action', 'Romance', 'Fantasy'];

let allManga     = [];
let activeGenre  = 'All';
let searchQuery  = '';

export async function initHome() {
  if (!requireAuth()) return;

  renderUserPill();
  renderGenreTabs();
  await loadManga();
  await loadContinueReading();

  // Search
  const searchInput = document.getElementById('search-input');
  searchInput?.addEventListener('input', (e) => {
    searchQuery = e.target.value.trim().toLowerCase();
    renderGrid();
  });

  // Mobile sidebar toggle
  document.getElementById('btn-menu')?.addEventListener('click', () => {
    document.querySelector('.sidebar')?.classList.toggle('open');
  });
}

// ── Genre Tabs ────────────────────────────────────────────────────────────
function renderGenreTabs() {
  const wrap = document.getElementById('genre-tabs');
  if (!wrap) return;

  wrap.innerHTML = GENRES.map(g => `
    <button class="filter-tab${g === activeGenre ? ' active' : ''}" data-genre="${g}">
      ${g}
    </button>
  `).join('');

  wrap.addEventListener('click', (e) => {
    const btn = e.target.closest('.filter-tab');
    if (!btn) return;
    activeGenre = btn.dataset.genre;
    wrap.querySelectorAll('.filter-tab').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    renderGrid();
  });
}

// ── Fetch manga from backend ──────────────────────────────────────────────
async function loadManga() {
  const grid = document.getElementById('manga-grid');
  if (!grid) return;

  grid.innerHTML = '<div class="loading-spinner"></div>';

  try {
    const data = await apiGetManga();
    allManga = Array.isArray(data) ? data : (data.data || []);
    window._allManga = allManga; // debug
    renderGrid();
  } catch (err) {
    grid.innerHTML = `
      <div class="empty-state" style="grid-column:1/-1">
        <div class="empty-icon">⚠️</div>
        <div class="empty-title">Couldn't load manga</div>
        <div class="empty-desc">${err.message}</div>
      </div>`;
  }
}

// ── Filter + render grid ──────────────────────────────────────────────────
function renderGrid() {
  const grid = document.getElementById('manga-grid');
  if (!grid) return;

  let list = allManga;

  // Genre filter
  if (activeGenre !== 'All') {
    list = list.filter(m =>
      Array.isArray(m.genres) &&
      m.genres.some(g => g.toLowerCase() === activeGenre.toLowerCase())
    );
  }

  // Search filter
  if (searchQuery) {
    list = list.filter(m =>
      m.title.toLowerCase().includes(searchQuery) ||
      (m.author || '').toLowerCase().includes(searchQuery)
    );
  }

  // Count badge
  const countEl = document.getElementById('manga-count');
  if (countEl) countEl.textContent = `${list.length} title${list.length !== 1 ? 's' : ''}`;

  if (list.length === 0) {
    grid.innerHTML = `
      <div class="empty-state" style="grid-column:1/-1">
        <div class="empty-icon">🔍</div>
        <div class="empty-title">No manga found</div>
        <div class="empty-desc">Try a different search or filter.</div>
      </div>`;
    return;
  }

  grid.innerHTML = list.map(m => mangaCardHTML(m)).join('');
}

// ── Manga card HTML ───────────────────────────────────────────────────────
function mangaCardHTML(m) {
  const statusClass = {
    ongoing:   'status-ongoing',
    completed: 'status-completed',
    hiatus:    'status-hiatus',
  }[m.status?.toLowerCase()] || 'status-ongoing';

  const genres = (m.genres || []).slice(0, 2);

  return `
    <div class="manga-card" onclick="goToManga('${m.id}')">
      <div class="manga-card-cover">
        ${m.cover_url
          ? `<img src="${m.cover_url}" alt="${escHtml(m.title)}" loading="lazy" onerror="this.parentElement.innerHTML='<div class=cover-placeholder>📖</div>'" />`
          : '<div class="cover-placeholder">📖</div>'}
        <span class="manga-status-badge ${statusClass}">${m.status || 'ongoing'}</span>
        <div class="manga-card-overlay">
          <div class="genre-tags-wrap">
            ${genres.map(g => `<span class="genre-tag">${escHtml(g)}</span>`).join('')}
          </div>
        </div>
      </div>
      <div class="manga-card-info">
        <div class="manga-card-title" title="${escHtml(m.title)}">${escHtml(m.title)}</div>
        <div class="manga-card-author">${escHtml(m.author || 'Unknown')}</div>
        <div class="manga-card-meta">
          <span class="manga-chapters">${m.total_chapters || '?'} ch</span>
        </div>
      </div>
    </div>
  `;
}

// ── Continue Reading section ──────────────────────────────────────────────
async function loadContinueReading() {
  const section = document.getElementById('continue-section');
  const grid    = document.getElementById('continue-grid');
  if (!section || !grid) return;

  try {
    const data = await apiGetLibrary();
    const items = data.data || [];
    const reading = items.filter(p => p.status === 'reading').slice(0, 6);

    if (reading.length === 0) {
      section.style.display = 'none';
      return;
    }

    section.style.display = 'block';
    grid.innerHTML = reading.map(p => continueCardHTML(p)).join('');
  } catch (_) {
    section.style.display = 'none';
  }
}

function continueCardHTML(p) {
  const pct = p.total_chapters > 0
    ? Math.min(100, Math.round((p.current_chapter / p.total_chapters) * 100))
    : 0;

  // Look up cover from already-loaded manga list
  const manga = allManga.find(m => m.id === p.manga_id);
  const coverUrl = manga?.cover_url || '';

  return `
    <div class="manga-card" onclick="goToManga('${p.manga_id}')">
      <div class="manga-card-cover">
        ${coverUrl
          ? `<img src="${coverUrl}" alt="${escHtml(p.title)}" loading="lazy" onerror="this.parentElement.innerHTML='<div class=cover-placeholder>📖</div>'" />`
          : '<div class="cover-placeholder">📖</div>'}
        <span class="manga-status-badge status-ongoing">Ch.${p.current_chapter}</span>
      </div>
      <div class="manga-card-info">
        <div class="manga-card-title">${escHtml(p.title)}</div>
        <div class="manga-card-author">${escHtml(p.author || '')}</div>
        <div class="progress-wrap">
          <div class="progress-fill" style="width:${pct}%"></div>
        </div>
        <div class="manga-card-meta" style="margin-top:4px">
          <span class="manga-chapters">${pct}% done</span>
        </div>
      </div>
    </div>
  `;
}

// ── Navigation ────────────────────────────────────────────────────────────
window.goToManga = (id) => {
  window.location.href = `/manga.html?id=${encodeURIComponent(id)}`;
};

// ── Utilities ─────────────────────────────────────────────────────────────
function escHtml(str) {
  return String(str || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}