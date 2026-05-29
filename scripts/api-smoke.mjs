const api = process.env.LITTLELIGHT_API_URL || 'http://localhost:8080'
const day = process.env.LITTLELIGHT_SMOKE_DAY || '2026-05-27'
const runId = Math.random().toString(16).slice(2, 10)
let userId = process.env.LITTLELIGHT_USER_ID || ''
let sessionToken = process.env.LITTLELIGHT_SESSION_TOKEN || ''

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function listItems(response) {
  if (Array.isArray(response)) return response
  if (Array.isArray(response?.items)) return response.items
  return []
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(sessionToken ? { Authorization: `Bearer ${sessionToken}` } : {}),
    ...(userId ? { 'X-User-ID': userId } : {}),
    ...(options.headers || {})
  }
  const response = await fetch(`${api}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  })
  const text = await response.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }
  if (!response.ok) {
    throw new Error(`${options.method || 'GET'} ${path} failed: ${response.status} ${text}`)
  }
  return data
}

async function waitForReady() {
  let lastError = null
  for (let i = 0; i < 60; i += 1) {
    try {
      const health = await request('/healthz')
      const ready = await request('/readyz')
      if (health.status === 'ok' && ready.status === 'ok') return ready
    } catch (error) {
      lastError = error
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
  throw new Error(`API did not become ready: ${lastError?.message || 'unknown error'}`)
}

async function main() {
  const ready = await waitForReady()
  assert(ready.dependencies?.postgres === 'ok', 'postgres readiness is not ok')
  assert(ready.dependencies?.redis === 'ok', 'redis readiness is not ok')

  const session = await request('/api/v1/auth/wechat/mock', {
    method: 'POST',
    body: { code: `smoke-${runId}`, nickName: `Smoke Teacher ${runId}` }
  })
  assert(session.userId, 'mock login did not return userId')
  assert(session.sessionToken, 'mock login did not return sessionToken')
  userId = session.userId
  sessionToken = session.sessionToken

  const profile = await request('/api/v1/me')
  assert(profile.name === `Smoke Teacher ${runId}`, 'profile did not use mock login name')

  const initialDashboard = await request(`/api/v1/dashboard?day=${day}`)
  assert(initialDashboard.todayLabel === day, 'dashboard day mismatch')

  const course = await request(`/api/v1/courses?day=${day}`, {
    method: 'POST',
    body: {
      title: `Smoke course ${runId}`,
      className: 'Class 1',
      location: 'Room 201',
      weekday: 3,
      startTime: '09:30',
      endTime: '10:15',
      note: 'API smoke verification'
    }
  })
  assert(course.id, 'course create did not return id')
  const courseLookup = await request(`/api/v1/courses/${course.id}`)
  assert(courseLookup.title === course.title, 'course lookup mismatch')

  const reminder = await request(`/api/v1/reminders?day=${day}`, {
    method: 'POST',
    body: {
      title: `Smoke reminder ${runId}`,
      category: 'todo',
      remindAt: `${day}T17:20:00+08:00`,
      note: 'API smoke verification'
    }
  })
  assert(reminder.id, 'reminder create did not return id')
  const reminderDone = await request(`/api/v1/reminders/${reminder.id}/complete`, { method: 'POST' })
  assert(reminderDone.ok === true, 'reminder complete did not return ok')

  const parent = await request(`/api/v1/parents?day=${day}`, {
    method: 'POST',
    body: {
      studentName: `Smoke Student ${runId}`,
      className: 'Class 5',
      parentName: `Smoke Parent ${runId}`,
      relationship: 'mother',
      contact: '13800000000',
      communicationStyle: 'empathetic first',
      riskLevel: 'medium',
      importantNotes: 'API smoke verification focus',
      nextAction: 'Follow up tomorrow'
    }
  })
  assert(parent.id, 'parent create did not return id')
  const parentLookup = await request(`/api/v1/parents/${parent.id}`)
  assert(parentLookup.studentName === parent.studentName, 'parent lookup mismatch')

  const record = await request('/api/v1/communication-records', {
    method: 'POST',
    body: {
      parentId: parent.id,
      student: parent.studentName,
      channel: 'wechat',
      reason: 'API smoke verification',
      summary: 'Share observation and next plan',
      result: 'Follow up tomorrow',
      riskLevel: 'medium',
      followUpAt: '2026-05-28T17:20:00+08:00'
    }
  })
  assert(record.id, 'communication record create did not return id')

  const drafts = await request('/api/v1/ai/parent-drafts', {
    method: 'POST',
    body: {
      issue: 'Student has been less focused in class recently',
      parentStyle: 'anxious',
      tone: 'warm',
      studentName: parent.studentName
    }
  })
  assert(Array.isArray(drafts) && drafts.length > 0, 'AI parent drafts are empty')

  const praise = await request('/api/v1/ai/praise', {
    method: 'POST',
    body: {
      persona: 'warm and concise',
      content: 'Finished an end-to-end API smoke verification',
      mood: 'steady'
    }
  })
  assert(praise.content, 'AI praise did not return content')

  const healing = await request('/api/v1/healing/entries', {
    method: 'POST',
    body: {
      type: 'praise',
      mood: 'steady',
      content: 'Finished an end-to-end API smoke verification',
      aiReply: praise.content
    }
  })
  assert(healing.id, 'healing entry create did not return id')

  const favorite = await request('/api/v1/me/favorites', {
    method: 'POST',
    body: {
      type: 'ai_praise',
      title: `Smoke favorite ${runId}`,
      content: praise.content
    }
  })
  assert(favorite.id, 'favorite create did not return id')

  const generationsResponse = await request('/api/v1/ai/generations?limit=1')
  const generations = listItems(generationsResponse)
  assert(Array.isArray(generations) && generations.length > 0, 'AI generation audit list is empty')
  assert(generationsResponse.pageInfo?.count === generations.length, 'AI generation audit list did not return pageInfo')
  assert(generationsResponse.pageInfo?.hasMore === true, 'AI generation audit list should expose hasMore when limit is smaller than result set')

  const dashboard = await request(`/api/v1/dashboard?day=${day}`)
  assert(dashboard.coursesCount >= 1, 'dashboard did not include courses')
  assert(dashboard.followUpsCount >= 1, 'dashboard did not include parent follow ups')

  console.log(`api smoke passed: ${runId}`)
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
