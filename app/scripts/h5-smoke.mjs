import { mkdtemp, rm } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn } from 'node:child_process'

const appURL = process.env.LITTLELIGHT_H5_URL || 'http://localhost:5173'
const apiURL = process.env.LITTLELIGHT_API_URL || 'http://localhost:8080'
const chromePath = process.env.CHROME_PATH || findChromePath()
const runId = Math.random().toString(16).slice(2, 8)
const reminderTitle = `H5SmokeTodo-${runId}`
const teacherName = `H5 Smoke ${runId}`
const remotePort = Number(process.env.CHROME_REMOTE_DEBUGGING_PORT || 9223)
let userDataDir = ''
let chrome = null
let websocket = null
let nextMessageID = 1
const callbacks = new Map()

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function findChromePath() {
  const candidates = [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
    '/usr/bin/google-chrome',
    '/usr/bin/google-chrome-stable',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser'
  ]
  return candidates.find((candidate) => existsSync(candidate)) || ''
}

async function main() {
  if (!chromePath) {
    throw new Error('Chrome executable not found. Set CHROME_PATH to run H5 smoke.')
  }
  await waitForHTTP(`${apiURL}/readyz`, (data) => data.status === 'ok')
  await waitForHTTP(appURL, () => true)
  await launchChrome()
  const tab = await newCDPTab(`${appURL}/#/pages/login/index`)
  await connectCDP(tab.webSocketDebuggerUrl)
  await command('Runtime.enable')
  await command('Page.enable')
  await command('DOM.enable')
  await waitForPageReady()
  await clearStorage()
  await navigate(`${appURL}/#/pages/login/index`)
  await waitForText('开发登录')
  await setInputByTestID('login-nickname-input', teacherName)
  await clickByTestID('dev-login-button')
  await waitForHash('/pages/home/index')
  await waitForText(teacherName)
  await clickByTestID('home-schedule-button')
  await waitForHash('/pages/schedule/index')
  await waitForText('待办事项')
  await clickByTestID('reminder-add-button')
  await waitForText('保存待办')
  await setInputByTestID('reminder-title-input', reminderTitle)
  await setInputByTestID('reminder-category-input', 'H5Smoke')
  await setInputByTestID('reminder-note-input', 'created from h5 smoke')
  await clickByTestID('reminder-save-button')
  await waitForText(reminderTitle)
  await assertReminderPersisted()
  console.log(`h5 smoke passed: ${reminderTitle}`)
}

async function launchChrome() {
  userDataDir = await mkdtemp(join(tmpdir(), 'littlelight-h5-smoke-'))
  chrome = spawn(chromePath, [
    `--remote-debugging-port=${remotePort}`,
    `--user-data-dir=${userDataDir}`,
    '--headless=new',
    '--disable-gpu',
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-dev-shm-usage',
    'about:blank'
  ], { stdio: ['ignore', 'ignore', 'pipe'] })
  chrome.stderr.on('data', () => {})
  await waitForHTTP(`http://127.0.0.1:${remotePort}/json/version`, () => true)
}

async function newCDPTab(url) {
  const response = await fetch(`http://127.0.0.1:${remotePort}/json/new?${encodeURIComponent(url)}`, {
    method: 'PUT'
  })
  if (!response.ok) {
    throw new Error(`create chrome tab failed: ${response.status} ${await response.text()}`)
  }
  return response.json()
}

async function connectCDP(webSocketDebuggerUrl) {
  websocket = new WebSocket(webSocketDebuggerUrl)
  websocket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data)
    if (!message.id) return
    const callback = callbacks.get(message.id)
    if (!callback) return
    callbacks.delete(message.id)
    if (message.error) callback.reject(new Error(`${message.error.message}: ${message.error.data || ''}`))
    else callback.resolve(message.result || {})
  })
  await new Promise((resolve, reject) => {
    websocket.addEventListener('open', resolve, { once: true })
    websocket.addEventListener('error', reject, { once: true })
  })
}

function command(method, params = {}) {
  const id = nextMessageID++
  websocket.send(JSON.stringify({ id, method, params }))
  return new Promise((resolve, reject) => {
    callbacks.set(id, { resolve, reject })
    setTimeout(() => {
      if (!callbacks.has(id)) return
      callbacks.delete(id)
      reject(new Error(`${method} timed out`))
    }, 10000)
  })
}

async function navigate(url) {
  await command('Page.navigate', { url })
  await waitForPageReady()
}

async function waitForPageReady() {
  await waitUntil(async () => {
    const readyState = await evalPage('document.readyState')
    return readyState === 'complete' || readyState === 'interactive'
  }, 'page ready')
  await command('Runtime.evaluate', {
    expression: 'new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))',
    awaitPromise: true
  })
}

async function clearStorage() {
  await evalPage(`
    localStorage.clear();
    sessionStorage.clear();
    indexedDB.databases ? indexedDB.databases().then((databases) => Promise.all(databases.map((db) => indexedDB.deleteDatabase(db.name)))) : Promise.resolve();
  `)
}

async function clickByTestID(testID) {
  const clicked = await evalPage(`
    (() => {
      const element = document.querySelector('[data-testid="${escapeJS(testID)}"]');
      if (!element) return false;
      element.scrollIntoView({ block: 'center', inline: 'center' });
      element.click();
      return true;
    })()
  `)
  assert(clicked, `missing clickable target: ${testID}`)
  await command('Runtime.evaluate', {
    expression: 'new Promise((resolve) => setTimeout(resolve, 250))',
    awaitPromise: true
  })
}

async function setInputByTestID(testID, value) {
  const updated = await evalPage(`
    (() => {
      const host = document.querySelector('[data-testid="${escapeJS(testID)}"]');
      if (!host) return false;
      const input = host.matches('input, textarea') ? host : host.querySelector('input, textarea');
      if (!input) return false;
      input.scrollIntoView({ block: 'center', inline: 'center' });
      input.focus();
      input.value = ${JSON.stringify(value)};
      input.dispatchEvent(new Event('input', { bubbles: true }));
      input.dispatchEvent(new Event('change', { bubbles: true }));
      return true;
    })()
  `)
  assert(updated, `missing input target: ${testID}`)
}

async function waitForText(text) {
  await waitUntil(async () => {
    const visibleText = await evalPage('document.body ? document.body.innerText : ""')
    return String(visibleText).includes(text)
  }, `text ${text}`)
}

async function waitForHash(fragment) {
  await waitUntil(async () => {
    const hash = await evalPage('location.hash')
    return String(hash).includes(fragment)
  }, `hash ${fragment}`)
  await waitForPageReady()
}

async function assertReminderPersisted() {
  const session = await evalPage(`
    (() => {
      const raw = localStorage.getItem('littlelight_wechat_session');
      try {
        const parsed = raw ? JSON.parse(raw) : null;
        if (parsed && parsed.type === 'object' && parsed.data) return parsed.data;
        return parsed;
      } catch {
        return null;
      }
    })()
  `)
  assert(session?.sessionToken, 'H5 login did not persist a session token')
  const response = await fetch(`${apiURL}/api/v1/reminders?day=${todayISO()}`, {
    headers: { Authorization: `Bearer ${session.sessionToken}` }
  })
  const body = await response.text()
  assert(response.ok, `reminder list failed: ${response.status} ${body}`)
  const reminders = JSON.parse(body)
  assert(Array.isArray(reminders), 'reminder list did not return an array')
  const reminder = reminders.find((item) => item.title === reminderTitle)
  assert(reminder, `created reminder not found through API: ${reminderTitle}`)
  assert(reminder.status === 'pending', `created reminder status mismatch: ${reminder.status}`)
}

async function evalPage(expression) {
  const result = await command('Runtime.evaluate', {
    expression,
    awaitPromise: true,
    returnByValue: true
  })
  if (result.exceptionDetails) {
    throw new Error(`page evaluation failed: ${JSON.stringify(result.exceptionDetails)}`)
  }
  return result.result?.value
}

async function waitForHTTP(url, predicate) {
  let lastError = null
  for (let attempt = 0; attempt < 60; attempt += 1) {
    try {
      const response = await fetch(url)
      const text = await response.text()
      let data = text
      try {
        data = text ? JSON.parse(text) : null
      } catch {}
      if (response.ok && predicate(data)) return data
      lastError = new Error(`${url} returned ${response.status} ${text}`)
    } catch (error) {
      lastError = error
    }
    await delay(1000)
  }
  throw lastError || new Error(`${url} did not become ready`)
}

async function waitUntil(predicate, label) {
  let lastError = null
  for (let attempt = 0; attempt < 80; attempt += 1) {
    try {
      if (await predicate()) return
    } catch (error) {
      lastError = error
    }
    await delay(250)
  }
  throw new Error(`Timed out waiting for ${label}${lastError ? `: ${lastError.message}` : ''}`)
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function todayISO() {
  const date = new Date()
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function escapeJS(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

async function cleanup() {
  if (websocket) websocket.close()
  if (chrome) {
    chrome.kill('SIGTERM')
    await new Promise((resolve) => chrome.once('exit', resolve))
  }
  if (userDataDir) await rm(userDataDir, { recursive: true, force: true })
}

main()
  .catch((error) => {
    console.error(error)
    process.exitCode = 1
  })
  .finally(cleanup)
