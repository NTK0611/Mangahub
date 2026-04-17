/**
 * auth.js — Login, register, logout logic for MangaHub
 */

import {
  apiLogin,
  apiRegister,
  setAuth,
  clearAuth,
  isLoggedIn,
  showToast,
  getUser,
} from './api.js';

export function initAuth() {
  if (isLoggedIn()) {
    window.location.href = 'home.html';
    return;
  }

  const loginTab    = document.getElementById('tab-login');
  const registerTab = document.getElementById('tab-register');
  const loginForm   = document.getElementById('form-login');
  const regForm     = document.getElementById('form-register');

  loginTab?.addEventListener('click', () => switchTab('login'));
  registerTab?.addEventListener('click', () => switchTab('register'));

  function switchTab(tab) {
    const isLogin = tab === 'login';
    loginTab.classList.toggle('active', isLogin);
    registerTab.classList.toggle('active', !isLogin);
    loginForm.classList.toggle('active', isLogin);
    regForm.classList.toggle('active', !isLogin);
    clearErrors();
  }

  function clearErrors() {
    document.querySelectorAll('.auth-error').forEach(el => {
      el.style.display = 'none';
      el.textContent = '';
    });
  }

  function showError(formId, msg) {
    const el = document.getElementById('error-' + formId);
    if (el) {
      el.textContent = msg;
      el.style.display = 'block';
    }
  }

  loginForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearErrors();
    const username = document.getElementById('login-username').value.trim();
    const password = document.getElementById('login-password').value;
    const btn = loginForm.querySelector('button[type=submit]');
    if (!username || !password) { showError('login', 'Please fill in all fields.'); return; }
    btn.disabled = true; btn.textContent = 'Signing in\u2026';
    try {
      const data = await apiLogin(username, password);
      setAuth(data.token, data.user_id, username);
      showToast('Welcome back!', 'Logged in as ' + username, 'success');
      setTimeout(() => { window.location.href = 'home.html'; }, 600);
    } catch (err) {
      showError('login', err.message || 'Invalid username or password.');
    } finally {
      btn.disabled = false; btn.textContent = 'Sign In';
    }
  });

  regForm?.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearErrors();
    const username = document.getElementById('reg-username').value.trim();
    const password = document.getElementById('reg-password').value;
    const confirm  = document.getElementById('reg-confirm').value;
    const btn = regForm.querySelector('button[type=submit]');
    if (!username || !password) { showError('register', 'Please fill in all fields.'); return; }
    if (password !== confirm) { showError('register', 'Passwords do not match.'); return; }
    if (password.length < 4) { showError('register', 'Password must be at least 4 characters.'); return; }
    btn.disabled = true; btn.textContent = 'Creating account\u2026';
    try {
      await apiRegister(username, password);
      const loginData = await apiLogin(username, password);
      setAuth(loginData.token, loginData.user_id, username);
      showToast('Account created!', 'Welcome to MangaHub, ' + username + '!', 'success');
      setTimeout(() => { window.location.href = 'home.html'; }, 600);
    } catch (err) {
      showError('register', err.message || 'Registration failed. Username may already exist.');
    } finally {
      btn.disabled = false; btn.textContent = 'Create Account';
    }
  });
}

export function logout() {
  clearAuth();
  window.location.href = 'index.html';
}

export function renderUserPill() {
  const user = getUser();
  if (!user) return;
  const nameEl   = document.getElementById('sidebar-username');
  const avatarEl = document.getElementById('sidebar-avatar');
  if (nameEl)   nameEl.textContent   = user.username;
  if (avatarEl) avatarEl.textContent = user.username.charAt(0).toUpperCase();
  document.getElementById('btn-logout')?.addEventListener('click', logout);
}