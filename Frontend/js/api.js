/**
 * api.js — Central fetch wrapper for MangaHub backend (port 8080)
 * Auto-attaches JWT, handles errors, shows toasts.
 */

// Auto-detect: use production URL when not on localhost
const API_BASE = window.location.hostname === 'localhost'
  ? 'http://localhost:8080'
  : 'https://mangahub-whua.onrender.com';

// ── JWT helpers ───────────────────────────────────────────────────────────
export function getToken() {
  return localStorage.getItem('mangahub_token');
}

export function getUser() {
  const raw = localStorage.getItem('mangahub_user');
  return raw ? JSON.parse(raw) : null;
}

export function setAuth(token, userId, username) {
  localStorage.setItem('mangahub_token', token);
  localStorage.setItem('mangahub_user', JSON.stringify({ id: userId, username }));
}

export function clearAuth() {
  localStorage.removeItem('mangahub_token');
  localStorage.removeItem('mangahub_user');
}

export function isLoggedIn() {
  return !!getToken();
}

export function requireAuth() {
  if (!isLoggedIn()) {
    window.location.href = 'index.html';
    return false;
  }
  return true;
}

// ── Core fetch wrapper ────────────────────────────────────────────────────
async function request(method, path, body = null, auth = true) {
  const headers = { 'Content-Type': 'application/json' };

  if (auth) {
    const token = getToken();
    if (!token) {
      clearAuth();
      window.location.href = 'index.html';
      throw new Error('Not authenticated');
    }
    headers['Authorization'] = `Bearer ${token}`;
  }

  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);

  try {
    const res = await fetch(`${API_BASE}${path}`, opts);

    if (res.status === 401) {
      clearAuth();
      window.location.href = 'index.html';
      throw new Error('Session expired');
    }

    const data = await res.json().catch(() => ({}));

    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }

    return data;
  } catch (err) {
    if (err.message !== 'Session expired' && err.message !== 'Not authenticated') {
      // bubble up — callers decide whether to show toast
    }
    throw err;
  }
}

// ── Auth endpoints ────────────────────────────────────────────────────────
export async function apiRegister(username, password) {
  return request('POST', '/auth/register', { username, password }, false);
}

export async function apiLogin(username, password) {
  const data = await request('POST', '/auth/login', { username, password }, false);
  return data; // { token, user_id }
}

// ── Manga endpoints ───────────────────────────────────────────────────────
export async function apiGetManga(search = '', status = '') {
  const params = new URLSearchParams();
  if (search) params.set('search', search);
  if (status) params.set('status', status);
  const qs = params.toString() ? `?${params}` : '';
  return request('GET', `/manga${qs}`);
}

export async function apiGetMangaById(id) {
  return request('GET', `/manga/${id}`);
}

// ── Library / Progress endpoints ──────────────────────────────────────────
export async function apiGetLibrary() {
  return request('GET', '/users/library');
}

export async function apiAddToLibrary(mangaId, status = 'reading', currentChapter = 0) {
  return request('POST', '/users/library', {
    manga_id: mangaId,
    status,
    current_chapter: currentChapter,
  });
}

export async function apiUpdateProgress(mangaId, currentChapter, status) {
  return request('PUT', '/users/progress', {
    manga_id: mangaId,
    current_chapter: currentChapter,
    status,
  });
}

// ── Notifications ─────────────────────────────────────────────────────────
export async function apiSendNotification(mangaId, message, notifType = 'info') {
  return request('POST', '/notifications/send', {
    manga_id: mangaId,
    message,
    type: notifType,
  });
}

// ── Health ────────────────────────────────────────────────────────────────
export async function apiHealth() {
  const res = await fetch(`${API_BASE}/health`);
  return res.json();
}

// ── WebSocket URL builder ─────────────────────────────────────────────────
export function wsUrl(roomId) {
  const token = getToken();
  const wsProto = window.location.hostname === 'localhost' ? 'ws:' : 'wss:';
  const wsHost = window.location.hostname === 'localhost'
    ? 'localhost:8080'
    : 'mangahub-whua.onrender.com';
  return `${wsProto}//${wsHost}/ws/chat/${roomId}?token=${token}`;
}

// ── Toast notification system ─────────────────────────────────────────────
function ensureToastContainer() {
  let c = document.getElementById('toast-container');
  if (!c) {
    c = document.createElement('div');
    c.id = 'toast-container';
    document.body.appendChild(c);
  }
  return c;
}

export function showToast(title, msg = '', type = 'info', duration = 4000) {
  const container = ensureToastContainer();

  const icons = {
    success: '✅',
    error: '❌',
    info: '💬',
    warning: '⚠️',
    notification: '🔔',
  };

  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.innerHTML = `
    <div class="toast-icon">${icons[type] || '💬'}</div>
    <div class="toast-body">
      <div class="toast-title">${title}</div>
      ${msg ? `<div class="toast-msg">${msg}</div>` : ''}
    </div>
  `;

  container.appendChild(toast);

  const remove = () => {
    toast.classList.add('removing');
    setTimeout(() => toast.remove(), 300);
  };

  toast.addEventListener('click', remove);
  setTimeout(remove, duration);
  // Keep Render backend alive — ping every 10 minutes
if (window.location.hostname !== 'localhost') {
  setInterval(() => {
    fetch(`${API_BASE}/health`).catch(() => {});
  }, 10 * 60 * 1000);
}
}