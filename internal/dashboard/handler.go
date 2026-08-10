package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

type Handler struct {
	memory   *memory.Manager
	analyzer *analyzer.ProjectAnalyzer
}

type memoryDTO struct {
	Key          string    `json:"key"`
	Content      string    `json:"content"`
	Tags         []string  `json:"tags"`
	Timestamp    time.Time `json:"timestamp"`
	Usage        int       `json:"usage"`
	Project      string    `json:"project,omitempty"`
	DecisionType string    `json:"decisionType,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Alternatives []string  `json:"alternatives,omitempty"`
	Promoted     bool      `json:"promoted,omitempty"` // Fase 3
}

type memoryStats struct {
	Total         int      `json:"total"`
	DecisionCount int      `json:"decisionCount"`
	TagCount      int      `json:"tagCount"`
	DecisionTypes []string `json:"decisionTypes"`
	PromotedCount int      `json:"promotedCount"` // Fase 3
}

type memoryListResponse struct {
	Items         []memoryDTO          `json:"items"`
	Stats         memoryStats          `json:"stats"`
	Projects      []memory.ProjectStat `json:"projects"`
	ActiveProject string               `json:"activeProject"`
}

func New(memoryManager *memory.Manager, analyzer *analyzer.ProjectAnalyzer) (*Handler, error) {
	if memoryManager == nil {
		return nil, fmt.Errorf("memory manager is required")
	}

	return &Handler{memory: memoryManager, analyzer: analyzer}, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", h.handleDashboard)
	mux.HandleFunc("/api/memories", h.handleMemories)
	mux.HandleFunc("/api/memories/", h.handleMemoryByKey)
	mux.HandleFunc("/api/projects", h.handleProjects)
	mux.HandleFunc("/api/project-summary", h.handleProjectSummary)
	mux.HandleFunc("/api/quick-context", h.handleQuickContext)
	mux.HandleFunc("/api/suggestions", h.handleSuggestions)
}

// handleProjects returns the list of known projects with memory counts.
func (h *Handler) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active":   h.memory.ActiveProject(),
		"projects": h.memory.ListProjects(),
	})
}

// handleSuggestions returns promotion candidates (Fase 3)
func (h *Handler) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 6
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	suggestions, err := h.memory.SuggestForPromotion(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": toMemoryDTOs(suggestions),
		"count": len(suggestions),
	})
}

var pageTemplate = template.Must(template.New("memory-dashboard").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Memory Dashboard</title>
  <style>
    :root {
      --bg: #f3efe6;
      --panel: rgba(255, 250, 242, 0.9);
      --panel-strong: #fffdf8;
      --line: rgba(33, 44, 36, 0.14);
      --text: #1f2a24;
      --muted: #58645d;
      --accent: #1d6b52;
      --accent-soft: #d8efe5;
      --danger: #8a2f2f;
      --shadow: 0 18px 48px rgba(39, 52, 45, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(29, 107, 82, 0.18), transparent 28%),
        radial-gradient(circle at top right, rgba(186, 143, 74, 0.12), transparent 22%),
        linear-gradient(180deg, #faf7f0 0%, var(--bg) 100%);
      min-height: 100vh;
    }
    .shell {
      width: min(1200px, calc(100vw - 32px));
      margin: 0 auto;
      padding: 28px 0 56px;
    }
    .hero {
      display: grid;
      grid-template-columns: 1.4fr 1fr;
      gap: 20px;
      margin-bottom: 20px;
    }
    .hero-card,
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(12px);
    }
    .hero-card {
      padding: 28px;
      position: relative;
      overflow: hidden;
    }
    .hero-card::after {
      content: "";
      position: absolute;
      inset: auto -80px -90px auto;
      width: 220px;
      height: 220px;
      background: radial-gradient(circle, rgba(29, 107, 82, 0.22), transparent 70%);
      pointer-events: none;
    }
    h1 {
      margin: 0 0 10px;
      font-size: clamp(2rem, 4vw, 3.2rem);
      line-height: 0.95;
      letter-spacing: -0.04em;
    }
    .hero-copy {
      max-width: 55ch;
      color: var(--muted);
      margin-bottom: 22px;
      font-size: 1rem;
      line-height: 1.55;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 12px;
    }
    .stat {
      padding: 16px;
      border-radius: 18px;
      background: var(--panel-strong);
      border: 1px solid var(--line);
    }
    .stat-label {
      display: block;
      font-size: 0.78rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--muted);
      margin-bottom: 8px;
    }
    .stat-value {
      font-size: 1.8rem;
      font-weight: 700;
    }
    .panel {
      padding: 22px;
    }
    .filters {
      display: grid;
      grid-template-columns: 2fr 1fr 1fr auto;
      gap: 12px;
      margin-bottom: 16px;
    }
    label {
      display: flex;
      flex-direction: column;
      gap: 6px;
      font-size: 0.82rem;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }
    input, select, button {
      font: inherit;
      border-radius: 14px;
      border: 1px solid var(--line);
      padding: 12px 14px;
    }
    input, select {
      background: rgba(255,255,255,0.85);
      color: var(--text);
    }
    button {
      cursor: pointer;
      background: var(--accent);
      color: white;
      border-color: transparent;
      transition: transform 120ms ease, opacity 120ms ease;
    }
    button:hover { transform: translateY(-1px); }
    .memory-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 14px;
    }
    .memory-card {
      background: var(--panel-strong);
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 18px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .memory-head {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: flex-start;
    }
    .memory-key {
      margin: 0;
      font-size: 1.05rem;
      line-height: 1.15;
      word-break: break-word;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      border-radius: 999px;
      padding: 5px 10px;
      background: var(--accent-soft);
      color: var(--accent);
      font-size: 0.78rem;
      font-weight: 600;
      white-space: nowrap;
    }
    .memory-meta {
      font-size: 0.86rem;
      color: var(--muted);
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
    }
    .memory-content {
      margin: 0;
      white-space: pre-wrap;
      line-height: 1.55;
    }
    .tags {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
    }
    .tag {
      border-radius: 999px;
      background: #efe8da;
      color: #5c5548;
      padding: 4px 10px;
      font-size: 0.78rem;
    }
    .memory-actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      margin-top: auto;
    }
    .memory-actions button {
      background: transparent;
      color: var(--danger);
      border-color: rgba(138, 47, 47, 0.18);
      padding: 8px 12px;
    }
    .empty {
      padding: 28px;
      border: 1px dashed var(--line);
      border-radius: 20px;
      text-align: center;
      color: var(--muted);
      background: rgba(255,255,255,0.45);
    }
    .footer-note {
      margin-top: 18px;
      color: var(--muted);
      font-size: 0.9rem;
    }
    @media (max-width: 860px) {
      .hero { grid-template-columns: 1fr; }
      .filters { grid-template-columns: 1fr; }
      .stats { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="hero-card">
        <div class="badge">Memory dashboard</div>
        <h1>Search and manage saved development memory.</h1>
        <p class="hero-copy">Inspect stored context, filter by decision type, review usage and recency, and remove stale entries directly from the running server.</p>
        <div class="stats">
          <div class="stat">
            <span class="stat-label">Total memories</span>
            <span class="stat-value" id="stat-total">0</span>
          </div>
          <div class="stat">
            <span class="stat-label">Decision entries</span>
            <span class="stat-value" id="stat-decisions">0</span>
          </div>
          <div class="stat">
            <span class="stat-label">Unique tags</span>
            <span class="stat-value" id="stat-tags">0</span>
          </div>
          <div class="stat">
            <span class="stat-label">Promoted</span>
            <span class="stat-value" id="stat-promoted">0</span>
          </div>
        </div>
      </div>
      <div class="panel">
        <div class="badge">Decision types</div>
        <div id="decision-types" class="tags" style="margin-top:16px"></div>
        <p class="footer-note">Use HTTP or SSE transport to access this dashboard. The data stays local to the server process.</p>
      </div>
    </section>

    <section class="panel">
      <div class="filters">
        <label>
          Search
          <input id="search" type="search" placeholder="Search by key, content, reason or alternatives">
        </label>
        <label>
          Project
          <select id="project">
            <option value="">Active project</option>
            <option value="*">All projects</option>
          </select>
        </label>
        <label>
          Decision type
          <select id="decision-type">
            <option value="">All</option>
          </select>
        </label>
        <label>
          Limit
          <select id="limit">
            <option value="25">25</option>
            <option value="50" selected>50</option>
            <option value="100">100</option>
            <option value="0">All</option>
          </select>
        </label>
        <label style="justify-content:flex-end">
          Actions
          <button id="refresh" type="button">Refresh</button>
        </label>
        <label style="align-items:center; gap:6px;">
          <input type="checkbox" id="promoted-only"> 
          <span style="font-size:0.85rem;">Promoted only</span>
        </label>
      </div>
      <div id="active-project-banner" class="footer-note" style="margin-top:8px"></div>

      <!-- Fase 3: Suggestions for promotion -->
      <h3 style="margin-top: 28px; margin-bottom: 4px; color: var(--text);">Suggested for Promotion</h3>
      <p style="margin:0 0 12px 0; font-size:0.8rem; color:var(--muted);">These are the highest-value memories worth keeping across sessions. Promote the ones that matter.</p>
      <div id="suggestions" class="memory-grid" style="margin-bottom: 24px;"></div>

      <div id="memories" class="memory-grid"></div>
    </section>
  </div>

  <script>
    const searchInput = document.getElementById('search');
    const projectSelect = document.getElementById('project');
    const decisionTypeSelect = document.getElementById('decision-type');
    const limitSelect = document.getElementById('limit');
    const refreshButton = document.getElementById('refresh');
    const memoriesContainer = document.getElementById('memories');
    const suggestionsContainer = document.getElementById('suggestions');
    const decisionTypesContainer = document.getElementById('decision-types');
    const activeProjectBanner = document.getElementById('active-project-banner');

    const statTotal = document.getElementById('stat-total');
    const statDecisions = document.getElementById('stat-decisions');
    const statTags = document.getElementById('stat-tags');
    const statPromoted = document.getElementById('stat-promoted');
    const promotedOnlyCheckbox = document.getElementById('promoted-only');

    let refreshTimer;
    let lastMemories = []; // for client-side promoted filter

    async function loadMemories() {
      const params = new URLSearchParams();
      if (searchInput.value.trim()) params.set('query', searchInput.value.trim());
      if (decisionTypeSelect.value) params.set('decisionType', decisionTypeSelect.value);
      if (projectSelect.value !== '') params.set('project', projectSelect.value);
      params.set('limit', limitSelect.value);

      const response = await fetch('/api/memories?' + params.toString());
      if (!response.ok) {
        memoriesContainer.innerHTML = '<div class="empty">Failed to load memories.</div>';
        return;
      }

      const data = await response.json();
      renderStats(data.stats || {});
      renderDecisionTypes((data.stats && data.stats.decisionTypes) || []);
      renderProjects(data.projects || [], data.activeProject || '');
      renderMemories(data.items || []);
    }

    function renderProjects(projects, active) {
      const prev = projectSelect.value;
      const opts = ['<option value="">Active project (' + (active || '—') + ')</option>',
                    '<option value="*">All projects</option>'];
      projects.forEach(p => {
        const label = p.project + ' (' + p.count + ')' + (p.active ? ' • active' : '');
        opts.push('<option value="' + p.project + '">' + label + '</option>');
      });
      projectSelect.innerHTML = opts.join('');
      // Restore previous selection if still valid.
      const valid = Array.from(projectSelect.options).some(o => o.value === prev);
      projectSelect.value = valid ? prev : '';
      if (activeProjectBanner) {
        activeProjectBanner.textContent = 'Active project: ' + (active || '—');
      }
    }

    function renderStats(stats) {
      statTotal.textContent = stats.total || 0;
      statDecisions.textContent = stats.decisionCount || 0;
      statTags.textContent = stats.tagCount || 0;
      if (statPromoted) {
        statPromoted.textContent = stats.promotedCount || 0;
      }

      const selected = decisionTypeSelect.value;
      decisionTypeSelect.innerHTML = '<option value="">All</option>';
      (stats.decisionTypes || []).forEach(type => {
        const option = document.createElement('option');
        option.value = type;
        option.textContent = type;
        if (type === selected) option.selected = true;
        decisionTypeSelect.appendChild(option);
      });
    }

    function renderDecisionTypes(types) {
      decisionTypesContainer.innerHTML = '';
      if (!types.length) {
        decisionTypesContainer.innerHTML = '<span class="tag">No decision types</span>';
        return;
      }
      types.forEach(type => {
        const span = document.createElement('span');
        span.className = 'tag';
        span.textContent = type;
        decisionTypesContainer.appendChild(span);
      });
    }

    function renderMemories(items) {
      lastMemories = items;

      let toRender = items;
      if (promotedOnlyCheckbox && promotedOnlyCheckbox.checked) {
        toRender = items.filter(i => i.promoted);
        if (!toRender.length) {
          memoriesContainer.innerHTML = '<div class="empty">No promoted memories match current filters.</div>';
          return;
        }
      }

      if (!toRender.length) {
        memoriesContainer.innerHTML = '<div class="empty">No memories matched the current filters.</div>';
        return;
      }

      memoriesContainer.innerHTML = toRender.map(item => {
        const timestamp = item.timestamp ? new Date(item.timestamp).toLocaleString() : 'n/a';
        const tags = (item.tags || []).map(tag => '<span class="tag">' + escapeHtml(tag) + '</span>').join('');
        const alternatives = (item.alternatives || []).length
          ? '<div><strong>Alternatives:</strong> ' + item.alternatives.map(escapeHtml).join(', ') + '</div>'
          : '';
        const reason = item.reason ? '<div><strong>Reason:</strong> ' + escapeHtml(item.reason) + '</div>' : '';
        const badge = item.decisionType ? '<span class="badge">' + escapeHtml(item.decisionType) + '</span>' : '';
        const promotedBadge = item.promoted ? '<span class="badge" style="background:#d4edda;color:#155724;">Promoted</span>' : '';

        return '' +
          '<article class="memory-card">' +
            '<div class="memory-head">' +
              '<h2 class="memory-key">' + escapeHtml(item.key) + '</h2>' +
              badge + promotedBadge +
            '</div>' +
            '<div class="memory-meta">' +
              '<span>Saved ' + escapeHtml(timestamp) + '</span>' +
              '<span>Usage ' + (item.usage || 0) + '</span>' +
              '<span>Project ' + escapeHtml(item.project || 'unassigned') + '</span>' +
            '</div>' +
            '<p class="memory-content">' + escapeHtml(item.content || '') + '</p>' +
            reason +
            alternatives +
            '<div class="tags">' + tags + '</div>' +
            '<div class="memory-actions">' +
              '<span class="footer-note">Key: ' + escapeHtml(item.key) + '</span>' +
              '<button type="button" data-key="' + encodeURIComponent(item.key) + '">Delete</button>' +
            '</div>' +
          '</article>';
      }).join('');

      memoriesContainer.querySelectorAll('button[data-key]').forEach(button => {
        button.addEventListener('click', async () => {
          const decodedKey = decodeURIComponent(button.dataset.key);
          const confirmed = window.confirm('Delete memory "' + decodedKey + '"?');
          if (!confirmed) return;

          const response = await fetch('/api/memories/' + button.dataset.key, { method: 'DELETE' });
          if (response.ok) {
            loadMemories();
          } else {
            window.alert('Failed to delete memory.');
          }
        });
      });
    }

    function escapeHtml(value) {
      return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
    }

    function scheduleRefresh() {
      window.clearTimeout(refreshTimer);
      refreshTimer = window.setTimeout(loadMemories, 250);
    }

    searchInput.addEventListener('input', scheduleRefresh);
    decisionTypeSelect.addEventListener('change', loadMemories);
    limitSelect.addEventListener('change', loadMemories);
    projectSelect.addEventListener('change', loadMemories);
    refreshButton.addEventListener('click', loadMemories);

    if (promotedOnlyCheckbox) {
      promotedOnlyCheckbox.addEventListener('change', () => {
        if (lastMemories.length) {
          renderMemories(lastMemories);
        }
      });
    }

    loadMemories();
    loadSuggestions();

    async function loadSuggestions() {
      if (!suggestionsContainer) return;
      try {
        const res = await fetch('/api/suggestions?limit=6');
        if (!res.ok) throw new Error('Failed');
        const data = await res.json();
        renderSuggestions(data.items || []);
      } catch (e) {
        suggestionsContainer.innerHTML = '<div class="empty">Could not load suggestions.</div>';
      }
    }

    function renderSuggestions(items) {
      if (!suggestionsContainer) return;

      var header = '<div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px;">' +
        '<span style="font-weight:600; color:var(--text);">Top candidates for long-term memory</span>' +
        '<button id="refresh-suggestions" style="font-size:0.75rem; padding:4px 10px; background:transparent; color:var(--accent); border:1px solid var(--accent);">Refresh</button>' +
        '</div>';

      if (!items.length) {
        suggestionsContainer.innerHTML = header + '<div class="empty">No strong promotion candidates right now.</div>';
        var refreshBtn = document.getElementById('refresh-suggestions');
        if (refreshBtn) refreshBtn.addEventListener('click', loadSuggestions);
        return;
      }

      var html = header;
      for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var badge = item.decisionType ? '<span class="badge">' + escapeHtml(item.decisionType) + '</span>' : '';
        var usageText = item.usage ? 'Used ' + item.usage + 'x' : '';
        var usage = usageText ? '<span style="font-size:0.75rem;color:var(--muted);">' + usageText + '</span>' : '';
        var reasonText = '';
        if (item.reason) {
          var shortReason = escapeHtml(item.reason.slice(0,120));
          if (item.reason.length > 120) shortReason += '...';
          reasonText = '<div style="font-size:0.8rem; color:#555; margin-top:4px;"><strong>Reason:</strong> ' + shortReason + '</div>';
        }

        html += '<article class="memory-card" style="border-left: 4px solid var(--accent);">' +
          '<div class="memory-head">' +
            '<h2 class="memory-key" style="font-size:1rem;">' + escapeHtml(item.key) + '</h2>' + badge +
          '</div>' +
          '<p class="memory-content" style="font-size:0.85rem; line-height:1.4;">' + escapeHtml((item.content || '').slice(0, 160)) + '...</p>' +
          reasonText +
          '<div style="margin-top:6px;">' + usage + '</div>' +
          '<div class="memory-actions">' +
            '<button type="button" class="promote-btn" data-key="' + encodeURIComponent(item.key) + '" style="background:var(--accent);color:white;border:none;padding:6px 14px;border-radius:6px;font-size:0.8rem;">Promote</button>' +
          '</div>' +
          '</article>';
      }
      suggestionsContainer.innerHTML = html;

      var refreshBtn2 = document.getElementById('refresh-suggestions');
      if (refreshBtn2) refreshBtn2.addEventListener('click', loadSuggestions);

      suggestionsContainer.querySelectorAll('.promote-btn').forEach(function(btn) {
        btn.addEventListener('click', async function() {
          var key = decodeURIComponent(btn.dataset.key);
          if (!confirm('Promote "' + key + '" to long-term persistent memory?')) return;

          btn.textContent = 'Promoting...';
          btn.disabled = true;

          var res = await fetch('/api/memories/' + encodeURIComponent(key), { method: 'POST' });
          if (res.ok) {
            btn.textContent = 'Promoted!';
            btn.style.background = '#28a745';
            setTimeout(function() {
              loadSuggestions();
              loadMemories();
            }, 900);
          } else {
            btn.textContent = 'Error';
            btn.disabled = false;
          }
        });
      });
    }
  </script>
</body>
</html>`))

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplate.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) handleMemories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	decisionType := strings.TrimSpace(r.URL.Query().Get("decisionType"))
	tags := splitCSV(r.URL.Query().Get("tags"))
	// Project filter: omit -> active project; "*" -> all; "unassigned" -> legacy.
	// Use raw value (do not trim "*") to preserve wildcard semantics.
	project := r.URL.Query().Get("project")

	items, err := h.queryMemories(query, tags, decisionType, project, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allItems, err := h.memory.ListMemories(0, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, memoryListResponse{
		Items:         toMemoryDTOs(items),
		Stats:         buildMemoryStats(allItems),
		Projects:      h.memory.ListProjects(),
		ActiveProject: h.memory.ActiveProject(),
	})
}

func (h *Handler) handleMemoryByKey(w http.ResponseWriter, r *http.Request) {
	key, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/api/memories/"))
	if err != nil || key == "" {
		http.Error(w, "invalid memory key", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := h.memory.Delete(key); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "key": key})
	case http.MethodGet:
		mem, err := h.memory.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, toMemoryDTO(mem))
	case http.MethodPost:
		// Promote this memory (Fase 3 dashboard action)
		if err := h.memory.Promote(key, "high"); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "promoted", "key": key})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) queryMemories(query string, tags []string, decisionType, project string, limit int) ([]*memory.Memory, error) {
	var (
		items []*memory.Memory
		err   error
	)

	switch {
	case query != "" || len(tags) > 0:
		items, err = h.memory.SearchWithProject(query, tags, project)
	case decisionType != "":
		items, err = h.memory.ListMemories(limit, decisionType)
	default:
		items, err = h.memory.ListMemories(limit, "")
	}
	if err != nil {
		return nil, err
	}

	// Apply project filter when listing (Search already filtered).
	if (query == "" && len(tags) == 0) && project != memory.WildcardProject {
		target := project
		if target == "" {
			target = h.memory.ActiveProject()
		}
		filtered := make([]*memory.Memory, 0, len(items))
		for _, item := range items {
			p := item.Project
			if p == "" {
				p = memory.UnassignedProject
			}
			if p == target {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if decisionType != "" && (query != "" || len(tags) > 0) {
		filtered := items[:0]
		for _, item := range items {
			if item.DecisionType == decisionType {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func buildMemoryStats(items []*memory.Memory) memoryStats {
	tags := make(map[string]bool)
	decisionTypes := make(map[string]bool)
	decisionCount := 0
	promotedCount := 0

	for _, item := range items {
		for _, tag := range item.Tags {
			tags[tag] = true
		}
		if item.DecisionType != "" {
			decisionCount++
			decisionTypes[item.DecisionType] = true
		}
		if item.Promoted {
			promotedCount++
		}
	}

	types := make([]string, 0, len(decisionTypes))
	for decisionType := range decisionTypes {
		types = append(types, decisionType)
	}
	sort.Strings(types)

	return memoryStats{
		Total:         len(items),
		DecisionCount: decisionCount,
		TagCount:      len(tags),
		DecisionTypes: types,
		// Fase 3
		PromotedCount: promotedCount,
	}
}

func toMemoryDTOs(items []*memory.Memory) []memoryDTO {
	result := make([]memoryDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toMemoryDTO(item))
	}
	return result
}

func toMemoryDTO(item *memory.Memory) memoryDTO {
	if item == nil {
		return memoryDTO{}
	}
	return memoryDTO{
		Key:          item.Key,
		Content:      item.Content,
		Tags:         item.Tags,
		Timestamp:    item.Timestamp,
		Usage:        item.Usage,
		Project:      item.Project,
		DecisionType: item.DecisionType,
		Reason:       item.Reason,
		Alternatives: item.Alternatives,
		Promoted:     item.Promoted, // Fase 3
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// handleProjectSummary returns a lightweight project summary for session start.
// This is the first enrichment from mcp-go-context to src/context.
func (h *Handler) handleProjectSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.analyzer == nil {
		http.Error(w, "analyzer not available", http.StatusServiceUnavailable)
		return
	}

	// Get project path from query or default to first configured path
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}

	depth := 2 // lightweight by default
	if rawDepth := r.URL.Query().Get("depth"); rawDepth != "" {
		if parsed, err := strconv.Atoi(rawDepth); err == nil && parsed > 0 && parsed <= 5 {
			depth = parsed
		}
	}

	structure, err := h.analyzer.AnalyzeProject(path, depth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a concise summary (not the full structure)
	summary := map[string]any{
		"rootPath":       structure.RootPath,
		"totalFiles":     0,
		"languages":      map[string]int{},
		"topDirectories": []string{},
		"stats": map[string]any{
			"totalSize": 0,
		},
		// Fase 0: light index status for cold-start observability
		"lightIndex": map[string]any{
			"enabled": true,
		},
	}

	// Enrich with actual light index state if analyzer supports it
	if stats := h.analyzer.LightIndexStats(); stats != nil {
		summary["lightIndex"] = stats
	}

	// Count files and languages
	if structure.Structure != nil {
		dirCount := 0
		for dir := range structure.Structure {
			summary["topDirectories"] = append(summary["topDirectories"].([]string), dir)
			dirCount++
		}
		// Keep only top 10 directories
		if dirs := summary["topDirectories"].([]string); len(dirs) > 10 {
			summary["topDirectories"] = dirs[:10]
		}
		_ = dirCount // suppress unused warning
	}

	writeJSON(w, http.StatusOK, summary)
}

// handleQuickContext returns context for a specific query without full analysis.
// This enables on-demand context enrichment from src/context.
func (h *Handler) handleQuickContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.analyzer == nil {
		http.Error(w, "analyzer not available", http.StatusServiceUnavailable)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}

	maxTokens := 2000
	if rawTokens := r.URL.Query().Get("maxTokens"); rawTokens != "" {
		if parsed, err := strconv.Atoi(rawTokens); err == nil && parsed > 0 && parsed <= 10000 {
			maxTokens = parsed
		}
	}

	context, err := h.analyzer.GetRelevantContext(query, nil, maxTokens)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":     query,
		"context":   context,
		"truncated": len(context) > maxTokens*4, // rough token estimate
	})
}
