const BASE_URL = import.meta.env?.VITE_API_BASE_URL || '/api/v1'
const USER_ID_KEY = 'littlelight_user_id'
const WECHAT_SESSION_KEY = 'littlelight_wechat_session'

function authHeader() {
  const userId = uni.getStorageSync(USER_ID_KEY)
  return userId ? { 'X-User-ID': userId } : {}
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
        else reject(res.data || res)
      },
      fail: reject
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
        else reject(data || res)
      },
      fail: reject
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
  if (session?.userId) {
    uni.setStorageSync(USER_ID_KEY, session.userId)
    uni.setStorageSync(WECHAT_SESSION_KEY, session)
  }
  return session
}

export function currentWechatSession() {
  return uni.getStorageSync(WECHAT_SESSION_KEY) || null
}

export const api = {
  wechatMockLogin: (data = {}) => request('/auth/wechat/mock', { method: 'POST', data }).then(saveWechatSession),
  currentWechatSession,
  me: () => request('/me'),
  updateMe: (data) => request('/me', { method: 'PUT', data }),
  favorites: (type) => request(type ? `/me/favorites?type=${encodeURIComponent(type)}` : '/me/favorites'),
  createFavorite: (data) => request('/me/favorites', { method: 'POST', data }),
  deleteFavorite: (id) => request(`/me/favorites/${id}`, { method: 'DELETE' }),
  dashboard: (day) => request(day ? `/dashboard?day=${encodeURIComponent(day)}` : '/dashboard'),
  courses: (weekday) => request(`/courses?weekday=${weekday}`),
  course: (id) => request(`/courses/${id}`),
  createCourse: (data) => request('/courses', { method: 'POST', data }),
  importCourses: (filePath) => upload('/courses/imports', filePath),
  updateCourse: (id, data) => request(`/courses/${id}`, { method: 'PUT', data }),
  deleteCourse: (id) => request(`/courses/${id}`, { method: 'DELETE' }),
  reminders: (day) => request(day ? `/reminders?day=${encodeURIComponent(day)}` : '/reminders'),
  reminder: (id) => request(`/reminders/${id}`),
  createReminder: (data) => request('/reminders', { method: 'POST', data }),
  updateReminder: (id, data) => request(`/reminders/${id}`, { method: 'PUT', data }),
  deleteReminder: (id) => request(`/reminders/${id}`, { method: 'DELETE' }),
  completeReminder: (id) => request(`/reminders/${id}/complete`, { method: 'POST' }),
  snoozeReminder: (id, until) => request(`/reminders/${id}/snooze`, { method: 'POST', data: { until } }),
  parents: () => request('/parents'),
  parent: (id) => request(`/parents/${id}`),
  createParent: (data) => request('/parents', { method: 'POST', data }),
  importParents: (filePath) => upload('/parents/imports', filePath),
  updateParent: (id, data) => request(`/parents/${id}`, { method: 'PUT', data }),
  deleteParent: (id) => request(`/parents/${id}`, { method: 'DELETE' }),
  records: (parentId) => request(parentId ? `/communication-records?parentId=${encodeURIComponent(parentId)}` : '/communication-records'),
  record: (id) => request(`/communication-records/${id}`),
  createRecord: (data) => request('/communication-records', { method: 'POST', data }),
  updateRecord: (id, data) => request(`/communication-records/${id}`, { method: 'PUT', data }),
  deleteRecord: (id) => request(`/communication-records/${id}`, { method: 'DELETE' }),
  aiGenerations: (scenario) => request(scenario ? `/ai/generations?scenario=${encodeURIComponent(scenario)}` : '/ai/generations'),
  aiGeneration: (id) => request(`/ai/generations/${id}`),
  parentDrafts: (data) => request('/ai/parent-drafts', { method: 'POST', data }),
  praise: (data) => request('/ai/praise', { method: 'POST', data }),
  healingEntries: (type) => request(type ? `/healing/entries?type=${encodeURIComponent(type)}` : '/healing/entries'),
  healingEntryDetail: (id) => request(`/healing/entries/${id}`),
  healingEntry: (data) => request('/healing/entries', { method: 'POST', data }),
  deleteHealingEntry: (id) => request(`/healing/entries/${id}`, { method: 'DELETE' })
}
