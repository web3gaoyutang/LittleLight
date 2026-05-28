import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { writeFile } from 'node:fs/promises'

const storage = new Map()
const calls = []
let nextRequest = { statusCode: 200, data: { ok: true } }
let nextUpload = { statusCode: 200, data: JSON.stringify({ imported: 1, skipped: 0 }) }

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
  }
}

const modulePath = await buildLoadableClientModule()
const { api, request, upload, currentWechatSession } = await import(pathToFileURL(modulePath))

await testRequestUsesBaseURLAndAuthHeader()
await testRequestRejectsHTTPError()
await testWechatMockLoginPersistsSession()
await testUploadParsesJSONAndAddsAuthHeader()
await testUploadRejectsInvalidJSON()

console.log('api client tests passed')

async function buildLoadableClientModule() {
  const sourcePath = new URL('../src/api/client.js', import.meta.url)
  let source = await readFile(sourcePath, 'utf8')
  source = source.replace(
    "const BASE_URL = import.meta.env?.VITE_API_BASE_URL || '/api/v1'",
    "const BASE_URL = globalThis.__VITE_API_BASE_URL__ || '/api/v1'"
  )
  const outputPath = join(tmpdir(), `littlelight-api-client-${Date.now()}.mjs`)
  await writeFile(outputPath, source, 'utf8')
  return outputPath
}

async function testRequestUsesBaseURLAndAuthHeader() {
  reset()
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

async function testRequestRejectsHTTPError() {
  reset()
  nextRequest = { statusCode: 404, data: { error: 'missing' } }

  await assert.rejects(() => request('/missing'), { error: 'missing' })
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

async function testUploadParsesJSONAndAddsAuthHeader() {
  reset()
  storage.set('littlelight_user_id', 'teacher-2')
  nextUpload = { statusCode: 200, data: JSON.stringify({ imported: 2, skipped: 0 }) }

  const result = await upload('/courses/imports', 'C:/tmp/courses.xlsx')

  assert.deepEqual(result, { imported: 2, skipped: 0 })
  assert.equal(calls[0].options.url, '/api/v1/courses/imports')
  assert.equal(calls[0].options.filePath, 'C:/tmp/courses.xlsx')
  assert.equal(calls[0].options.header['X-User-ID'], 'teacher-2')
}

async function testUploadRejectsInvalidJSON() {
  reset()
  nextUpload = { statusCode: 200, data: 'not-json' }

  await assert.rejects(() => upload('/parents/imports', 'C:/tmp/parents.xlsx'), SyntaxError)
}

function reset() {
  storage.clear()
  calls.length = 0
  nextRequest = { statusCode: 200, data: { ok: true } }
  nextUpload = { statusCode: 200, data: JSON.stringify({ ok: true }) }
}
