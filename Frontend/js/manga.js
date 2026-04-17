/**
 * manga.js — Manga Detail page logic
 * Handles: metadata display, add-to-library, progress update, WebSocket chat
 */

import {
  requireAuth,
  apiGetMangaById,
  apiGetLibrary,
  apiAddToLibrary,
  apiUpdateProgress,
  wsUrl,
  showToast,
  getUser,
} from './api.js';
import { renderUserPill } from './auth.js';

let currentManga    = null;
let userProgress    = null;
let ws              = null;
let myUsername      = '';
let reconnectTimer  = null;
let wsRoomId        = '';

export async function initMangaDetail() {
  if (!requireAuth()) return;

  renderUserPill();

  const params = new URLSearchParams(window.location.search);
  const id = params.get('id');
  if (!id) {
    window.location.href = 'home.html';
    return;
  }

  wsRoomId = id;
  const user = getUser();
  myUsername = user?.username || '';

  await loadManga(id);
  await loadProgress(id);
  initChat(id);
}

// ── Manga detail ──────────────────────────────────────────────────────────
async function loadManga(id) {
  try {
    currentManga = await apiGetMangaById(id);
    renderManga(currentManga);
  } catch (err) {
    document.getElementById('manga-detail').innerHTML = `
      <div class="empty-state">
        <div class="empty-icon">⚠️</div>
        <div class="empty-title">Manga not found</div>
        <div class="empty-desc">${escHtml(err.message)}</div>
      </div>`;
  }
}

function renderManga(m) {
  document.title = `${m.title} — MangaHub`;

  const statusClass = {
    ongoing: 'status-ongoing',
    completed: 'status-completed',
    hiatus: 'status-hiatus',
  }[m.status?.toLowerCase()] || 'status-ongoing';

  document.getElementById('manga-detail').innerHTML = `
    <div class="manga-detail-hero">
      <div class="manga-cover-lg">
        ${m.cover_url
          ? `<img src="${m.cover_url}" alt="${escHtml(m.title)}" onerror="this.outerHTML='<div style=display:flex;align-items:center;justify-content:center;height:100%;font-size:4rem;color:var(--text-muted)>📖</div>'" />`
          : '<div style="display:flex;align-items:center;justify-content:center;height:100%;font-size:4rem;color:var(--text-muted)">📖</div>'}
      </div>

      <div class="manga-meta">
        <h1>${escHtml(m.title)}</h1>
        <div class="manga-meta-row">
          <span class="manga-status-badge ${statusClass}" style="position:static">${m.status || 'Ongoing'}</span>
          <span class="stat-chip blue">📖 ${m.total_chapters || '?'} chapters</span>
          ${m.author ? `<span class="stat-chip">✍️ ${escHtml(m.author)}</span>` : ''}
        </div>

        ${m.genres?.length ? `
          <div class="genre-tags-wrap" style="margin-bottom:14px">
            ${m.genres.map(g => `<span class="genre-tag">${escHtml(g)}</span>`).join('')}
          </div>` : ''}

        ${m.description ? `
          <p class="manga-description" id="manga-desc">${escHtml(m.description)}</p>
          <button class="btn btn-ghost btn-sm" onclick="toggleDesc()" id="desc-toggle" style="margin-bottom:14px;font-size:0.75rem">
            Show more ▾
          </button>` : ''}

        <div class="manga-actions" id="manga-actions">
          <!-- Populated after progress check -->
        </div>
      </div>
    </div>

    <!-- Progress section -->
    <div id="progress-section" style="display:none;margin-bottom:28px">
      <div class="card" style="padding:20px">
        <div class="section-title" style="margin-bottom:12px">Your Progress</div>
        <div style="display:flex;align-items:center;gap:16px;flex-wrap:wrap">
          <div>
            <div style="font-size:0.78rem;color:var(--text-muted);margin-bottom:4px">Current Chapter</div>
            <div style="font-size:1.5rem;font-weight:700;font-family:'Rajdhani',sans-serif" id="prog-chapter">—</div>
          </div>
          <div style="flex:1;min-width:120px">
            <div style="font-size:0.78rem;color:var(--text-muted);margin-bottom:6px" id="prog-pct-label">0% complete</div>
            <div class="progress-wrap" style="height:8px">
              <div class="progress-fill" id="prog-bar" style="width:0%"></div>
            </div>
          </div>
          <div id="prog-status-badge"></div>
        </div>
      </div>
    </div>
  `;

  // Desc toggle
  window.toggleDesc = () => {
    const desc = document.getElementById('manga-desc');
    const btn  = document.getElementById('desc-toggle');
    if (desc.style.webkitLineClamp === 'unset') {
      desc.style.webkitLineClamp = '4';
      desc.style.overflow = 'hidden';
      btn.textContent = 'Show more ▾';
    } else {
      desc.style.webkitLineClamp = 'unset';
      desc.style.overflow = 'visible';
      btn.textContent = 'Show less ▴';
    }
  };
}

async function loadProgress(mangaId) {
  try {
    const data = await apiGetLibrary();
    const items = data.data || [];
    userProgress = items.find(p => p.manga_id === mangaId) || null;
    renderActions();
    if (userProgress) renderProgress();
  } catch (_) {
    renderActions();
  }
}

function renderActions() {
  const wrap = document.getElementById('manga-actions');
  if (!wrap || !currentManga) return;

  if (!userProgress) {
    wrap.innerHTML = `
      <button class="btn btn-primary" onclick="addToLibrary('reading')">
        ➕ Add to Library
      </button>
      <button class="btn btn-secondary" onclick="addToLibrary('plan_to_read')">
        🔖 Plan to Read
      </button>
    `;
  } else {
    wrap.innerHTML = `
      <button class="btn btn-orange" onclick="openProgressModal()">
        📝 Update Progress
      </button>
      <button class="btn btn-ghost btn-sm" onclick="addToLibrary('${userProgress.status}')">
        ✓ In Library
      </button>
    `;
  }
}

function renderProgress() {
  if (!userProgress || !currentManga) return;
  const section = document.getElementById('progress-section');
  if (section) section.style.display = 'block';

  const chapEl  = document.getElementById('prog-chapter');
  const barEl   = document.getElementById('prog-bar');
  const pctEl   = document.getElementById('prog-pct-label');
  const badgeEl = document.getElementById('prog-status-badge');

  const total = currentManga.total_chapters || 0;
  const curr  = userProgress.current_chapter || 0;
  const pct   = total > 0 ? Math.min(100, Math.round((curr / total) * 100)) : 0;

  if (chapEl)  chapEl.textContent  = `Ch.${curr} / ${total || '?'}`;
  if (barEl)   barEl.style.width   = `${pct}%`;
  if (pctEl)   pctEl.textContent   = `${pct}% complete`;
  if (badgeEl) badgeEl.innerHTML   = `<span class="stat-chip ${userProgress.status === 'completed' ? 'green' : 'blue'}">${formatStatus(userProgress.status)}</span>`;
}

// ── Library actions ───────────────────────────────────────────────────────
window.addToLibrary = async (status) => {
  if (!currentManga) return;
  const btn = document.querySelector('#manga-actions .btn-primary, #manga-actions .btn-secondary');
  if (btn) { btn.disabled = true; btn.textContent = 'Adding…'; }

  try {
    await apiAddToLibrary(currentManga.id, status, userProgress?.current_chapter || 0);
    showToast('Added to library!', `${currentManga.title} added as "${formatStatus(status)}"`, 'success');

    // Refresh progress
    const data = await apiGetLibrary();
    const items = data.data || [];
    userProgress = items.find(p => p.manga_id === currentManga.id) || null;
    renderActions();
    renderProgress();
  } catch (err) {
    showToast('Error', err.message, 'error');
    if (btn) { btn.disabled = false; btn.textContent = 'Add to Library'; }
  }
};

window.openProgressModal = () => {
  const modal = document.getElementById('progress-modal');
  if (!modal || !currentManga) return;

  document.getElementById('pm-chapter').value = userProgress?.current_chapter || 0;
  document.getElementById('pm-chapter').max   = currentManga.total_chapters || 9999;
  document.getElementById('pm-status').value  = userProgress?.status || 'reading';
  document.getElementById('pm-total').textContent = `of ${currentManga.total_chapters || '?'} chapters`;

  modal.style.display = 'flex';
};

// ── WebSocket Chat ────────────────────────────────────────────────────────
function initChat(roomId) {
  connectWS(roomId);

  const form  = document.getElementById('chat-form');
  const input = document.getElementById('chat-input');

  form?.addEventListener('submit', (e) => {
    e.preventDefault();
    const msg = input.value.trim();
    if (!msg) return;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ message: msg }));
      input.value = '';
    } else {
      showToast('Not connected', 'Reconnecting to chat…', 'warning');
      connectWS(roomId);
    }
  });
}

function connectWS(roomId) {
  setWsStatus('connecting');

  const url = wsUrl(roomId);
  ws = new WebSocket(url);

  ws.onopen = () => {
    setWsStatus('connected');
    clearTimeout(reconnectTimer);
  };

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      appendChatMessage(msg);
    } catch (_) {}
  };

  ws.onclose = () => {
    setWsStatus('disconnected');
    reconnectTimer = setTimeout(() => connectWS(roomId), 5000);
  };

  ws.onerror = () => {
    ws.close();
  };
}

function setWsStatus(status) {
  const dot    = document.getElementById('ws-dot');
  const label  = document.getElementById('ws-status-label');
  if (!dot) return;

  dot.className = `ws-dot ${status === 'connected' ? 'connected' : status === 'connecting' ? 'connecting' : ''}`;

  const labels = {
    connected:    '🟢 Live',
    connecting:   '🟡 Connecting…',
    disconnected: '🔴 Disconnected',
  };
  if (label) label.textContent = labels[status] || '';
}

function appendChatMessage(msg) {
  const container = document.getElementById('chat-messages');
  if (!container) return;

  const isOwn    = msg.username === myUsername;
  const isSystem = msg.type === 'join' || msg.type === 'leave';

  const el = document.createElement('div');

  if (isSystem) {
    el.className = 'chat-msg system';
    el.innerHTML = `<div class="chat-bubble">💬 ${escHtml(msg.message)}</div>`;
  } else {
    el.className = `chat-msg${isOwn ? ' own' : ''}`;
    const initials = (msg.username || '?').charAt(0).toUpperCase();
    const time = new Date(msg.timestamp * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

    el.innerHTML = `
      <div class="chat-avatar">${initials}</div>
      <div class="chat-msg-body">
        <div class="chat-msg-header">
          <span class="chat-username">${escHtml(msg.username)}</span>
          <span class="chat-time">${time}</span>
        </div>
        <div class="chat-bubble">${escHtml(msg.message)}</div>
      </div>
    `;
  }

  container.appendChild(el);
  container.scrollTop = container.scrollHeight;
}

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

function escHtml(str) {
  return String(str || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}