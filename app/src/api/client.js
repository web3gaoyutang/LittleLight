const BASE_URL = import.meta.env?.VITE_API_BASE_URL || '/api/v1'
const USER_ID_KEY = 'littlelight_user_id'
const WECHAT_SESSION_KEY = 'littlelight_wechat_session'
const LOGIN_PAGE = '/pages/login/index'
let redirectingToLogin = false

function authHeader() {
  const session = currentWechatSession()
  if (session?.sessionToken) return { Authorization: `Bearer ${session.sessionToken}` }
  if (!devAuthEnabled()) return {}
  const userId = uni.getStorageSync(USER_ID_KEY)
  return userId ? { 'X-User-ID': userId } : {}
}

function normalizeError(error) {
  if (!error) return { message: '请求失败，请稍后重试' }
  if (typeof error === 'string') return { message: error }
  const statusCode = error.statusCode || error.status || error.code
  const retryAfter = retryAfterSeconds(error.header || error.headers || {})
  return {
    ...error,
    statusCode,
    unauthorized: statusCode === 401,
    rateLimited: statusCode === 429,
    retryAfter,
    message: error.detail || error.error || error.errMsg || '请求失败，请稍后重试'
  }
}

function retryAfterSeconds(headers) {
  const value = headerValue(headers, 'Retry-After')
  if (value === undefined || value === null || value === '') return 0
  const seconds = Number(value)
  if (Number.isFinite(seconds) && seconds >= 0) return seconds
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 0
  return Math.max(0, Math.ceil((date.getTime() - Date.now()) / 1000))
}

function headerValue(headers, name) {
  if (!headers || typeof headers !== 'object') return undefined
  const direct = headers[name]
  if (direct !== undefined) return direct
  const lower = name.toLowerCase()
  const key = Object.keys(headers).find((item) => item.toLowerCase() === lower)
  return key ? headers[key] : undefined
}

function queryString(params = {}) {
  const search = Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
    .join('&')
  return search ? `?${search}` : ''
}

export function listItems(response) {
  if (Array.isArray(response)) return response
  if (Array.isArray(response?.items)) return response.items
  return []
}

export function listPageInfo(response, fallbackLimit = 50, fallbackOffset = 0) {
  const items = listItems(response)
  if (response?.pageInfo) {
    const info = response.pageInfo
    return {
      limit: Number.isFinite(info.limit) ? info.limit : fallbackLimit,
      offset: Number.isFinite(info.offset) ? info.offset : fallbackOffset,
      count: Number.isFinite(info.count) ? info.count : items.length,
      nextOffset: Number.isFinite(info.nextOffset) ? info.nextOffset : fallbackOffset + items.length,
      hasMore: !!info.hasMore
    }
  }
  return {
    limit: fallbackLimit,
    offset: fallbackOffset,
    count: items.length,
    nextOffset: fallbackOffset + items.length,
    hasMore: items.length === fallbackLimit
  }
}

export function request(path, options = {}) {
  return new Promise((resolve, reject) => {
    uni.request({
      url: BASE_URL + path,
      method: options.method || 'GET',
      data: options.data,
      header: { 'Content-Type': 'application/json', ...authHeader(), ...(options.header || {}) },
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data)
        else reject(handleAuthError(normalizeError({ ...(res.data || {}), statusCode: res.statusCode, header: res.header })))
      },
      fail: (error) => reject(handleAuthError(normalizeError(error)))
    })
  })
}

export function upload(path, file, name = 'file') {
  return new Promise((resolve, reject) => {
    const options = {
      url: BASE_URL + path,
      name,
      header: authHeader(),
      success: (res) => {
        let data = res.data
        try {
          data = typeof data === 'string' ? JSON.parse(data) : data
        } catch (error) {
          reject(error)
          return
        }
        if (res.statusCode >= 200 && res.statusCode < 300) resolve(data)
        else reject(handleAuthError(normalizeError({ ...(data || {}), statusCode: res.statusCode, header: res.header })))
      },
      fail: (error) => reject(handleAuthError(normalizeError(error)))
    }
    if (typeof file === 'string') {
      options.filePath = file
    } else {
      options.files = [{ name, file }]
    }
    uni.uploadFile(options)
  })
}

function saveWechatSession(session) {
  if (!session?.userId || !session?.sessionToken) {
    throw new Error('登录响应缺少服务端 session，请重新登录')
  }
  if (sessionExpired(session)) {
    throw new Error('登录 session 已过期，请重新登录')
  }
  uni.setStorageSync(USER_ID_KEY, session.userId)
  uni.setStorageSync(WECHAT_SESSION_KEY, session)
  return session
}

export function currentWechatSession() {
  const session = uni.getStorageSync(WECHAT_SESSION_KEY) || null
  if (sessionExpired(session)) {
    logout()
    return null
  }
  return session
}

export function apiURL(path) {
  return BASE_URL + path
}

export function isLoggedIn() {
  const session = currentWechatSession()
  return !!session?.sessionToken
}

function sessionExpired(session) {
  if (!session?.expiresAt) return false
  const expiresAt = new Date(session.expiresAt).getTime()
  return !Number.isFinite(expiresAt) || expiresAt <= Date.now()
}

export function logout() {
  uni.removeStorageSync(USER_ID_KEY)
  uni.removeStorageSync(WECHAT_SESSION_KEY)
}

export async function logoutRemote() {
  try {
    await request('/auth/logout', { method: 'POST' })
  } finally {
    logout()
  }
}

export function isDevAuthAvailable() {
  return devAuthEnabled()
}

function devAuthEnabled() {
  return !!import.meta.env?.DEV || import.meta.env?.VITE_ENABLE_MOCK_LOGIN === 'true'
}

export function resetAuthRedirect() {
  redirectingToLogin = false
}

function handleAuthError(error) {
  if (error?.unauthorized) redirectToLogin()
  return error
}

function redirectToLogin() {
  logout()
  if (redirectingToLogin) return
  redirectingToLogin = true
  uni.reLaunch({
    url: LOGIN_PAGE,
    complete: () => {
      redirectingToLogin = false
    }
  })
}

export const api = {
  wechatLogin: (data = {}) => request('/auth/wechat', { method: 'POST', data }).then(saveWechatSession),
  wechatMockLogin: (data = {}) => request('/auth/wechat/mock', { method: 'POST', data }).then(saveWechatSession),
  currentWechatSession,
  isLoggedIn,
  isDevAuthAvailable,
  resetAuthRedirect,
  logout,
  logoutRemote,
  me: () => request('/me'),
  updateMe: (data) => request('/me', { method: 'PUT', data }),
  deleteMe: () => request('/me', { method: 'DELETE' }),
  exportMe: () => request('/me/export'),
  favorites: (type, params = {}) => request(`/me/favorites${queryString({ ...params, type })}`),
  createFavorite: (data) => request('/me/favorites', { method: 'POST', data }),
  deleteFavorite: (id) => request(`/me/favorites/${id}`, { method: 'DELETE' }),
  entitlements: () => request('/billing/entitlements'),
  checkout: (data = {}) => request('/billing/checkout', { method: 'POST', data }),
  notificationSettings: () => request('/notifications/settings'),
  updateNotificationSettings: (data) => request('/notifications/settings', { method: 'PUT', data }),
  syncStatus: () => request('/sync/status'),
  dashboard: (day) => request(day ? `/dashboard?day=${encodeURIComponent(day)}` : '/dashboard'),
  courses: (weekday) => request(`/courses?weekday=${weekday}`),
  course: (id) => request(`/courses/${id}`),
  createCourse: (data) => request('/courses', { method: 'POST', data }),
  importCourses: (filePath, options = {}) => upload(`/courses/imports${queryString({ preview: options.preview ? 'true' : '' })}`, filePath),
  courseImportTemplateURL: () => apiURL('/courses/imports/template'),
  updateCourse: (id, data) => request(`/courses/${id}`, { method: 'PUT', data }),
  deleteCourse: (id) => request(`/courses/${id}`, { method: 'DELETE' }),
  reminders: (day) => request(day ? `/reminders?day=${encodeURIComponent(day)}` : '/reminders'),
  reminder: (id) => request(`/reminders/${id}`),
  createReminder: (data) => request('/reminders', { method: 'POST', data }),
  updateReminder: (id, data) => request(`/reminders/${id}`, { method: 'PUT', data }),
  deleteReminder: (id) => request(`/reminders/${id}`, { method: 'DELETE' }),
  completeReminder: (id) => request(`/reminders/${id}/complete`, { method: 'POST' }),
  snoozeReminder: (id, until) => request(`/reminders/${id}/snooze`, { method: 'POST', data: { until } }),
  parents: (params = {}) => request(`/parents${queryString(params)}`),
  parent: (id) => request(`/parents/${id}`),
  createParent: (data) => request('/parents', { method: 'POST', data }),
  importParents: (filePath, options = {}) => upload(`/parents/imports${queryString({ preview: options.preview ? 'true' : '' })}`, filePath),
  parentImportTemplateURL: () => apiURL('/parents/imports/template'),
  updateParent: (id, data) => request(`/parents/${id}`, { method: 'PUT', data }),
  deleteParent: (id) => request(`/parents/${id}`, { method: 'DELETE' }),
  records: (parentId, params = {}) => request(`/communication-records${queryString({ ...params, parentId })}`),
  record: (id) => request(`/communication-records/${id}`),
  createRecord: (data) => request('/communication-records', { method: 'POST', data }),
  updateRecord: (id, data) => request(`/communication-records/${id}`, { method: 'PUT', data }),
  completeRecordFollowUp: (id) => request(`/communication-records/${id}/complete-follow-up`, { method: 'POST' }),
  deleteRecord: (id) => request(`/communication-records/${id}`, { method: 'DELETE' }),
  aiGenerations: (scenario, params = {}) => request(`/ai/generations${queryString({ ...params, scenario })}`),
  aiGeneration: (id) => request(`/ai/generations/${id}`),
  deleteAIGeneration: (id) => request(`/ai/generations/${id}`, { method: 'DELETE' }),
  createAIAction: (generationId, data) => request(`/ai/generations/${generationId}/actions`, { method: 'POST', data }),
  parentDrafts: (data) => request('/ai/parent-drafts', { method: 'POST', data }),
  praise: (data) => request('/ai/praise', { method: 'POST', data }),
  healingEntries: (type, params = {}) => request(`/healing/entries${queryString({ ...params, type })}`),
  healingEntryDetail: (id) => request(`/healing/entries/${id}`),
  healingEntry: (data) => request('/healing/entries', { method: 'POST', data }),
  deleteHealingEntry: (id) => request(`/healing/entries/${id}`, { method: 'DELETE' })
}
