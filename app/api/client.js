const BASE_URL = import.meta.env?.VITE_API_BASE_URL || '/api/v1'

export function request(path, options = {}) {
  return new Promise((resolve, reject) => {
    uni.request({
      url: BASE_URL + path,
      method: options.method || 'GET',
      data: options.data,
      header: { 'Content-Type': 'application/json', ...(options.header || {}) },
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data)
        else reject(res.data || res)
      },
      fail: reject
    })
  })
}

export const api = {
  me: () => request('/me'),
  updateMe: (data) => request('/me', { method: 'PUT', data }),
  favorites: (type) => request(type ? `/me/favorites?type=${encodeURIComponent(type)}` : '/me/favorites'),
  createFavorite: (data) => request('/me/favorites', { method: 'POST', data }),
  deleteFavorite: (id) => request(`/me/favorites/${id}`, { method: 'DELETE' }),
  dashboard: (day) => request(day ? `/dashboard?day=${encodeURIComponent(day)}` : '/dashboard'),
  courses: (weekday) => request(`/courses?weekday=${weekday}`),
  course: (id) => request(`/courses/${id}`),
  createCourse: (data) => request('/courses', { method: 'POST', data }),
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
  updateParent: (id, data) => request(`/parents/${id}`, { method: 'PUT', data }),
  deleteParent: (id) => request(`/parents/${id}`, { method: 'DELETE' }),
  records: (parentId) => request(parentId ? `/communication-records?parentId=${encodeURIComponent(parentId)}` : '/communication-records'),
  record: (id) => request(`/communication-records/${id}`),
  createRecord: (data) => request('/communication-records', { method: 'POST', data }),
  updateRecord: (id, data) => request(`/communication-records/${id}`, { method: 'PUT', data }),
  deleteRecord: (id) => request(`/communication-records/${id}`, { method: 'DELETE' }),
  parentDrafts: (data) => request('/ai/parent-drafts', { method: 'POST', data }),
  praise: (data) => request('/ai/praise', { method: 'POST', data }),
  healingEntry: (data) => request('/healing/entries', { method: 'POST', data })
}
