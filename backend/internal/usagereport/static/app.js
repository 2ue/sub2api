const AUTH_TOKEN_KEY = 'auth_token'
const API_BASE = 'api/v1/report'

const els = {
  sessionMeta: document.getElementById('session-meta'),
  queryTimezone: document.getElementById('query-timezone'),
  startDate: document.getElementById('start-date'),
  endDate: document.getElementById('end-date'),
  queryButton: document.getElementById('query-button'),
  scopeToggle: document.getElementById('scope-toggle'),
  otherScopeButton: document.getElementById('other-scope-btn'),
  searchPanel: document.getElementById('search-panel'),
  userSearch: document.getElementById('user-search'),
  searchResults: document.getElementById('search-results'),
  selectedUser: document.getElementById('selected-user'),
  summaryRange: document.getElementById('summary-range'),
  metricRequests: document.getElementById('metric-requests'),
  metricTokens: document.getElementById('metric-tokens'),
  metricCost: document.getElementById('metric-cost'),
  metricGroups: document.getElementById('metric-groups'),
  metricPlatforms: document.getElementById('metric-platforms'),
  groupTbody: document.getElementById('group-tbody'),
  platformTbody: document.getElementById('platform-tbody'),
  errorBanner: document.getElementById('error-banner'),
}

const state = {
  bootstrap: null,
  selectedUser: null,
  scope: 'self',
  searchTimer: null,
  loading: false,
}

function todayString() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function currency(value) {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(Number(value || 0))
}

function number(value) {
  return new Intl.NumberFormat(undefined).format(Number(value || 0))
}

function showError(message) {
  if (!message) {
    els.errorBanner.classList.add('hidden')
    els.errorBanner.textContent = ''
    return
  }
  els.errorBanner.textContent = message
  els.errorBanner.classList.remove('hidden')
}

function setLoading(loading) {
  state.loading = loading
  els.queryButton.disabled = loading
  els.queryButton.textContent = loading ? 'Loading...' : 'Query'
}

async function api(path, options = {}) {
  const token = localStorage.getItem(AUTH_TOKEN_KEY)
  if (!token) {
    throw new Error('Missing auth token')
  }

  const headers = new Headers(options.headers || {})
  headers.set('Authorization', `Bearer ${token}`)
  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(path, {
    credentials: 'include',
    ...options,
    headers,
  })

  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    throw new Error(payload?.message || `HTTP ${response.status}`)
  }
  if (!payload || payload.code !== 0) {
    throw new Error(payload?.message || 'Request failed')
  }
  return payload.data
}

function setSelectedUser(user) {
  state.selectedUser = user
  renderSelectedUser()
}

function renderSelectedUser() {
  if (!state.bootstrap) {
    els.selectedUser.textContent = ''
    return
  }
  const current = state.bootstrap.current_user
  const target = state.selectedUser || current
  const sameUser = target.id === current.id
  const suffix = target.deleted ? ' (deleted)' : ''
  els.selectedUser.textContent = sameUser
    ? `Selected: ${target.display_name || target.email}${suffix}`
    : `Selected: ${target.display_name || target.email}${suffix} [${target.role || 'user'}]`
}

function setScope(scope) {
  state.scope = scope
  const buttons = els.scopeToggle.querySelectorAll('.segmented-option')
  buttons.forEach((button) => {
    button.classList.toggle('active', button.dataset.scope === scope)
  })

  const canSearch = Boolean(state.bootstrap?.can_search_users)
  const showOther = scope === 'other' && canSearch
  els.searchPanel.classList.toggle('hidden', !showOther)
  if (!showOther) {
    els.searchResults.innerHTML = ''
    els.userSearch.value = ''
    setSelectedUser(state.bootstrap?.current_user || null)
  }
  renderSelectedUser()
}

function renderSearchResults(users) {
  if (!users.length) {
    els.searchResults.innerHTML = '<div class="muted">No users</div>'
    return
  }

  const currentId = state.bootstrap?.current_user?.id
  els.searchResults.innerHTML = users
    .map((user) => {
      const active = state.selectedUser && state.selectedUser.id === user.id ? 'active' : ''
      const deleted = user.deleted ? ' (deleted)' : ''
      const self = user.id === currentId ? 'Self' : user.role || 'user'
      return `
        <button type="button" class="search-hit ${active}" data-user-id="${user.id}">
          <div>${user.display_name || user.email}${deleted}</div>
          <div class="muted">${user.email} · ${self}</div>
        </button>
      `
    })
    .join('')

  els.searchResults.querySelectorAll('[data-user-id]').forEach((button) => {
    button.addEventListener('click', () => {
      const id = Number(button.getAttribute('data-user-id'))
      const user = users.find((item) => item.id === id)
      if (user) {
        setSelectedUser(user)
      }
    })
  })
}

async function searchUsers(query) {
  if (!state.bootstrap?.can_search_users) {
    return
  }
  const q = query.trim()
  if (!q) {
    els.searchResults.innerHTML = ''
    return
  }
  const users = await api(`${API_BASE}/users?q=${encodeURIComponent(q)}`)
  renderSearchResults(users)
}

function scheduleSearch(value) {
  clearTimeout(state.searchTimer)
  state.searchTimer = setTimeout(() => {
    searchUsers(value).catch((error) => showError(error.message))
  }, 250)
}

function renderSummary(summary) {
  const { query, totals, group_breakdown: groupRows, platform_breakdown: platformRows } = summary
  const target = query.target_user

  els.summaryRange.textContent = `${query.start_date} -> ${query.end_date} · ${query.timezone} · ${target.display_name || target.email}`
  els.metricRequests.textContent = number(totals.requests)
  els.metricTokens.textContent = number(totals.total_tokens)
  els.metricCost.textContent = currency(totals.actual_cost)
  els.metricGroups.textContent = number(groupRows.length)
  els.metricPlatforms.textContent = number(platformRows.length)

  if (groupRows.length === 0) {
    els.groupTbody.innerHTML = '<tr><td colspan="5" class="empty">No rows</td></tr>'
  } else {
    els.groupTbody.innerHTML = groupRows
      .map((row) => `
        <tr>
          <td>${row.group_name || 'Ungrouped'}</td>
          <td>${row.platform || 'unknown'}</td>
          <td class="num">${number(row.requests)}</td>
          <td class="num">${number(row.total_tokens)}</td>
          <td class="num">${currency(row.actual_cost)}</td>
        </tr>
      `)
      .join('')
  }

  if (platformRows.length === 0) {
    els.platformTbody.innerHTML = '<tr><td colspan="4" class="empty">No rows</td></tr>'
  } else {
    els.platformTbody.innerHTML = platformRows
      .map((row) => `
        <tr>
          <td>${row.platform || 'unknown'}</td>
          <td class="num">${number(row.requests)}</td>
          <td class="num">${number(row.total_tokens)}</td>
          <td class="num">${currency(row.actual_cost)}</td>
        </tr>
      `)
      .join('')
  }
}

async function runQuery() {
  if (!state.bootstrap) {
    return
  }
  const startDate = els.startDate.value.trim()
  const endDate = els.endDate.value.trim()
  if (!startDate || !endDate) {
    showError('Start date and end date are required.')
    return
  }

  const actor = state.bootstrap.current_user
  let userId = null
  if (state.scope === 'other') {
    if (!state.bootstrap.can_search_users) {
      showError('Current session cannot query other users.')
      return
    }
    if (!state.selectedUser || state.selectedUser.id === actor.id) {
      showError('Select a user first.')
      return
    }
    userId = state.selectedUser.id
  }

  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || ''
  const body = {
    start_date: startDate,
    end_date: endDate,
    timezone,
  }
  if (userId) {
    body.user_id = userId
  }

  showError('')
  setLoading(true)
  try {
    const summary = await api(`${API_BASE}/summary`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
    renderSummary(summary)
  } catch (error) {
    showError(error.message)
  } finally {
    setLoading(false)
  }
}

async function bootstrap() {
  const token = localStorage.getItem(AUTH_TOKEN_KEY)
  if (!token) {
    els.sessionMeta.textContent = 'Missing token'
    showError('Sign in first.')
    setLoading(false)
    els.queryButton.disabled = true
    els.queryButton.textContent = 'Query'
    return
  }

  showError('')
  setLoading(true)
  try {
    const data = await api(`${API_BASE}/bootstrap`)
    state.bootstrap = data
    els.sessionMeta.textContent = `${data.current_user.display_name || data.current_user.email} · ${data.current_user.role}`
    els.queryTimezone.textContent = `Browser timezone: ${Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown'}`

    const today = todayString()
    els.startDate.value = today
    els.endDate.value = today

    setSelectedUser(data.current_user)
    setScope('self')
    if (!data.can_search_users) {
      els.otherScopeButton.disabled = true
      els.otherScopeButton.title = 'Not available for this account'
    }
  } catch (error) {
    els.sessionMeta.textContent = 'Bootstrap failed'
    showError(error.message)
    els.queryButton.disabled = true
    els.queryButton.textContent = 'Query'
  } finally {
    setLoading(false)
    if (!state.bootstrap) {
      els.queryButton.disabled = true
      els.queryButton.textContent = 'Query'
    }
  }
}

function bindEvents() {
  els.queryButton.addEventListener('click', () => {
    runQuery().catch((error) => showError(error.message))
  })

  els.scopeToggle.addEventListener('click', (event) => {
    const target = event.target
    if (!(target instanceof HTMLElement)) {
      return
    }
    const scope = target.dataset.scope
    if (!scope) {
      return
    }
    if (scope === 'other' && !state.bootstrap?.can_search_users) {
      return
    }
    setScope(scope)
  })

  els.userSearch.addEventListener('input', (event) => {
    const target = event.target
    if (!(target instanceof HTMLInputElement)) {
      return
    }
    scheduleSearch(target.value)
  })

  els.startDate.addEventListener('change', () => showError(''))
  els.endDate.addEventListener('change', () => showError(''))
}

document.title = 'Usage Report'
bindEvents()
bootstrap().catch((error) => showError(error.message))
