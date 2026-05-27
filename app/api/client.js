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
  dashboard: () => request('/dashboard'),
  courses: (weekday) => request(`/courses?weekday=${weekday}`),
  reminders: () => request('/reminders'),
  createReminder: (data) => request('/reminders', { method: 'POST', data }),
  completeReminder: (id) => request(`/reminders/${id}/complete`, { method: 'POST' }),
  parents: () => request('/parents'),
  records: () => request('/communication-records'),
  parentDrafts: (data) => request('/ai/parent-drafts', { method: 'POST', data }),
  praise: (data) => request('/ai/praise', { method: 'POST', data }),
  healingEntry: (data) => request('/healing/entries', { method: 'POST', data })
}
