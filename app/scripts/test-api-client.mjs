import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { writeFile } from 'node:fs/promises'

const storage = new Map()
const calls = []
const navigationCalls = []
const modalCalls = []
let nextRequest = { statusCode: 200, data: { ok: true } }
let nextUpload = { statusCode: 200, data: JSON.stringify({ imported: 1, skipped: 0 }) }
let nextModal = { confirm: true }
let nextChosenFile = { path: 'C:/tmp/import.csv' }

globalThis.uni = {
  getStorageSync(key) {
    return storage.get(key)
  },
  setStorageSync(key, value) {
    storage.set(key, value)
  },
  request(options) {
    calls.push({ type: 'request', options })
    if (nextRequest.fail) {
      options.fail(nextRequest.fail)
      return
    }
    options.success(nextRequest)
  },
  uploadFile(options) {
    calls.push({ type: 'upload', options })
    if (nextUpload.fail) {
      options.fail(nextUpload.fail)
      return
    }
    options.success(nextUpload)
  },
  removeStorageSync(key) {
    storage.delete(key)
  },
  reLaunch(options) {
    navigationCalls.push(options)
    options.complete?.()
  },
  showModal(options) {
    modalCalls.push(options)
    options.success?.(nextModal)
  },
  chooseFile(options) {
    calls.push({ type: 'chooseFile', options })
    if (nextChosenFile?.fail) {
      options.fail?.(nextChosenFile.fail)
      return
    }
    options.success?.({ tempFiles: [nextChosenFile] })
  },
  chooseMessageFile(options) {
    calls.push({ type: 'chooseMessageFile', options })
    if (nextChosenFile?.fail) {
      options.fail?.(nextChosenFile.fail)
      return
    }
    options.success?.({ tempFiles: [nextChosenFile] })
  }
}

const modulePath = await buildLoadableClientModule()
const { api, request, upload, currentWechatSession, listItems, listPageInfo, apiWSURL } = await import(pathToFileURL(modulePath))
const uiModulePath = await buildLoadableUIModule()
const { chooseImportFile, confirmAction, isHighRiskSafety } = await import(pathToFileURL(uiModulePath))

await testHighRiskSafetyHelper()
await testConfirmActionHelper()
await testChooseImportFileUsesWechatMessageFileFirst()
await testChooseImportFileFallsBackToChooseFile()
await testRequestUsesBaseURLAndAuthHeader()
await testAPIWSURLBuildsWebSocketURL()
await testRequestDoesNotUseDevHeaderInProduction()
await testExpiredSessionIsClearedBeforeRequest()
await testDevAuthAvailabilityFollowsEnvFlags()
await testRequestRejectsHTTPError()
await testRequestNormalizesRateLimitError()
await testUnauthorizedClearsSessionAndRedirects()
await testWechatLoginPersistsSession()
await testWechatLoginRejectsMissingSessionToken()
await testWechatMockLoginPersistsSession()
await testRequestPrefersBearerSession()
await testListQueryHelpers()
await testListResponseHelpers()
await testAccountDeleteEndpointHelper()
await testAccountExportEndpointHelper()
await testBoundaryEndpointHelpers()
await testCommunicationFollowUpEndpointHelper()
await testAIActionEndpointHelper()
await testAIGenerationDeleteEndpointHelper()
await testAIDraftMetadataPassesThrough()
await testRemoteLogoutRevokesThenClearsSession()
await testRemoteLogoutClearsSessionOnFailure()
await testLogoutClearsSession()
await testUploadParsesJSONAndAddsAuthHeader()
await testImportPreviewAddsPreviewQuery()
await testUploadRejectsInvalidJSON()

console.log('api client tests passed')

async function buildLoadableClientModule() {
  const sourcePath = new URL('../src/api/client.js', import.meta.url)
  let source = await readFile(sourcePath, 'utf8')
  source = source.replace(
    "const BASE_URL = import.meta.env?.VITE_API_BASE_URL || '/api/v1'",
    "const BASE_URL = globalThis.__VITE_API_BASE_URL__ || '/api/v1'"
  )
  source = source.replace(
    "const APP_BASE_URL = import.meta.env?.VITE_APP_API_BASE_URL || BASE_URL",
    "const APP_BASE_URL = globalThis.__VITE_APP_API_BASE_URL__ || BASE_URL"
  )
  source = source.replaceAll('import.meta.env?.DEV', 'globalThis.__VITE_DEV__')
  source = source.replaceAll(
    "import.meta.env?.VITE_ENABLE_MOCK_LOGIN === 'true'",
    "globalThis.__VITE_ENABLE_MOCK_LOGIN__ === 'true'"
  )
  source = source.replace(
    "const absolute = new URL(httpURL, window.location.origin)",
    "const absolute = new URL(httpURL, globalThis.__WINDOW_ORIGIN__ || 'http://localhost:5173')"
  )
  source = rewriteAPIWSURLForNodeTest(source)
  const outputPath = join(tmpdir(), `littlelight-api-client-${Date.now()}.mjs`)
  await writeFile(outputPath, source, 'utf8')
  return outputPath
}

function rewriteAPIWSURLForNodeTest(source) {
  const start = source.indexOf('export function apiWSURL(path, params = {}) {')
  const end = source.indexOf('\nfunction normalizeAbsoluteAppAPIURL', start)
  assert.notEqual(start, -1, 'client module must expose apiWSURL')
  assert.notEqual(end, -1, 'client module must keep normalizeAbsoluteAppAPIURL after apiWSURL')
  const replacement = `export function apiWSURL(path, params = {}) {
  const query = queryString(params)
  if (globalThis.__UNI_PLATFORM__ === 'app') {
    const httpURL = normalizeAbsoluteAppAPIURL(globalThis.__VITE_APP_API_BASE_URL__ || BASE_URL) + path + query
    return httpURL.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:')
  }
  const httpURL = apiURL(path) + query
  const absolute = new URL(httpURL, globalThis.__WINDOW_ORIGIN__ || 'http://localhost:5173')
  absolute.protocol = absolute.protocol === 'https:' ? 'wss:' : 'ws:'
  return absolute.toString()
}
`
  return source.slice(0, start) + replacement + source.slice(end + 1)
}

async function testAPIWSURLBuildsWebSocketURL() {
  reset()
  globalThis.__WINDOW_ORIGIN__ = 'http://localhost:5173'
  globalThis.__UNI_PLATFORM__ = 'h5'

  const url = apiWSURL('/dictation/stream', { token: 'session-token', language: 'zh_cn', sampleRate: 16000 })

  assert.equal(url, 'ws://localhost:5173/api/v1/dictation/stream?token=session-token&language=zh_cn&sampleRate=16000')

  globalThis.__UNI_PLATFORM__ = 'app'
  globalThis.__VITE_APP_API_BASE_URL__ = 'https://api.example.com/api/v1'
  const appURL = apiWSURL('/dictation/stream', { token: 'session-token' })
  assert.equal(appURL, 'wss://api.example.com/api/v1/dictation/stream?token=session-token')

  globalThis.__VITE_APP_API_BASE_URL__ = '/api/v1'
  assert.throws(() => apiWSURL('/dictation/stream'), /VITE_APP_API_BASE_URL/)
}

async function buildLoadableUIModule() {
  const sourcePath = new URL('../src/utils/ui.js', import.meta.url)
  const source = await readFile(sourcePath, 'utf8')
  const outputPath = join(tmpdir(), `littlelight-ui-${Date.now()}.mjs`)
  await writeFile(outputPath, source, 'utf8')
  return outputPath
}

async function testHighRiskSafetyHelper() {
  assert.equal(isHighRiskSafety('crisis_support_required'), true)
  assert.equal(isHighRiskSafety('student_safety_review_required'), true)
  assert.equal(isHighRiskSafety('medical_review_required'), true)
  assert.equal(isHighRiskSafety('teacher_review_required'), false)
  assert.equal(isHighRiskSafety('self_care'), false)
  assert.equal(isHighRiskSafety('unknown'), false)
}

async function testConfirmActionHelper() {
  reset()
  nextModal = { confirm: true }
  const confirmed = await confirmAction({ title: '删除课程', content: '确认删除吗？', confirmText: '删除' })
  assert.equal(confirmed, true)
  assert.equal(modalCalls.length, 1)
  assert.equal(modalCalls[0].title, '删除课程')
  assert.equal(modalCalls[0].confirmText, '删除')

  reset()
  nextModal = { confirm: false }
  assert.equal(await confirmAction({ title: '删除课程' }), false)
}

async function testChooseImportFileUsesWechatMessageFileFirst() {
  reset()
  nextChosenFile = { path: 'wxfile://imports/parents.xlsx' }

  const file = await chooseImportFile()

  assert.equal(file, 'wxfile://imports/parents.xlsx')
  assert.equal(calls.length, 1)
  assert.equal(calls[0].type, 'chooseMessageFile')
  assert.deepEqual(calls[0].options.extension, ['xlsx', 'csv'])
}

async function testChooseImportFileFallsBackToChooseFile() {
  reset()
  const original = globalThis.uni.chooseMessageFile
  delete globalThis.uni.chooseMessageFile
  nextChosenFile = { file: new Blob(['a,b']) }
  try {
    const file = await chooseImportFile({ extensions: ['.csv'] })

    assert.equal(file, nextChosenFile.file)
    assert.equal(calls.length, 1)
    assert.equal(calls[0].type, 'chooseFile')
    assert.deepEqual(calls[0].options.extension, ['.csv'])
  } finally {
    globalThis.uni.chooseMessageFile = original
  }
}

async function testRequestUsesBaseURLAndAuthHeader() {
  reset()
  globalThis.__VITE_DEV__ = true
  storage.set('littlelight_user_id', 'user-1')
  nextRequest = { statusCode: 200, data: { items: [] } }

  const data = await request('/dashboard')

  assert.deepEqual(data, { items: [] })
  assert.equal(calls.length, 1)
  assert.equal(calls[0].options.url, '/api/v1/dashboard')
  assert.equal(calls[0].options.method, 'GET')
  assert.equal(calls[0].options.header['Content-Type'], 'application/json')
  assert.equal(calls[0].options.header['X-User-ID'], 'user-1')
}

async function testRequestDoesNotUseDevHeaderInProduction() {
  reset()
  storage.set('littlelight_user_id', 'user-1')
  nextRequest = { statusCode: 200, data: { ok: true } }

  await request('/me')

  assert.equal(calls[0].options.header.Authorization, undefined)
  assert.equal(calls[0].options.header['X-User-ID'], undefined)
}

async function testExpiredSessionIsClearedBeforeRequest() {
  reset()
  storage.set('littlelight_user_id', 'teacher-1')
  storage.set('littlelight_wechat_session', {
    userId: 'teacher-1',
    sessionToken: 'expired-token',
    expiresAt: '2000-01-01T00:00:00.000Z'
  })
  nextRequest = { statusCode: 401, data: { error: 'missing auth token' } }

  await assert.rejects(() => request('/me'), { unauthorized: true, statusCode: 401 })

  assert.equal(calls[0].options.header.Authorization, undefined)
  assert.equal(storage.has('littlelight_user_id'), false)
  assert.equal(storage.has('littlelight_wechat_session'), false)
}

async function testDevAuthAvailabilityFollowsEnvFlags() {
  reset()
  assert.equal(api.isDevAuthAvailable(), false)

  globalThis.__VITE_DEV__ = true
  assert.equal(api.isDevAuthAvailable(), true)

  globalThis.__VITE_DEV__ = false
  globalThis.__VITE_ENABLE_MOCK_LOGIN__ = 'true'
  assert.equal(api.isDevAuthAvailable(), true)
}

async function testRequestRejectsHTTPError() {
  reset()
  nextRequest = { statusCode: 404, data: { error: 'missing' } }

  await assert.rejects(() => request('/missing'), { error: 'missing' })
}

async function testRequestNormalizesRateLimitError() {
  reset()
  nextRequest = {
    statusCode: 429,
    data: { error: 'rate limited', detail: 'too many AI generation requests' },
    header: { 'Retry-After': '42' }
  }

  await assert.rejects(() => request('/ai/praise'), {
    statusCode: 429,
    rateLimited: true,
    retryAfter: 42,
    message: 'too many AI generation requests'
  })
}

async function testUnauthorizedClearsSessionAndRedirects() {
  reset()
  storage.set('littlelight_user_id', 'teacher-1')
  storage.set('littlelight_wechat_session', { userId: 'teacher-1', sessionToken: 'expired-token' })
  nextRequest = { statusCode: 401, data: { error: 'expired auth token' } }

  await assert.rejects(() => request('/me'), { unauthorized: true, statusCode: 401 })

  assert.equal(storage.has('littlelight_user_id'), false)
  assert.equal(storage.has('littlelight_wechat_session'), false)
  assert.equal(navigationCalls.length, 1)
  assert.equal(navigationCalls[0].url, '/pages/login/index')
}

async function testWechatMockLoginPersistsSession() {
  reset()
  nextRequest = {
    statusCode: 200,
    data: {
      userId: 'teacher-1',
      sessionToken: 'mock-token',
      openId: 'openid',
      profile: { name: '林老师' }
    }
  }

  const session = await api.wechatMockLogin({ code: 'dev-code' })

  assert.equal(calls[0].options.url, '/api/v1/auth/wechat/mock')
  assert.equal(calls[0].options.method, 'POST')
  assert.deepEqual(calls[0].options.data, { code: 'dev-code' })
  assert.equal(storage.get('littlelight_user_id'), 'teacher-1')
  assert.deepEqual(currentWechatSession(), session)
}

async function testWechatLoginPersistsSession() {
  reset()
  nextRequest = {
    statusCode: 200,
    data: {
      userId: 'teacher-real',
      sessionToken: 'real-token',
      openId: 'real-openid',
      profile: { name: '微信老师' },
      expiresAt: '2999-01-01T00:00:00.000Z'
    }
  }

  const session = await api.wechatLogin({ code: 'wx-code', nickName: '微信老师' })

  assert.equal(calls[0].options.url, '/api/v1/auth/wechat')
  assert.equal(calls[0].options.method, 'POST')
  assert.deepEqual(calls[0].options.data, { code: 'wx-code', nickName: '微信老师' })
  assert.equal(storage.get('littlelight_user_id'), 'teacher-real')
  assert.deepEqual(currentWechatSession(), session)
}

async function testWechatLoginRejectsMissingSessionToken() {
  reset()
  nextRequest = {
    statusCode: 200,
    data: {
      userId: 'teacher-missing-token',
      openId: 'real-openid',
      profile: { name: '微信老师' }
    }
  }

  await assert.rejects(() => api.wechatLogin({ code: 'wx-code', nickName: '微信老师' }), /缺少服务端 session/)

  assert.equal(storage.has('littlelight_user_id'), false)
  assert.equal(storage.has('littlelight_wechat_session'), false)
}

async function testRequestPrefersBearerSession() {
  reset()
  storage.set('littlelight_user_id', 'user-legacy')
  storage.set('littlelight_wechat_session', { userId: 'teacher-1', sessionToken: 'session-token' })
  nextRequest = { statusCode: 200, data: { ok: true } }

  await request('/me')

  assert.equal(calls[0].options.header.Authorization, 'Bearer session-token')
  assert.equal(calls[0].options.header['X-User-ID'], undefined)
}

async function testListQueryHelpers() {
  reset()
  nextRequest = { statusCode: 200, data: [] }

  await api.parents({ q: '林', limit: 20, offset: 10 })
  await api.records('parent-1', { q: '睡眠', limit: 10 })
  await api.aiGenerations('praise', { q: '安全' })
  await api.healingEntries('praise', { limit: 5 })
  await api.favorites('ai_praise', { q: '恢复', limit: 5, offset: 10 })

  assert.equal(calls[0].options.url, '/api/v1/parents?q=%E6%9E%97&limit=20&offset=10')
  assert.equal(calls[1].options.url, '/api/v1/communication-records?q=%E7%9D%A1%E7%9C%A0&limit=10&parentId=parent-1')
  assert.equal(calls[2].options.url, '/api/v1/ai/generations?q=%E5%AE%89%E5%85%A8&scenario=praise')
  assert.equal(calls[3].options.url, '/api/v1/healing/entries?limit=5&type=praise')
  assert.equal(calls[4].options.url, '/api/v1/me/favorites?q=%E6%81%A2%E5%A4%8D&limit=5&offset=10&type=ai_praise')
}

async function testListResponseHelpers() {
  const page = {
    items: [{ id: 'parent-1' }],
    pageInfo: { limit: 1, offset: 0, count: 1, nextOffset: 1, hasMore: true }
  }
  assert.deepEqual(listItems(page), [{ id: 'parent-1' }])
  assert.deepEqual(listPageInfo(page, 20, 0), page.pageInfo)
  assert.deepEqual(listItems([{ id: 'legacy' }]), [{ id: 'legacy' }])
  assert.deepEqual(listPageInfo([{ id: 'legacy' }], 1, 2), {
    limit: 1,
    offset: 2,
    count: 1,
    nextOffset: 3,
    hasMore: true
  })
}

async function testAccountDeleteEndpointHelper() {
  reset()
  nextRequest = { statusCode: 200, data: { ok: true } }

  await api.deleteMe()

  assert.equal(calls[0].options.url, '/api/v1/me')
  assert.equal(calls[0].options.method, 'DELETE')
}

async function testAccountExportEndpointHelper() {
  reset()
  nextRequest = {
    statusCode: 200,
    data: {
      profile: { name: '林老师' },
      courses: [],
      reminders: [],
      parents: [],
      communicationRecords: [],
      healingEntries: [],
      aiGenerations: [],
      favorites: [],
      exportedAt: '2026-05-29T00:00:00Z'
    }
  }

  await api.exportMe()

  assert.equal(calls[0].options.url, '/api/v1/me/export')
  assert.equal(calls[0].options.method, 'GET')
}

async function testBoundaryEndpointHelpers() {
  reset()
  nextRequest = { statusCode: 200, data: { ok: true } }

  await api.entitlements()
  await api.checkout({ plan: 'monthly', provider: 'wechat', mock: true })
  await api.notificationSettings()
  await api.updateNotificationSettings({ reminderPolicy: 'normal' })
  await api.syncStatus()

  assert.equal(calls[0].options.url, '/api/v1/billing/entitlements')
  assert.equal(calls[0].options.method, 'GET')
  assert.equal(calls[1].options.url, '/api/v1/billing/checkout')
  assert.equal(calls[1].options.method, 'POST')
  assert.deepEqual(calls[1].options.data, { plan: 'monthly', provider: 'wechat', mock: true })
  assert.equal(calls[2].options.url, '/api/v1/notifications/settings')
  assert.equal(calls[3].options.url, '/api/v1/notifications/settings')
  assert.equal(calls[3].options.method, 'PUT')
  assert.deepEqual(calls[3].options.data, { reminderPolicy: 'normal' })
  assert.equal(calls[4].options.url, '/api/v1/sync/status')
}

async function testCommunicationFollowUpEndpointHelper() {
  reset()
  nextRequest = { statusCode: 200, data: { id: 'record-1', followUpStatus: 'done' } }

  await api.completeRecordFollowUp('record-1')

  assert.equal(calls[0].options.url, '/api/v1/communication-records/record-1/complete-follow-up')
  assert.equal(calls[0].options.method, 'POST')
}

async function testAIActionEndpointHelper() {
	reset()
	nextRequest = { statusCode: 200, data: { id: 'action-1', action: 'copy' } }

  await api.createAIAction('generation-1', { action: 'copy', draftId: 'draft-1', note: '已复核' })
  await api.createAIAction('generation-1', { action: 'review_confirmed', draftId: 'draft-1', note: '确认复核' })
  await api.createAIAction('generation-1', { action: 'review_skipped', draftId: 'draft-1', note: '暂缓使用' })

  assert.equal(calls[0].options.url, '/api/v1/ai/generations/generation-1/actions')
  assert.equal(calls[0].options.method, 'POST')
  assert.deepEqual(calls[0].options.data, { action: 'copy', draftId: 'draft-1', note: '已复核' })
  assert.deepEqual(calls[1].options.data, { action: 'review_confirmed', draftId: 'draft-1', note: '确认复核' })
	assert.deepEqual(calls[2].options.data, { action: 'review_skipped', draftId: 'draft-1', note: '暂缓使用' })
}

async function testAIGenerationDeleteEndpointHelper() {
  reset()
  nextRequest = { statusCode: 200, data: { ok: true } }

  await api.deleteAIGeneration('generation-1')

  assert.equal(calls[0].options.url, '/api/v1/ai/generations/generation-1')
  assert.equal(calls[0].options.method, 'DELETE')
}

async function testAIDraftMetadataPassesThrough() {
	reset()
	nextRequest = {
    statusCode: 200,
    data: {
      id: 'praise-1',
      generationId: 'generation-1',
      content: '先给自己一点恢复空间。',
      safety: 'self_care',
      provider: 'mock',
      source: 'local_fallback',
      fallback: true,
      reviewRequired: true,
      safetyNote: '模型服务暂不可用，当前内容来自本地降级模板。',
      safetyLevel: 'review',
      safetyReason: '模型服务暂不可用，本次内容来自本地降级模板，需要人工复核后再使用。',
      safetySignals: ['fallback']
    }
  }

  const draft = await api.praise({ content: '今天很忙' })

  assert.equal(draft.fallback, true)
  assert.equal(draft.reviewRequired, true)
  assert.equal(draft.source, 'local_fallback')
  assert.equal(draft.generationId, 'generation-1')
  assert.equal(draft.safetyLevel, 'review')
  assert.equal(draft.safetyReason, '模型服务暂不可用，本次内容来自本地降级模板，需要人工复核后再使用。')
  assert.deepEqual(draft.safetySignals, ['fallback'])
}

async function testRemoteLogoutRevokesThenClearsSession() {
  reset()
  storage.set('littlelight_user_id', 'teacher-1')
  storage.set('littlelight_wechat_session', { userId: 'teacher-1', sessionToken: 'session-token' })
  nextRequest = { statusCode: 200, data: { ok: true } }

  await api.logoutRemote()

  assert.equal(calls[0].options.url, '/api/v1/auth/logout')
  assert.equal(calls[0].options.method, 'POST')
  assert.equal(calls[0].options.header.Authorization, 'Bearer session-token')
  assert.equal(storage.has('littlelight_user_id'), false)
  assert.equal(storage.has('littlelight_wechat_session'), false)
}

async function testRemoteLogoutClearsSessionOnFailure() {
  reset()
  storage.set('littlelight_user_id', 'teacher-1')
  storage.set('littlelight_wechat_session', { userId: 'teacher-1', sessionToken: 'session-token' })
  nextRequest = { statusCode: 500, data: { error: 'server down' } }

  await assert.rejects(() => api.logoutRemote(), { error: 'server down' })

  assert.equal(storage.has('littlelight_user_id'), false)
  assert.equal(storage.has('littlelight_wechat_session'), false)
}

async function testLogoutClearsSession() {
  reset()
  storage.set('littlelight_user_id', 'teacher-1')
  storage.set('littlelight_wechat_session', { userId: 'teacher-1', sessionToken: 'token' })

  api.logout()

  assert.equal(storage.has('littlelight_user_id'), false)
  assert.equal(storage.has('littlelight_wechat_session'), false)
}

async function testUploadParsesJSONAndAddsAuthHeader() {
  reset()
  globalThis.__VITE_DEV__ = true
  storage.set('littlelight_user_id', 'teacher-2')
  nextUpload = { statusCode: 200, data: JSON.stringify({ imported: 2, skipped: 0 }) }

  const result = await upload('/courses/imports', 'C:/tmp/courses.xlsx')

  assert.deepEqual(result, { imported: 2, skipped: 0 })
  assert.equal(calls[0].options.url, '/api/v1/courses/imports')
  assert.equal(calls[0].options.filePath, 'C:/tmp/courses.xlsx')
  assert.equal(calls[0].options.header['X-User-ID'], 'teacher-2')
}

async function testImportPreviewAddsPreviewQuery() {
  reset()
  nextUpload = { statusCode: 200, data: JSON.stringify({ imported: 0, skipped: 0, preview: [] }) }

  await api.importCourses('C:/tmp/courses.csv', { preview: true })
  await api.importParents('C:/tmp/parents.csv', { preview: true })

  assert.equal(calls[0].options.url, '/api/v1/courses/imports?preview=true')
  assert.equal(calls[1].options.url, '/api/v1/parents/imports?preview=true')
}

async function testUploadRejectsInvalidJSON() {
  reset()
  nextUpload = { statusCode: 200, data: 'not-json' }

  await assert.rejects(() => upload('/parents/imports', 'C:/tmp/parents.xlsx'), SyntaxError)
}

function reset() {
  storage.clear()
  calls.length = 0
  navigationCalls.length = 0
  modalCalls.length = 0
  globalThis.__VITE_DEV__ = false
  globalThis.__VITE_ENABLE_MOCK_LOGIN__ = 'false'
  api.resetAuthRedirect()
  nextRequest = { statusCode: 200, data: { ok: true } }
  nextUpload = { statusCode: 200, data: JSON.stringify({ ok: true }) }
  nextModal = { confirm: true }
  nextChosenFile = { path: 'C:/tmp/import.csv' }
}
