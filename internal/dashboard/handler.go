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

	"github.com/scopweb/mcp-go-context/internal/memory"
)

type Handler struct {
	memory *memory.Manager
}

type memoryDTO struct {
	Key          string    `json:"key"`
	Content      string    `json:"content"`
	Tags         []string  `json:"tags"`
	Timestamp    time.Time `json:"timestamp"`
	Usage        int       `json:"usage"`
	DecisionType string    `json:"decisionType,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Alternatives []string  `json:"alternatives,omitempty"`
}

type memoryStats struct {
	Total         int      `json:"total"`
	DecisionCount int      `json:"decisionCount"`
	TagCount      int      `json:"tagCount"`
	DecisionTypes []string `json:"decisionTypes"`
}

type memoryListResponse struct {
	Items []memoryDTO `json:"items"`
	Stats memoryStats `json:"stats"`
}

func New(memoryManager *memory.Manager) (*Handler, error) {
	if memoryManager == nil {
		return nil, fmt.Errorf("memory manager is required")
	}

	return &Handler{memory: memoryManager}, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", h.handleDashboard)
	mux.HandleFunc("/api/memories", h.handleMemories)
	mux.HandleFunc("/api/memories/", h.handleMemoryByKey)
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
      </div>
      <div id="memories" class="memory-grid"></div>
    </section>
  </div>

  <script>
    const searchInput = document.getElementById('search');
    const decisionTypeSelect = document.getElementById('decision-type');
    const limitSelect = document.getElementById('limit');
    const refreshButton = document.getElementById('refresh');
    const memoriesContainer = document.getElementById('memories');
    const decisionTypesContainer = document.getElementById('decision-types');

    const statTotal = document.getElementById('stat-total');
    const statDecisions = document.getElementById('stat-decisions');
    const statTags = document.getElementById('stat-tags');

    let refreshTimer;

    async function loadMemories() {
      const params = new URLSearchParams();
      if (searchInput.value.trim()) params.set('query', searchInput.value.trim());
      if (decisionTypeSelect.value) params.set('decisionType', decisionTypeSelect.value);
      params.set('limit', limitSelect.value);

      const response = await fetch('/api/memories?' + params.toString());
      if (!response.ok) {
        memoriesContainer.innerHTML = '<div class="empty">Failed to load memories.</div>';
        return;
      }

      const data = await response.json();
      renderStats(data.stats || {});
      renderDecisionTypes(data.stats?.decisionTypes || []);
      renderMemories(data.items || []);
    }

    function renderStats(stats) {
      statTotal.textContent = stats.total || 0;
      statDecisions.textContent = stats.decisionCount || 0;
      statTags.textContent = stats.tagCount || 0;

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
      if (!items.length) {
        memoriesContainer.innerHTML = '<div class="empty">No memories matched the current filters.</div>';
        return;
      }

      memoriesContainer.innerHTML = items.map(item => {
        const timestamp = item.timestamp ? new Date(item.timestamp).toLocaleString() : 'n/a';
        const tags = (item.tags || []).map(tag => '<span class="tag">' + escapeHtml(tag) + '</span>').join('');
        const alternatives = (item.alternatives || []).length
          ? '<div><strong>Alternatives:</strong> ' + item.alternatives.map(escapeHtml).join(', ') + '</div>'
          : '';
        const reason = item.reason ? '<div><strong>Reason:</strong> ' + escapeHtml(item.reason) + '</div>' : '';
        const badge = item.decisionType ? '<span class="badge">' + escapeHtml(item.decisionType) + '</span>' : '';

        return '' +
          '<article class="memory-card">' +
            '<div class="memory-head">' +
              '<h2 class="memory-key">' + escapeHtml(item.key) + '</h2>' +
              badge +
            '</div>' +
            '<div class="memory-meta">' +
              '<span>Saved ' + escapeHtml(timestamp) + '</span>' +
              '<span>Usage ' + (item.usage || 0) + '</span>' +
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
    refreshButton.addEventListener('click', loadMemories);

    loadMemories();
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

	items, err := h.queryMemories(query, tags, decisionType, limit)
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
		Items: toMemoryDTOs(items),
		Stats: buildMemoryStats(allItems),
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
		mem, err := h.memory.Retrieve(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, toMemoryDTO(mem))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) queryMemories(query string, tags []string, decisionType string, limit int) ([]*memory.Memory, error) {
	var (
		items []*memory.Memory
		err   error
	)

	switch {
	case query != "" || len(tags) > 0:
		items, err = h.memory.Search(query, tags)
	case decisionType != "":
		items, err = h.memory.ListMemories(limit, decisionType)
	default:
		items, err = h.memory.ListMemories(limit, "")
	}
	if err != nil {
		return nil, err
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

	for _, item := range items {
		for _, tag := range item.Tags {
			tags[tag] = true
		}
		if item.DecisionType != "" {
			decisionCount++
			decisionTypes[item.DecisionType] = true
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
		DecisionType: item.DecisionType,
		Reason:       item.Reason,
		Alternatives: item.Alternatives,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
