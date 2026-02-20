package studio

import (
	"fmt"
	"strings"
)

// GetFrontendHTML returns the complete frontend HTML as a single-page React application.
func GetFrontendHTML(cfg Config) string {
	prefix := strings.TrimSuffix(cfg.Prefix, "/")
	readOnly := "false"
	if cfg.ReadOnly {
		readOnly = "true"
	}
	disableSQL := "false"
	if cfg.DisableSQL {
		disableSQL = "true"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GORM Studio</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/react/18.2.0/umd/react.production.min.js" integrity="sha384-tMH8h3BGESGckSAVGZ82T9n90ztNXxvdwvdM6UoR56cYcf+0iGXBliJ29D+wZ/x8" crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/18.2.0/umd/react-dom.production.min.js" integrity="sha384-bm7MnzvK++ykSwVJ2tynSE5TRdN+xL418osEVF2DE/L/gfWHj91J2Sphe582B1Bh" crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/babel-standalone/7.23.9/babel.min.js" integrity="sha384-ku9eM40vVDsFUiERorrdlHlF0LIhdfn716M7TntM72Uo98T7LWiogD3hNenPx8Q0" crossorigin="anonymous"></script>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=DM+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg-primary: #0a0a0f;
  --bg-secondary: #111118;
  --bg-tertiary: #1a1a24;
  --bg-elevated: #22222e;
  --bg-hover: #2a2a38;
  --bg-active: #32324a;
  --border: #2a2a3a;
  --border-light: #363648;
  --text-primary: #e8e8f0;
  --text-secondary: #9090a8;
  --text-muted: #606078;
  --accent: #6c5ce7;
  --accent-hover: #7c6cf7;
  --accent-muted: rgba(108, 92, 231, 0.15);
  --success: #00b894;
  --danger: #ff6b6b;
  --danger-hover: #ff5252;
  --warning: #fdcb6e;
  --info: #74b9ff;
  --font-sans: 'DM Sans', -apple-system, sans-serif;
  --font-mono: 'JetBrains Mono', monospace;
  --radius: 8px;
  --radius-sm: 5px;
  --radius-lg: 12px;
  --shadow: 0 4px 24px rgba(0,0,0,0.4);
  --shadow-sm: 0 2px 8px rgba(0,0,0,0.3);
  --transition: 150ms ease;
}

[data-theme="light"] {
  --bg-primary: #f5f5f8;
  --bg-secondary: #ffffff;
  --bg-tertiary: #eeeef2;
  --bg-elevated: #e8e8ec;
  --bg-hover: #dddde4;
  --bg-active: #d0d0dc;
  --border: #d0d0d8;
  --border-light: #c0c0cc;
  --text-primary: #1a1a2e;
  --text-secondary: #555568;
  --text-muted: #888898;
  --accent: #6c5ce7;
  --accent-hover: #5a4bd6;
  --accent-muted: rgba(108, 92, 231, 0.12);
  --shadow: 0 4px 24px rgba(0,0,0,0.08);
  --shadow-sm: 0 2px 8px rgba(0,0,0,0.06);
}

html, body, #root { height: 100%%; font-family: var(--font-sans); background: var(--bg-primary); color: var(--text-primary); }

/* Scrollbar */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--border-light); }

/* Layout */
.app { display: flex; height: 100%%; overflow: hidden; }
.sidebar { width: 260px; min-width: 260px; background: var(--bg-secondary); border-right: 1px solid var(--border); display: flex; flex-direction: column; }
.main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }

/* Sidebar */
.sidebar-header { padding: 20px; border-bottom: 1px solid var(--border); }
.sidebar-logo { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; }
.sidebar-logo svg { width: 28px; height: 28px; }
.sidebar-logo span { font-weight: 700; font-size: 16px; letter-spacing: -0.3px; }
.sidebar-logo .accent { color: var(--accent); }
.sidebar-search { width: 100%%; padding: 8px 12px; background: var(--bg-tertiary); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-primary); font-family: var(--font-sans); font-size: 13px; outline: none; transition: border var(--transition); }
.sidebar-search:focus { border-color: var(--accent); }
.sidebar-search::placeholder { color: var(--text-muted); }

.table-list { flex: 1; overflow-y: auto; padding: 8px; }
.table-item { display: flex; align-items: center; justify-content: space-between; padding: 10px 12px; border-radius: var(--radius-sm); cursor: pointer; transition: all var(--transition); font-size: 13px; margin-bottom: 2px; }
.table-item:hover { background: var(--bg-hover); }
.table-item.active { background: var(--accent-muted); color: var(--accent-hover); }
.table-item .name { display: flex; align-items: center; gap: 8px; font-weight: 500; }
.table-item .name svg { width: 14px; height: 14px; opacity: 0.6; }
.table-item .count { font-size: 11px; color: var(--text-muted); background: var(--bg-tertiary); padding: 2px 7px; border-radius: 10px; font-family: var(--font-mono); }
.table-item.active .count { background: rgba(108,92,231,0.25); color: var(--accent-hover); }

.sidebar-footer { padding: 12px 16px; border-top: 1px solid var(--border); display: flex; gap: 6px; }
.sidebar-btn { flex: 1; padding: 7px; font-size: 12px; font-family: var(--font-sans); background: var(--bg-tertiary); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-secondary); cursor: pointer; transition: all var(--transition); text-align: center; }
.sidebar-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.sidebar-btn.active { background: var(--accent-muted); color: var(--accent); border-color: var(--accent); }

/* Main Header */
.main-header { display: flex; align-items: center; justify-content: space-between; padding: 16px 24px; border-bottom: 1px solid var(--border); background: var(--bg-secondary); }
.main-title { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.main-title h2 { font-size: 18px; font-weight: 600; letter-spacing: -0.3px; }
.main-title .badge { font-size: 11px; padding: 3px 8px; background: var(--accent-muted); color: var(--accent); border-radius: 10px; font-family: var(--font-mono); }
.header-actions { display: flex; gap: 8px; align-items: center; }

/* Breadcrumbs */
.breadcrumbs { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--text-muted); }
.breadcrumbs span { cursor: pointer; color: var(--info); }
.breadcrumbs span:hover { text-decoration: underline; }
.breadcrumbs .sep { color: var(--text-muted); cursor: default; }
.breadcrumbs .current { color: var(--text-primary); cursor: default; font-weight: 600; }

/* Buttons */
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 7px 14px; font-size: 13px; font-family: var(--font-sans); font-weight: 500; border: 1px solid var(--border); border-radius: var(--radius-sm); cursor: pointer; transition: all var(--transition); }
.btn svg { width: 14px; height: 14px; }
.btn-default { background: var(--bg-tertiary); color: var(--text-primary); }
.btn-default:hover { background: var(--bg-hover); border-color: var(--border-light); }
.btn-primary { background: var(--accent); color: white; border-color: var(--accent); }
.btn-primary:hover { background: var(--accent-hover); }
.btn-danger { background: transparent; color: var(--danger); border-color: var(--danger); }
.btn-danger:hover { background: rgba(255,107,107,0.1); }
.btn-sm { padding: 4px 8px; font-size: 12px; }
.btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* Data Table */
.table-container { flex: 1; overflow: auto; position: relative; }
.data-table { width: 100%%; border-collapse: collapse; font-size: 13px; }
.data-table th { position: sticky; top: 0; background: var(--bg-tertiary); padding: 10px 16px; text-align: left; font-weight: 600; font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; color: var(--text-secondary); border-bottom: 1px solid var(--border); white-space: nowrap; cursor: pointer; user-select: none; z-index: 2; position: relative; }
.data-table th:hover { color: var(--text-primary); }
.data-table th.sorted { color: var(--accent); }
.data-table th .sort-icon { display: inline-block; margin-left: 4px; font-size: 10px; }
.data-table td { padding: 8px 16px; border-bottom: 1px solid var(--border); white-space: nowrap; max-width: 300px; overflow: hidden; text-overflow: ellipsis; }
.data-table tr { transition: background var(--transition); }
.data-table tbody tr:hover { background: var(--bg-hover); }
.data-table tbody tr.selected { background: var(--accent-muted); }
.data-table .cell-null { color: var(--text-muted); font-style: italic; }
.data-table .cell-pk { color: var(--accent); font-family: var(--font-mono); font-size: 12px; }
.data-table .cell-fk { color: var(--info); cursor: pointer; text-decoration: underline; text-decoration-style: dotted; }
.data-table .cell-fk:hover { color: #a0d0ff; }
.data-table .cell-bool { display: inline-block; width: 8px; height: 8px; border-radius: 50%%; }
.cell-bool.true { background: var(--success); }
.cell-bool.false { background: var(--danger); }
.data-table .row-checkbox { width: 16px; height: 16px; accent-color: var(--accent); cursor: pointer; }
.data-table .actions-cell { display: flex; gap: 4px; }
.data-table .cell-long { cursor: pointer; color: var(--text-secondary); }
.data-table .cell-long:hover { color: var(--info); }

/* Inline editing */
.inline-edit { background: var(--bg-primary); border: 1px solid var(--accent); border-radius: 3px; padding: 2px 6px; color: var(--text-primary); font-family: var(--font-sans); font-size: 13px; outline: none; width: 100%%; min-width: 80px; }

/* Column resize handle */
.col-resize { position: absolute; right: 0; top: 0; bottom: 0; width: 4px; cursor: col-resize; z-index: 3; }
.col-resize:hover, .col-resize.active { background: var(--accent); }

/* Sticky checkbox column */
.data-table th.sticky-col, .data-table td.sticky-col { position: sticky; left: 0; z-index: 3; background: var(--bg-tertiary); }
.data-table td.sticky-col { background: var(--bg-primary); }
.data-table tbody tr:hover td.sticky-col { background: var(--bg-hover); }
.data-table tbody tr.selected td.sticky-col { background: var(--accent-muted); }

/* Pagination */
.pagination { display: flex; align-items: center; justify-content: space-between; padding: 12px 24px; border-top: 1px solid var(--border); background: var(--bg-secondary); font-size: 13px; }
.pagination-info { color: var(--text-secondary); }
.pagination-controls { display: flex; align-items: center; gap: 4px; }
.page-btn { padding: 5px 10px; background: var(--bg-tertiary); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-primary); cursor: pointer; font-size: 12px; transition: all var(--transition); font-family: var(--font-sans); }
.page-btn:hover:not(:disabled) { background: var(--bg-hover); }
.page-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.page-btn.active { background: var(--accent); border-color: var(--accent); color: white; }

/* Filter Bar */
.filter-bar { display: flex; align-items: center; gap: 8px; padding: 10px 24px; background: var(--bg-secondary); border-bottom: 1px solid var(--border); flex-wrap: wrap; }
.filter-input { padding: 6px 10px; background: var(--bg-tertiary); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-primary); font-family: var(--font-sans); font-size: 12px; outline: none; }
.filter-input:focus { border-color: var(--accent); }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 100; animation: fadeIn 0.15s ease; }
.modal { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius-lg); width: 560px; max-width: 90vw; max-height: 80vh; display: flex; flex-direction: column; box-shadow: var(--shadow); animation: slideUp 0.2s ease; }
.modal.modal-wide { width: 800px; }
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 18px 24px; border-bottom: 1px solid var(--border); }
.modal-header h3 { font-size: 16px; font-weight: 600; }
.modal-close { background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 20px; padding: 4px; line-height: 1; }
.modal-close:hover { color: var(--text-primary); }
.modal-body { padding: 24px; overflow-y: auto; flex: 1; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid var(--border); }

/* JSON Viewer */
.json-viewer { background: var(--bg-primary); border: 1px solid var(--border); border-radius: var(--radius); padding: 16px; font-family: var(--font-mono); font-size: 13px; line-height: 1.6; white-space: pre-wrap; word-break: break-all; max-height: 60vh; overflow: auto; color: var(--text-primary); }

/* Form */
.form-group { margin-bottom: 16px; }
.form-label { display: block; font-size: 12px; font-weight: 600; color: var(--text-secondary); margin-bottom: 6px; text-transform: uppercase; letter-spacing: 0.4px; }
.form-input { width: 100%%; padding: 8px 12px; background: var(--bg-tertiary); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-primary); font-family: var(--font-sans); font-size: 13px; outline: none; transition: border var(--transition); }
.form-input:focus { border-color: var(--accent); }
.form-hint { font-size: 11px; color: var(--text-muted); margin-top: 4px; }

/* SQL Editor */
.sql-panel { display: flex; flex-direction: column; height: 100%%; }
.sql-editor-area { flex: 0 0 auto; padding: 16px 24px; border-bottom: 1px solid var(--border); background: var(--bg-secondary); }
.sql-textarea { width: 100%%; min-height: 120px; padding: 12px; background: var(--bg-primary); border: 1px solid var(--border); border-radius: var(--radius); color: var(--text-primary); font-family: var(--font-mono); font-size: 13px; line-height: 1.6; resize: vertical; outline: none; }
.sql-textarea:focus { border-color: var(--accent); }
.sql-actions { display: flex; align-items: center; justify-content: space-between; margin-top: 10px; }
.sql-status { font-size: 12px; color: var(--text-muted); }
.sql-status.error { color: var(--danger); }
.sql-status.success { color: var(--success); }
.sql-results { flex: 1; overflow: auto; }

/* Relation Panel */
.relation-chips { display: flex; flex-wrap: wrap; gap: 6px; padding: 10px 24px; background: var(--bg-secondary); border-bottom: 1px solid var(--border); }
.rel-chip { display: inline-flex; align-items: center; gap: 5px; padding: 5px 12px; background: var(--bg-tertiary); border: 1px solid var(--border); border-radius: 20px; font-size: 12px; cursor: pointer; transition: all var(--transition); }
.rel-chip:hover { border-color: var(--accent); color: var(--accent); }
.rel-chip .rel-type { font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); }

/* Column visibility dropdown */
.col-vis-dropdown { position: relative; }
.col-vis-menu { position: absolute; top: 100%%; right: 0; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius); padding: 8px 0; z-index: 50; min-width: 180px; max-height: 300px; overflow-y: auto; box-shadow: var(--shadow); }
.col-vis-item { display: flex; align-items: center; gap: 8px; padding: 6px 12px; font-size: 12px; cursor: pointer; transition: background var(--transition); }
.col-vis-item:hover { background: var(--bg-hover); }
.col-vis-item input { accent-color: var(--accent); }

/* Toast */
.toast { position: fixed; bottom: 24px; right: 24px; padding: 12px 20px; border-radius: var(--radius); font-size: 13px; z-index: 200; animation: slideUp 0.2s ease; box-shadow: var(--shadow); }
.toast-success { background: #1a3a2a; border: 1px solid var(--success); color: var(--success); }
.toast-error { background: #3a1a1a; border: 1px solid var(--danger); color: var(--danger); }
[data-theme="light"] .toast-success { background: #e8f8f0; }
[data-theme="light"] .toast-error { background: #fde8e8; }

/* Animations */
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes slideUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* Empty State */
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%%; color: var(--text-muted); gap: 12px; }
.empty-state svg { width: 48px; height: 48px; opacity: 0.3; }
.empty-state p { font-size: 14px; }

/* Loading */
.spinner { width: 20px; height: 20px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%%; animation: spin 0.6s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.loading-center { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%%; gap: 16px; color: var(--text-muted); font-size: 13px; }

/* Theme toggle */
.theme-toggle { width: 32px; height: 32px; border-radius: 50%%; background: var(--bg-tertiary); border: 1px solid var(--border); color: var(--text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all var(--transition); padding: 0; }
.theme-toggle:hover { background: var(--bg-hover); color: var(--text-primary); }
.theme-toggle svg { width: 16px; height: 16px; }

/* Export button */
.export-dropdown { position: relative; }
.export-menu { position: absolute; top: 100%%; right: 0; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius); padding: 4px 0; z-index: 50; min-width: 140px; box-shadow: var(--shadow); }
.export-menu-item { padding: 6px 12px; font-size: 12px; cursor: pointer; transition: background var(--transition); display: block; width: 100%%; text-align: left; border: none; background: none; color: var(--text-primary); font-family: var(--font-sans); }
.export-menu-item:hover { background: var(--bg-hover); }
</style>
</head>
<body>
<div id="root">
  <div class="loading-center">
    <div class="spinner" style="width:32px;height:32px;border-width:3px"></div>
    <span>Loading GORM Studio...</span>
  </div>
</div>
<script>
window.__STUDIO_CONFIG__ = {
  prefix: '%s',
  readOnly: %s,
  disableSQL: %s
};
</script>
<script type="text/babel">
const { useState, useEffect, useCallback, useRef, useMemo } = React;

const API = window.__STUDIO_CONFIG__.prefix + '/api';
const CONFIG = window.__STUDIO_CONFIG__;

// ─── API Helper ─────────────────────────────────────────────
async function api(path, opts = {}) {
  const res = await fetch(API + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Request failed');
  return data;
}

// ─── Icons ──────────────────────────────────────────────────
const Icons = {
  Table: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/></svg>,
  Plus: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>,
  Trash: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>,
  Edit: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>,
  Search: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>,
  Terminal: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>,
  Refresh: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>,
  Link: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>,
  Play: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>,
  Key: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>,
  Database: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>,
  X: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>,
  Sun: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>,
  Moon: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>,
  Download: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>,
  Columns: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/></svg>,
  Eye: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>,
};

// ─── Toast ──────────────────────────────────────────────────
function Toast({ toast, onClose }) {
  useEffect(() => {
    if (toast) {
      const t = setTimeout(onClose, 3000);
      return () => clearTimeout(t);
    }
  }, [toast]);
  if (!toast) return null;
  return <div className={'toast toast-' + toast.type}>{toast.message}</div>;
}

// ─── Modal ──────────────────────────────────────────────────
function Modal({ title, onClose, children, footer, wide }) {
  useEffect(() => {
    const handleKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [onClose]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className={'modal' + (wide ? ' modal-wide' : '')} onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>{title}</h3>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}

// ─── JSON Viewer Modal ──────────────────────────────────────
function JsonViewerModal({ value, columnName, onClose }) {
  let formatted = value;
  try { formatted = JSON.stringify(JSON.parse(value), null, 2); } catch(e) {}
  return (
    <Modal title={'View: ' + columnName} onClose={onClose} wide>
      <div className="json-viewer">{formatted}</div>
    </Modal>
  );
}

// ─── Record Form ────────────────────────────────────────────
function RecordForm({ columns, data, onChange, onSubmit }) {
  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && onSubmit) {
      e.preventDefault();
      onSubmit();
    }
  };
  return (
    <div onKeyDown={handleKeyDown}>
      {columns.filter(c => !c.is_primary_key || data[c.name] !== undefined).map(col => (
        <div className="form-group" key={col.name}>
          <label className="form-label">{col.name}{col.is_primary_key ? ' (PK)' : ''}{col.is_foreign_key ? ' (FK)' : ''}</label>
          <input
            className="form-input"
            value={data[col.name] ?? ''}
            onChange={e => onChange({ ...data, [col.name]: e.target.value === '' ? null : e.target.value })}
            disabled={col.is_primary_key}
            placeholder={col.is_nullable ? 'NULL' : 'Required'}
          />
          <div className="form-hint">{col.type}{col.go_type ? ' · ' + col.go_type : ''}{col.is_nullable ? ' · nullable' : ''}</div>
        </div>
      ))}
    </div>
  );
}

// ─── Column Visibility Dropdown ─────────────────────────────
function ColumnVisibility({ columns, hiddenCols, setHiddenCols }) {
  const [open, setOpen] = useState(false);
  const ref = useRef(null);

  useEffect(() => {
    const handleClick = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  return (
    <div className="col-vis-dropdown" ref={ref}>
      <button className="btn btn-default btn-sm" onClick={() => setOpen(!open)} title="Column visibility">
        <Icons.Columns />
      </button>
      {open && (
        <div className="col-vis-menu">
          {columns.map(col => (
            <label className="col-vis-item" key={col.name}>
              <input
                type="checkbox"
                checked={!hiddenCols.has(col.name)}
                onChange={() => {
                  const next = new Set(hiddenCols);
                  next.has(col.name) ? next.delete(col.name) : next.add(col.name);
                  setHiddenCols(next);
                }}
              />
              {col.name}
            </label>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Export Dropdown ─────────────────────────────────────────
function ExportButton({ table }) {
  const [open, setOpen] = useState(false);
  const ref = useRef(null);

  useEffect(() => {
    const handleClick = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const doExport = (format) => {
    window.open(API + '/tables/' + encodeURIComponent(table) + '/export?format=' + format, '_blank');
    setOpen(false);
  };

  return (
    <div className="export-dropdown" ref={ref}>
      <button className="btn btn-default btn-sm" onClick={() => setOpen(!open)} title="Export">
        <Icons.Download />
      </button>
      {open && (
        <div className="export-menu">
          <button className="export-menu-item" onClick={() => doExport('json')}>Export JSON</button>
          <button className="export-menu-item" onClick={() => doExport('csv')}>Export CSV</button>
        </div>
      )}
    </div>
  );
}

// ─── Data Table Component ───────────────────────────────────
function DataTable({ table, schema, onNavigate, showToast, breadcrumbs }) {
  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);
  const [pages, setPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState('');
  const [sortOrder, setSortOrder] = useState('asc');
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState(new Set());
  const [editModal, setEditModal] = useState(null);
  const [formData, setFormData] = useState({});
  const [hiddenCols, setHiddenCols] = useState(new Set());
  const [colWidths, setColWidths] = useState({});
  const [inlineEdit, setInlineEdit] = useState(null); // {row, col, value}
  const [viewerModal, setViewerModal] = useState(null); // {value, col}
  const [showDeleted, setShowDeleted] = useState(false);
  const [hasSoftDelete, setHasSoftDelete] = useState(false);

  const tableInfo = schema.tables.find(t => t.name === table);
  const allColumns = tableInfo?.columns || [];
  const columns = allColumns.filter(c => !hiddenCols.has(c.name));
  const relations = tableInfo?.relations || [];
  const pk = tableInfo?.primary_keys?.[0] || allColumns.find(c => c.is_primary_key)?.name || 'id';

  const fetchRows = useCallback(async () => {
    setLoading(true);
    try {
      let url = '/tables/' + encodeURIComponent(table) + '/rows?page=' + page + '&page_size=' + pageSize;
      if (sortBy) url += '&sort_by=' + sortBy + '&sort_order=' + sortOrder;
      if (search) url += '&search=' + encodeURIComponent(search);
      if (showDeleted) url += '&show_deleted=true';
      const data = await api(url);
      setRows(data.rows || []);
      setTotal(data.total);
      setPages(data.pages);
      if (data.soft_delete !== undefined) setHasSoftDelete(data.soft_delete);
    } catch (err) {
      showToast('error', err.message);
    }
    setLoading(false);
  }, [table, page, pageSize, sortBy, sortOrder, search, showDeleted]);

  useEffect(() => { fetchRows(); setSelected(new Set()); }, [fetchRows]);
  useEffect(() => { setPage(1); setSortBy(''); setSearch(''); setHiddenCols(new Set()); setShowDeleted(false); }, [table]);

  const handleSort = (col) => {
    if (sortBy === col) setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    else { setSortBy(col); setSortOrder('asc'); }
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this record?')) return;
    try {
      await api('/tables/' + encodeURIComponent(table) + '/rows/' + id, { method: 'DELETE' });
      showToast('success', 'Record deleted');
      fetchRows();
    } catch (err) { showToast('error', err.message); }
  };

  const handleBulkDelete = async () => {
    if (!confirm('Delete ' + selected.size + ' records?')) return;
    try {
      await api('/tables/' + encodeURIComponent(table) + '/rows/bulk-delete', { method: 'POST', body: { ids: Array.from(selected) } });
      showToast('success', selected.size + ' records deleted');
      setSelected(new Set());
      fetchRows();
    } catch (err) { showToast('error', err.message); }
  };

  const handleSave = async () => {
    try {
      if (editModal === 'create') {
        await api('/tables/' + encodeURIComponent(table) + '/rows', { method: 'POST', body: formData });
        showToast('success', 'Record created');
      } else {
        const id = editModal[pk];
        await api('/tables/' + encodeURIComponent(table) + '/rows/' + id, { method: 'PUT', body: formData });
        showToast('success', 'Record updated');
      }
      setEditModal(null);
      fetchRows();
    } catch (err) { showToast('error', err.message); }
  };

  // Inline edit save
  const saveInlineEdit = async () => {
    if (!inlineEdit) return;
    const { row, col, value } = inlineEdit;
    try {
      await api('/tables/' + encodeURIComponent(table) + '/rows/' + row[pk], {
        method: 'PUT',
        body: { [col]: value === '' ? null : value }
      });
      showToast('success', 'Cell updated');
      setInlineEdit(null);
      fetchRows();
    } catch (err) { showToast('error', err.message); }
  };

  const openCreate = () => {
    const defaults = {};
    allColumns.forEach(c => { if (!c.is_primary_key) defaults[c.name] = null; });
    setFormData(defaults);
    setEditModal('create');
  };

  const openEdit = (row) => { setFormData({ ...row }); setEditModal(row); };

  const toggleSelect = (id) => { const next = new Set(selected); next.has(id) ? next.delete(id) : next.add(id); setSelected(next); };
  const toggleSelectAll = () => { selected.size === rows.length ? setSelected(new Set()) : setSelected(new Set(rows.map(r => r[pk]))); };

  const isLongValue = (val) => typeof val === 'string' && val.length > 100;

  const formatCell = (value, col, row) => {
    if (value === null || value === undefined) return <span className="cell-null">NULL</span>;
    if (col.is_primary_key) return <span className="cell-pk">{String(value)}</span>;
    if (col.is_foreign_key) {
      return (
        <span className="cell-fk" onClick={() => onNavigate(col.foreign_table, col.foreign_key, value, table)} title={'Go to ' + col.foreign_table}>
          {String(value)} →
        </span>
      );
    }
    if (typeof value === 'boolean') return <span className={'cell-bool ' + value}></span>;
    const str = String(value);
    if (str.length > 80) {
      return <span className="cell-long" onClick={() => setViewerModal({ value: str, col: col.name })} title="Click to view full content">{str.substring(0, 80)}...</span>;
    }
    return str;
  };

  const searchTimer = useRef(null);
  const handleSearch = (val) => {
    clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => { setSearch(val); setPage(1); }, 300);
  };

  // Column resize
  const resizeRef = useRef(null);
  const handleResizeStart = (e, colName) => {
    e.stopPropagation();
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = colWidths[colName] || 150;
    const onMove = (me) => {
      const newWidth = Math.max(60, startWidth + me.clientX - startX);
      setColWidths(prev => ({ ...prev, [colName]: newWidth }));
    };
    const onUp = () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  };

  return (
    <>
      {/* Breadcrumbs */}
      {breadcrumbs && breadcrumbs.length > 1 && (
        <div className="breadcrumbs" style={{padding:'8px 24px',background:'var(--bg-secondary)',borderBottom:'1px solid var(--border)'}}>
          {breadcrumbs.map((b, i) => (
            <React.Fragment key={i}>
              {i > 0 && <span className="sep">/</span>}
              {i < breadcrumbs.length - 1 ? (
                <span onClick={() => onNavigate(b.table, null, null, null, i)}>{b.table}</span>
              ) : (
                <span className="current">{b.table}</span>
              )}
            </React.Fragment>
          ))}
        </div>
      )}

      {/* Relations */}
      {relations.length > 0 && (
        <div className="relation-chips">
          {relations.map(rel => (
            <span key={rel.name} className="rel-chip" title={rel.type + ' → ' + rel.table}>
              <Icons.Link />
              {rel.name}
              <span className="rel-type">{rel.type}</span>
            </span>
          ))}
        </div>
      )}

      {/* Filter Bar */}
      <div className="filter-bar">
        <div style={{position:'relative',display:'flex',alignItems:'center'}}>
          <input className="filter-input" placeholder="Search all columns..." onChange={e => handleSearch(e.target.value)} style={{paddingLeft:30,width:260}} />
          <span style={{position:'absolute',left:8,color:'var(--text-muted)',width:14,height:14}}><Icons.Search /></span>
        </div>
        {hasSoftDelete && (
          <label style={{display:'flex',alignItems:'center',gap:4,fontSize:12,color:'var(--text-secondary)',cursor:'pointer'}}>
            <input type="checkbox" checked={showDeleted} onChange={e => setShowDeleted(e.target.checked)} style={{accentColor:'var(--accent)'}} />
            Show deleted
          </label>
        )}
        <div style={{flex:1}} />
        <ColumnVisibility columns={allColumns} hiddenCols={hiddenCols} setHiddenCols={setHiddenCols} />
        <ExportButton table={table} />
        {selected.size > 0 && !CONFIG.readOnly && (
          <button className="btn btn-danger btn-sm" onClick={handleBulkDelete}><Icons.Trash /> Delete {selected.size}</button>
        )}
        {!CONFIG.readOnly && (
          <button className="btn btn-primary btn-sm" onClick={openCreate}><Icons.Plus /> Add Record</button>
        )}
        <button className="btn btn-default btn-sm" onClick={fetchRows}><Icons.Refresh /></button>
      </div>

      {/* Table */}
      <div className="table-container">
        {loading ? (
          <div className="loading-center"><div className="spinner"></div></div>
        ) : rows.length === 0 ? (
          <div className="empty-state"><Icons.Database /><p>No records found</p></div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                {!CONFIG.readOnly && (
                  <th className="sticky-col" style={{width:36}}>
                    <input type="checkbox" className="row-checkbox" checked={selected.size === rows.length && rows.length > 0} onChange={toggleSelectAll} />
                  </th>
                )}
                {columns.map(col => (
                  <th key={col.name} className={sortBy === col.name ? 'sorted' : ''} onClick={() => handleSort(col.name)} style={colWidths[col.name] ? {width: colWidths[col.name], minWidth: colWidths[col.name]} : {}}>
                    <span style={{display:'flex',alignItems:'center',gap:4}}>
                      {col.is_primary_key && <span style={{width:12,height:12,color:'var(--warning)'}}><Icons.Key /></span>}
                      {col.is_foreign_key && <span style={{width:12,height:12,color:'var(--info)'}}><Icons.Link /></span>}
                      {col.name}
                      {sortBy === col.name && <span className="sort-icon">{sortOrder === 'asc' ? '↑' : '↓'}</span>}
                    </span>
                    <div className="col-resize" onMouseDown={e => handleResizeStart(e, col.name)} />
                  </th>
                ))}
                {!CONFIG.readOnly && <th style={{width:80}}>Actions</th>}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={row[pk] ?? i} className={selected.has(row[pk]) ? 'selected' : ''}>
                  {!CONFIG.readOnly && (
                    <td className="sticky-col"><input type="checkbox" className="row-checkbox" checked={selected.has(row[pk])} onChange={() => toggleSelect(row[pk])} /></td>
                  )}
                  {columns.map(col => (
                    <td key={col.name}
                      onDoubleClick={() => {
                        if (!CONFIG.readOnly && !col.is_primary_key) {
                          setInlineEdit({ row, col: col.name, value: row[col.name] ?? '' });
                        }
                      }}
                      style={colWidths[col.name] ? {maxWidth: colWidths[col.name]} : {}}
                    >
                      {inlineEdit && inlineEdit.row[pk] === row[pk] && inlineEdit.col === col.name ? (
                        <input
                          className="inline-edit"
                          value={inlineEdit.value ?? ''}
                          onChange={e => setInlineEdit({ ...inlineEdit, value: e.target.value })}
                          onKeyDown={e => {
                            if (e.key === 'Enter') saveInlineEdit();
                            if (e.key === 'Escape') setInlineEdit(null);
                          }}
                          onBlur={saveInlineEdit}
                          autoFocus
                        />
                      ) : formatCell(row[col.name], col, row)}
                    </td>
                  ))}
                  {!CONFIG.readOnly && (
                    <td>
                      <div className="actions-cell">
                        <button className="btn btn-default btn-sm" onClick={() => openEdit(row)} title="Edit"><Icons.Edit /></button>
                        <button className="btn btn-danger btn-sm" onClick={() => handleDelete(row[pk])} title="Delete"><Icons.Trash /></button>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Pagination */}
      <div className="pagination">
        <div className="pagination-info">{total} records · Page {page} of {pages || 1}</div>
        <div className="pagination-controls">
          <button className="page-btn" disabled={page <= 1} onClick={() => setPage(1)}>«</button>
          <button className="page-btn" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>‹</button>
          {Array.from({ length: Math.min(5, pages) }, (_, i) => {
            let p;
            if (pages <= 5) p = i + 1;
            else if (page <= 3) p = i + 1;
            else if (page >= pages - 2) p = pages - 4 + i;
            else p = page - 2 + i;
            return <button key={p} className={'page-btn' + (p === page ? ' active' : '')} onClick={() => setPage(p)}>{p}</button>;
          })}
          <button className="page-btn" disabled={page >= pages} onClick={() => setPage(p => p + 1)}>›</button>
          <button className="page-btn" disabled={page >= pages} onClick={() => setPage(pages)}>»</button>
        </div>
      </div>

      {/* Edit/Create Modal */}
      {editModal && (
        <Modal
          title={editModal === 'create' ? 'Create Record' : 'Edit Record'}
          onClose={() => setEditModal(null)}
          footer={<>
            <button className="btn btn-default" onClick={() => setEditModal(null)}>Cancel</button>
            <button className="btn btn-primary" onClick={handleSave}>{editModal === 'create' ? 'Create' : 'Save'}</button>
          </>}
        >
          <RecordForm columns={allColumns} data={formData} onChange={setFormData} onSubmit={handleSave} />
        </Modal>
      )}

      {/* JSON/Text Viewer Modal */}
      {viewerModal && (
        <JsonViewerModal value={viewerModal.value} columnName={viewerModal.col} onClose={() => setViewerModal(null)} />
      )}
    </>
  );
}

// ─── SQL Editor Component ───────────────────────────────────
function SQLEditor({ showToast, schema }) {
  const [query, setQuery] = useState('SELECT * FROM ');
  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState(null);
  const [history, setHistory] = useState([]);
  const [savedQueries, setSavedQueries] = useState(() => {
    try { return JSON.parse(localStorage.getItem('gorm_studio_saved_queries') || '[]'); } catch { return []; }
  });
  const [showSaved, setShowSaved] = useState(false);
  const textareaRef = useRef(null);

  const execute = async () => {
    if (!query.trim()) return;
    setLoading(true);
    setStatus(null);
    try {
      const data = await api('/sql', { method: 'POST', body: { query: query.trim() } });
      setResults(data);
      setStatus({ type: 'success', text: data.type === 'read' ? (data.total + ' rows returned') : (data.rows_affected + ' rows affected') });
      setHistory(h => [{ query: query.trim(), time: new Date().toLocaleTimeString() }, ...h.slice(0, 19)]);
    } catch (err) {
      setStatus({ type: 'error', text: err.message });
      setResults(null);
    }
    setLoading(false);
  };

  const saveQuery = () => {
    const name = prompt('Save query as:');
    if (!name) return;
    const next = [...savedQueries, { name, query: query.trim() }];
    setSavedQueries(next);
    localStorage.setItem('gorm_studio_saved_queries', JSON.stringify(next));
    showToast('success', 'Query saved');
  };

  const deleteSavedQuery = (idx) => {
    const next = savedQueries.filter((_, i) => i !== idx);
    setSavedQueries(next);
    localStorage.setItem('gorm_studio_saved_queries', JSON.stringify(next));
  };

  const handleKeyDown = (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); execute(); }
    // Tab for basic autocomplete
    if (e.key === 'Tab' && schema?.tables) {
      e.preventDefault();
      const el = textareaRef.current;
      const pos = el.selectionStart;
      const text = query.substring(0, pos);
      const word = text.split(/\s/).pop().toLowerCase();
      if (word.length > 0) {
        const tables = schema.tables.map(t => t.name);
        const match = tables.find(t => t.toLowerCase().startsWith(word));
        if (match) {
          const newQuery = query.substring(0, pos - word.length) + match + query.substring(pos);
          setQuery(newQuery);
        }
      }
    }
  };

  return (
    <div className="sql-panel">
      <div className="sql-editor-area">
        <textarea
          ref={textareaRef}
          className="sql-textarea"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Write your SQL query here..."
          spellCheck={false}
        />
        <div className="sql-actions">
          <div style={{display:'flex',gap:8,alignItems:'center'}}>
            {status && <span className={'sql-status ' + status.type}>{status.text}</span>}
          </div>
          <div style={{display:'flex',gap:8,alignItems:'center'}}>
            <button className="btn btn-default btn-sm" onClick={() => setShowSaved(!showSaved)}>
              {showSaved ? 'History' : 'Saved'}
            </button>
            <button className="btn btn-default btn-sm" onClick={saveQuery}>Save</button>
            <span style={{fontSize:11,color:'var(--text-muted)'}}>Ctrl+Enter / Tab</span>
            <button className="btn btn-primary btn-sm" onClick={execute} disabled={loading}>
              {loading ? <div className="spinner" style={{width:14,height:14}}></div> : <Icons.Play />}
              Run
            </button>
          </div>
        </div>
      </div>

      <div className="sql-results">
        {results?.rows && results.rows.length > 0 ? (
          <table className="data-table">
            <thead>
              <tr>{results.columns?.map(col => <th key={col}>{col}</th>)}</tr>
            </thead>
            <tbody>
              {results.rows.map((row, i) => (
                <tr key={i}>
                  {results.columns?.map(col => (
                    <td key={col}>{row[col] === null ? <span className="cell-null">NULL</span> : String(row[col])}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        ) : results?.type === 'write' ? (
          <div className="empty-state"><p style={{color:'var(--success)'}}>Query executed ({results.rows_affected} rows affected)</p></div>
        ) : (
          <div className="empty-state"><Icons.Terminal /><p>Run a query to see results</p></div>
        )}
      </div>

      {/* History / Saved */}
      {(history.length > 0 || savedQueries.length > 0) && (
        <div style={{borderTop:'1px solid var(--border)',padding:'8px 16px',maxHeight:140,overflowY:'auto',background:'var(--bg-secondary)'}}>
          <div style={{fontSize:11,color:'var(--text-muted)',marginBottom:4,fontWeight:600}}>{showSaved ? 'SAVED QUERIES' : 'HISTORY'}</div>
          {showSaved ? savedQueries.map((s, i) => (
            <div key={i} style={{fontSize:12,padding:'3px 0',display:'flex',gap:8,alignItems:'center'}}>
              <span style={{cursor:'pointer',color:'var(--text-secondary)',fontFamily:'var(--font-mono)',flex:1,overflow:'hidden',textOverflow:'ellipsis',whiteSpace:'nowrap'}} onClick={() => setQuery(s.query)} title={s.query}>
                <span style={{color:'var(--accent)',marginRight:8}}>{s.name}</span>{s.query}
              </span>
              <button style={{background:'none',border:'none',color:'var(--danger)',cursor:'pointer',fontSize:11,padding:2}} onClick={() => deleteSavedQuery(i)}>×</button>
            </div>
          )) : history.map((h, i) => (
            <div key={i} style={{fontSize:12,padding:'3px 0',cursor:'pointer',color:'var(--text-secondary)',fontFamily:'var(--font-mono)',overflow:'hidden',textOverflow:'ellipsis',whiteSpace:'nowrap'}} onClick={() => setQuery(h.query)} title={h.query}>
              <span style={{color:'var(--text-muted)',marginRight:8}}>{h.time}</span>{h.query}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Tools Panel ─────────────────────────────────────────────
function ToolCard({ title, description, children }) {
  return React.createElement('div', {style:{background:'var(--bg-secondary)',border:'1px solid var(--border)',borderRadius:'var(--radius)',padding:16,marginBottom:12}},
    React.createElement('div', {style:{fontWeight:600,fontSize:14,marginBottom:4}}, title),
    React.createElement('div', {style:{fontSize:12,color:'var(--text-muted)',marginBottom:12}}, description),
    children
  );
}

function FileUploader({ endpoint, accept, showToast, onSuccess, extraFields }) {
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);

  const upload = async (file) => {
    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);
    if (extraFields) {
      Object.entries(extraFields).forEach(([k,v]) => { if (v) formData.append(k, v); });
    }
    try {
      const res = await fetch(CONFIG.prefix + '/api' + endpoint, { method: 'POST', body: formData });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Upload failed');
      if (onSuccess) onSuccess(data);
    } catch (err) {
      showToast('error', err.message);
    }
    setUploading(false);
  };

  return React.createElement('div', {
    onDragOver: (e) => { e.preventDefault(); setDragOver(true); },
    onDragLeave: () => setDragOver(false),
    onDrop: (e) => { e.preventDefault(); setDragOver(false); if (e.dataTransfer.files[0]) upload(e.dataTransfer.files[0]); },
    onClick: () => {
      const input = document.createElement('input');
      input.type = 'file'; input.accept = accept;
      input.onchange = (e) => { if (e.target.files[0]) upload(e.target.files[0]); };
      input.click();
    },
    style: {
      border: '2px dashed ' + (dragOver ? 'var(--accent)' : 'var(--border)'),
      borderRadius: 'var(--radius)', padding: 20, textAlign: 'center',
      background: dragOver ? 'rgba(108,92,231,0.08)' : 'transparent',
      cursor: 'pointer', transition: 'all 0.2s'
    }
  },
    uploading
      ? React.createElement('div', {className:'spinner', style:{margin:'0 auto'}})
      : React.createElement(React.Fragment, null,
          React.createElement('div', {style:{fontSize:24,marginBottom:8}}, '\u2191'),
          React.createElement('div', {style:{fontSize:12,color:'var(--text-muted)'}}, 'Drop file here or click to browse'),
          React.createElement('div', {style:{fontSize:11,color:'var(--text-muted)',marginTop:4}}, accept)
        )
  );
}

function ToolsPanel({ schema, showToast, onRefresh }) {
  const [importTable, setImportTable] = useState('');
  const [goCodeModal, setGoCodeModal] = useState(null);
  const API = CONFIG.prefix + '/api';
  const tables = schema?.tables || [];
  const exportFormats = ['sql','json','yaml','dbml','png','pdf'];
  const dataFormats = ['json','csv','sql'];

  return React.createElement('div', {style:{padding:24,overflowY:'auto',height:'calc(100vh - 60px)'}},
    // Export Section
    React.createElement('h3', {style:{marginBottom:16,color:'var(--text-secondary)',fontSize:12,textTransform:'uppercase',letterSpacing:1}}, 'Export'),

    React.createElement(ToolCard, {title:'Schema Export', description:'Export database schema in various formats'},
      React.createElement('div', {style:{display:'flex',gap:8,flexWrap:'wrap'}},
        exportFormats.map(fmt =>
          React.createElement('button', {key:fmt, className:'btn btn-default', style:{fontSize:12,padding:'6px 12px'},
            onClick:() => window.open(API+'/export/schema?format='+fmt, '_blank')
          }, fmt.toUpperCase())
        )
      )
    ),

    React.createElement(ToolCard, {title:'Data Export', description:'Export all database data'},
      React.createElement('div', {style:{display:'flex',gap:8,flexWrap:'wrap'}},
        dataFormats.map(fmt =>
          React.createElement('button', {key:fmt, className:'btn btn-default', style:{fontSize:12,padding:'6px 12px'},
            onClick:() => window.open(API+'/export/data?format='+fmt, '_blank')
          }, fmt.toUpperCase())
        )
      )
    ),

    React.createElement(ToolCard, {title:'Go Models', description:'Generate Go struct code from database schema'},
      React.createElement('button', {className:'btn btn-primary', style:{fontSize:12,padding:'6px 16px'},
        onClick:() => window.open(API+'/export/models', '_blank')
      }, 'Download models.go')
    ),

    // Import Section (hidden in read-only mode)
    !CONFIG.readOnly && React.createElement(React.Fragment, null,
      React.createElement('h3', {style:{marginTop:32,marginBottom:16,color:'var(--text-secondary)',fontSize:12,textTransform:'uppercase',letterSpacing:1}}, 'Import'),

      React.createElement(ToolCard, {title:'Schema Import', description:'Create tables from schema files (.sql, .json, .yaml, .dbml)'},
        React.createElement(FileUploader, {
          endpoint:'/import/schema', accept:'.sql,.json,.yaml,.yml,.dbml', showToast:showToast,
          onSuccess:(data) => {
            showToast('success', 'Created tables: ' + (data.tables_created||[]).join(', '));
            if (data.go_code) setGoCodeModal(data.go_code);
            onRefresh();
          }
        })
      ),

      React.createElement(ToolCard, {title:'Data Import', description:'Import data from files (.json, .csv, .sql, .xlsx)'},
        React.createElement('div', {style:{marginBottom:12}},
          React.createElement('select', {
            style:{width:'100%%',padding:'8px 12px',background:'var(--bg-tertiary)',color:'var(--text-primary)',border:'1px solid var(--border)',borderRadius:'var(--radius)',fontSize:13,marginBottom:8},
            value:importTable, onChange:(e)=>setImportTable(e.target.value)
          },
            React.createElement('option', {value:''}, 'Select table (required for CSV/Excel)'),
            tables.map(t => React.createElement('option', {key:t.name,value:t.name}, t.name))
          )
        ),
        React.createElement(FileUploader, {
          endpoint:'/import/data', accept:'.json,.csv,.sql,.xlsx', showToast:showToast,
          extraFields:{table:importTable},
          onSuccess:(data) => {
            showToast('success', (data.rows_inserted||0) + ' rows imported into ' + (data.tables_affected||[]).join(', '));
            onRefresh();
          }
        })
      ),

      React.createElement(ToolCard, {title:'Go Models Import', description:'Create tables from Go struct definitions (.go file)'},
        React.createElement(FileUploader, {
          endpoint:'/import/models', accept:'.go', showToast:showToast,
          onSuccess:(data) => {
            showToast('success', 'Created tables: ' + (data.tables_created||[]).join(', ') + ' from structs: ' + (data.structs_parsed||[]).join(', '));
            onRefresh();
          }
        })
      )
    ),

    // Go Code Modal
    goCodeModal && React.createElement(Modal, {title:'Generated Go Models', onClose:()=>setGoCodeModal(null), wide:true},
      React.createElement('pre', {style:{background:'var(--bg-primary)',padding:16,borderRadius:'var(--radius)',overflow:'auto',maxHeight:500,fontSize:13,fontFamily:'JetBrains Mono, monospace',whiteSpace:'pre-wrap'}}, goCodeModal),
      React.createElement('div', {style:{marginTop:12,display:'flex',gap:8}},
        React.createElement('button', {className:'btn btn-primary', onClick:() => {
          navigator.clipboard.writeText(goCodeModal);
          showToast('success', 'Copied to clipboard');
        }}, 'Copy to Clipboard'),
        React.createElement('button', {className:'btn btn-default', onClick:()=>setGoCodeModal(null)}, 'Close')
      )
    )
  );
}

// ─── App ────────────────────────────────────────────────────
function App() {
  const [schema, setSchema] = useState(null);
  const [activeTable, setActiveTable] = useState(null);
  const [view, setView] = useState('data');
  const [tableSearch, setTableSearch] = useState('');
  const [toast, setToast] = useState(null);
  const [loading, setLoading] = useState(true);
  const [theme, setTheme] = useState(() => localStorage.getItem('gorm_studio_theme') || 'dark');
  const [breadcrumbs, setBreadcrumbs] = useState([]);

  const showToast = (type, message) => setToast({ type, message });

  // Apply theme
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('gorm_studio_theme', theme);
  }, [theme]);

  useEffect(() => {
    api('/schema').then(data => {
      setSchema(data);
      if (data.tables?.length > 0) {
        setActiveTable(data.tables[0].name);
        setBreadcrumbs([{ table: data.tables[0].name }]);
      }
      setLoading(false);
    }).catch(err => {
      showToast('error', 'Failed to load schema: ' + err.message);
      setLoading(false);
    });
  }, []);

  const refreshSchema = async () => {
    try {
      const data = await api('/schema/refresh', { method: 'POST' });
      setSchema(data);
      showToast('success', 'Schema refreshed');
    } catch (err) { showToast('error', err.message); }
  };

  // Navigate with breadcrumb support
  const handleNavigate = (table, column, value, fromTable, breadcrumbIdx) => {
    if (breadcrumbIdx !== undefined) {
      // Click on breadcrumb — go back
      setBreadcrumbs(prev => prev.slice(0, breadcrumbIdx + 1));
      setActiveTable(table);
      return;
    }
    setActiveTable(table);
    if (fromTable) {
      setBreadcrumbs(prev => [...prev, { table, from: fromTable, column, value }]);
    } else {
      setBreadcrumbs([{ table }]);
    }
  };

  const handleTableClick = (tableName) => {
    setActiveTable(tableName);
    setView('data');
    setBreadcrumbs([{ table: tableName }]);
  };

  const filteredTables = useMemo(() => {
    if (!schema?.tables) return [];
    if (!tableSearch) return schema.tables;
    return schema.tables.filter(t => t.name.toLowerCase().includes(tableSearch.toLowerCase()));
  }, [schema, tableSearch]);

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKey = (e) => {
      if (e.key === 'Escape' && view === 'sql') setView('data');
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [view]);

  if (loading) {
    return <div className="loading-center"><div className="spinner" style={{width:32,height:32,borderWidth:3}}></div><span>Loading schema...</span></div>;
  }

  const activeTableInfo = schema?.tables?.find(t => t.name === activeTable);

  return (
    <div className="app">
      {/* Sidebar */}
      <div className="sidebar">
        <div className="sidebar-header">
          <div className="sidebar-logo">
            <Icons.Database />
            <span>GORM <span className="accent">Studio</span></span>
            <div style={{marginLeft:'auto'}}>
              <button className="theme-toggle" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')} title="Toggle theme">
                {theme === 'dark' ? <Icons.Sun /> : <Icons.Moon />}
              </button>
            </div>
          </div>
          <input className="sidebar-search" placeholder="Search tables..." value={tableSearch} onChange={e => setTableSearch(e.target.value)} />
        </div>

        <div className="table-list">
          {filteredTables.map(table => (
            <div key={table.name} className={'table-item' + (activeTable === table.name ? ' active' : '')} onClick={() => handleTableClick(table.name)}>
              <span className="name"><Icons.Table />{table.name}</span>
              <span className="count">{table.row_count}</span>
            </div>
          ))}
          {filteredTables.length === 0 && (
            <div style={{padding:16,textAlign:'center',color:'var(--text-muted)',fontSize:13}}>No tables found</div>
          )}
        </div>

        <div className="sidebar-footer">
          <button className={'sidebar-btn' + (view === 'data' ? ' active' : '')} onClick={() => setView('data')}>Tables</button>
          {!CONFIG.disableSQL && (
            <button className={'sidebar-btn' + (view === 'sql' ? ' active' : '')} onClick={() => setView('sql')}>SQL</button>
          )}
          <button className={'sidebar-btn' + (view === 'tools' ? ' active' : '')} onClick={() => setView('tools')}>Tools</button>
          <button className="sidebar-btn" onClick={refreshSchema}>↻</button>
        </div>
      </div>

      {/* Main Content */}
      <div className="main">
        {view === 'data' ? (
          activeTable ? (
            <>
              <div className="main-header">
                <div className="main-title">
                  <h2>{activeTable}</h2>
                  <span className="badge">{activeTableInfo?.row_count ?? 0} rows</span>
                  {activeTableInfo?.relations?.length > 0 && (
                    <span className="badge" style={{background:'rgba(116,185,255,0.15)',color:'var(--info)'}}>{activeTableInfo.relations.length} relations</span>
                  )}
                </div>
              </div>
              <DataTable
                key={activeTable}
                table={activeTable}
                schema={schema}
                onNavigate={handleNavigate}
                showToast={showToast}
                breadcrumbs={breadcrumbs}
              />
            </>
          ) : (
            <div className="empty-state"><Icons.Database /><p>Select a table from the sidebar</p></div>
          )
        ) : view === 'tools' ? (
          <>
            <div className="main-header">
              <div className="main-title">
                <h2>Import & Export Tools</h2>
                <span className="badge">Database tools</span>
              </div>
            </div>
            <ToolsPanel schema={schema} showToast={showToast} onRefresh={refreshSchema} />
          </>
        ) : (
          <>
            <div className="main-header">
              <div className="main-title">
                <h2>SQL Editor</h2>
                <span className="badge">Raw queries</span>
              </div>
            </div>
            <SQLEditor showToast={showToast} schema={schema} />
          </>
        )}
      </div>

      <Toast toast={toast} onClose={() => setToast(null)} />
    </div>
  );
}

ReactDOM.render(<App />, document.getElementById('root'));
</script>
</body>
</html>`, prefix, readOnly, disableSQL)
}
