import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const pagesJSON = JSON.parse(await read('src/pages.json'))
const packageJSON = JSON.parse(await read('package.json'))

await testH5SmokeScriptIsExposed()
await testLoginIsFirstPage()
await testLoginUsesRealWechatFlowWithDevFallback()
await testProtectedPagesUseStateAndAuthGuard()
await testRealCRUDFormsAreStillWired()
await testCoreFormsExposeStableFieldTargets()
await testH5SmokeTargetsStayStable()
await testLongTextControlsMatchBackendLimits()
await testUniAppInputsKeepFocusableBox()
await testReminderCreatePayloadOmitsServerManagedStatus()
await testCommunicationRecordPayloadOmitsUIOnlyFields()
await testHighTrafficActionsPreventDuplicateSubmits()
await testImportPreviewAndConfirmFlowsAreWired()
await testAIActionsKeepReviewGate()
await testProfileExposesMockMembershipCheckout()
await testProfileExposesAccountDeletion()
await testProfileExposesAccountExport()
await testVoiceDictationSupportsH5AndApp()

console.log('page contract tests passed')

async function testH5SmokeScriptIsExposed() {
  assert.equal(packageJSON.scripts?.['test:h5-smoke'], 'node scripts/h5-smoke.mjs', 'package.json must expose the H5 browser smoke test')
}

async function testLoginIsFirstPage() {
  assert.equal(pagesJSON.pages?.[0]?.path, 'pages/login/index', 'login page must be first for login-first startup')
  const tabPages = new Set((pagesJSON.tabBar?.list || []).map((item) => item.pagePath))
  assert.equal(tabPages.has('pages/login/index'), false, 'login page must not be a tab page')
}

async function testLoginUsesRealWechatFlowWithDevFallback() {
  const source = await read('src/pages/login/index.vue')
  assertIncludes(source, 'api.wechatLogin(', 'login page must call the real WeChat auth endpoint')
  assertIncludes(source, 'api.wechatMockLogin(', 'login page must keep explicit development login for local debugging')
  assertIncludes(source, 'api.isDevAuthAvailable()', 'development login must be gated by API/env config')
  assertIncludes(source, 'provider: \'weixin\'', 'login page must request the WeChat provider')
  assertIncludes(source, 'canRequestWechatLogin()', 'login page must separate MP-WEIXIN auth capability from H5 fallback')
  assertIncludes(source, 'api.isLoggedIn()', 'login page must skip to the app when a valid session exists')
}

async function testProtectedPagesUseStateAndAuthGuard() {
  const protectedPages = [
    'src/pages/home/index.vue',
    'src/pages/schedule/index.vue',
    'src/pages/communication/index.vue',
    'src/pages/communication/parent-detail.vue',
    'src/pages/heal/index.vue',
    'src/pages/profile/index.vue',
    'src/pages/profile/pro-benefits.vue',
    'src/pages/profile/pro-pay.vue',
    'src/pages/profile/ai-log-detail.vue'
  ]
  for (const page of protectedPages) {
    const source = await read(page)
    assertIncludes(source, 'AppState', `${page} must render unified loading/error/empty states`)
    assertIncludes(source, 'ensureLoggedIn(api)', `${page} must guard authenticated data loading`)
    assert.match(source, /error\s*=\s*ref\(/, `${page} must track a page-level error state`)
  }
}

async function testRealCRUDFormsAreStillWired() {
  const expectations = {
    'src/pages/schedule/index.vue': [
      'api.createCourse(',
      'api.updateCourse(',
      'api.createReminder(',
      'api.updateReminder(',
      'validateCourseForm()',
      'validateReminderForm()'
    ],
    'src/pages/communication/index.vue': [
      'api.createParent(',
      'api.updateParent(',
      'api.createRecord(',
      'api.updateRecord(',
      'validateParentForm(',
      'validateRecordForm('
    ],
    'src/pages/communication/parent-detail.vue': [
      'api.updateParent(',
      'api.createRecord(',
      'api.updateRecord(',
      'validateParentForm(',
      'validateRecordForm('
    ],
    'src/pages/profile/index.vue': [
      'api.updateMe(',
      'api.createFavorite(',
      'validateProfileForm(',
      'validateFavoriteForm('
    ]
  }
  for (const [page, needles] of Object.entries(expectations)) {
    const source = await read(page)
    for (const needle of needles) {
      assertIncludes(source, needle, `${page} must keep real form flow: ${needle}`)
    }
    assertIncludes(source, ':disabled="saving"', `${page} must disable submit buttons while saving`)
  }
}

async function testCoreFormsExposeStableFieldTargets() {
  const expectations = {
    'src/pages/login/index.vue': [
      'data-testid="login-nickname-input"'
    ],
    'src/pages/schedule/index.vue': [
      'data-testid="course-title-input"',
      'data-testid="course-class-input"',
      'data-testid="course-start-input"',
      'data-testid="course-end-input"',
      'data-testid="reminder-title-input"',
      'data-testid="reminder-time-input"'
    ],
    'src/pages/communication/index.vue': [
      'data-testid="parent-draft-issue-input"',
      'data-testid="parent-student-input"',
      'data-testid="parent-name-input"',
      'data-testid="record-student-input"',
      'data-testid="record-summary-input"',
      'data-testid="record-follow-up-date-input"'
    ],
    'src/pages/communication/parent-detail.vue': [
      'data-testid="detail-parent-student-input"',
      'data-testid="detail-parent-name-input"',
      'data-testid="detail-record-summary-input"',
      'data-testid="detail-record-follow-up-date-input"'
    ],
    'src/pages/profile/index.vue': [
      'data-testid="profile-name-input"',
      'data-testid="profile-school-input"',
      'data-testid="favorite-title-input"',
      'data-testid="favorite-content-input"'
    ],
    'src/pages/heal/index.vue': [
      'data-testid="praise-content-input"',
      'data-testid="healing-search-input"'
    ]
  }
  for (const [page, needles] of Object.entries(expectations)) {
    const source = await read(page)
    for (const needle of needles) {
      assertIncludes(source, needle, `${page} must expose stable field targets for runtime and accessibility checks: ${needle}`)
    }
    assertIncludes(source, 'aria-label=', `${page} must name important input fields for assistive technology`)
  }
}

async function testH5SmokeTargetsStayStable() {
  const expectations = {
    'src/pages/login/index.vue': [
      'data-testid="login-nickname-input"',
      'data-testid="wechat-login-button"'
    ],
    'src/pages/home/index.vue': [
      'data-testid="home-schedule-button"',
      'data-testid="home-today-schedule-button"',
      'data-testid="home-manage-schedule-button"'
    ],
    'src/pages/schedule/index.vue': [
      'data-testid="reminder-add-button"',
      'data-testid="reminder-save-button"',
      'data-testid="reminder-title-input"',
      'data-testid="reminder-category-input"',
      'data-testid="reminder-note-input"'
    ]
  }
  for (const [page, needles] of Object.entries(expectations)) {
    const source = await read(page)
    for (const needle of needles) {
      assertIncludes(source, needle, `${page} must keep stable H5 smoke target: ${needle}`)
    }
  }
}

async function testLongTextControlsMatchBackendLimits() {
  const expectations = {
    'src/pages/schedule/index.vue': [
      'data-testid="course-note-input"',
      'maxlength="1000"',
      'data-testid="reminder-note-input"'
    ],
    'src/pages/communication/index.vue': [
      'data-testid="parent-draft-issue-input"',
      'maxlength="1200"',
      'data-testid="parent-notes-input"',
      'maxlength="2000"',
      'data-testid="parent-next-action-input"',
      'maxlength="1000"',
      'data-testid="record-summary-input"',
      'data-testid="record-result-input"'
    ],
    'src/pages/communication/parent-detail.vue': [
      'data-testid="detail-parent-notes-input"',
      'maxlength="2000"',
      'data-testid="detail-parent-next-action-input"',
      'maxlength="1000"',
      'data-testid="detail-record-summary-input"',
      'data-testid="detail-record-result-input"'
    ],
    'src/pages/profile/index.vue': [
      'data-testid="favorite-content-input"',
      'maxlength="2000"'
    ],
    'src/pages/heal/index.vue': [
      'data-testid="praise-content-input"',
      'maxlength="1000"'
    ]
  }
  for (const [page, needles] of Object.entries(expectations)) {
    const source = await read(page)
    for (const needle of needles) {
      assertIncludes(source, needle, `${page} must keep control maxlength aligned with backend validation: ${needle}`)
    }
  }
}

async function testUniAppInputsKeepFocusableBox() {
  const source = await read('src/static/common.css')
  assertIncludes(source, '.input { min-height:', 'shared input styling must keep a visible focusable field box')
  assertIncludes(source, '.input .uni-input-wrapper', 'uni-app H5 input wrapper must receive an explicit height')
  assertIncludes(source, '.input .uni-input-input', 'uni-app H5 native input must receive an explicit height')
  assertIncludes(source, '.textarea .uni-textarea-textarea', 'uni-app H5 textarea must receive an explicit height')
}

async function testReminderCreatePayloadOmitsServerManagedStatus() {
  const source = await read('src/pages/schedule/index.vue')
  const start = source.indexOf('function cleanReminderPayload(form, includeStatus = false)')
  assert.notEqual(start, -1, 'schedule page must centralize reminder payload cleaning')
  const end = source.indexOf('async function complete', start)
  const body = source.slice(start, end)
  assertIncludes(body, 'if (includeStatus)', 'reminder payload must only include status when updating')
  assertIncludes(body, 'payload.status = form.status || \'pending\'', 'reminder update payload must preserve status')
  assert.doesNotMatch(body, /status:\s*form\.status/, 'reminder create payload must not send status because backend owns initial status')
}

async function testCommunicationRecordPayloadOmitsUIOnlyFields() {
  const source = await read('src/pages/communication/index.vue')
  const start = source.indexOf('function cleanRecordPayload(form)')
  assert.notEqual(start, -1, 'communication page must centralize record payload cleaning')
  const end = source.indexOf('function recordFollowUpISO', start)
  const body = source.slice(start, end)
  assert.doesNotMatch(body, /\.\.\.form/, 'record payload must not spread UI-only followUpDate/followUpTime fields')
  assertIncludes(body, 'parentId: form.parentId || null', 'record payload must preserve selected parent linkage')
  assertIncludes(body, 'riskLevel: form.riskLevel || \'low\'', 'record payload must preserve risk level without spreading form internals')
}

async function testImportPreviewAndConfirmFlowsAreWired() {
  const expectations = {
    'src/pages/schedule/index.vue': ['chooseImportFile()', 'api.importCourses(uploadFile, { preview: true })', 'confirmImportSchedule', 'failureCsv'],
    'src/pages/communication/index.vue': ['chooseImportFile()', 'api.importParents(uploadFile, { preview: true })', 'confirmImportParents', 'failureCsv']
  }
  for (const [page, needles] of Object.entries(expectations)) {
    const source = await read(page)
    for (const needle of needles) {
      assertIncludes(source, needle, `${page} must keep import preview/confirm/failure feedback: ${needle}`)
    }
  }
}

async function testHighTrafficActionsPreventDuplicateSubmits() {
  const expectations = {
    'src/pages/schedule/index.vue': [
      'const pendingAction = ref',
      'isActionPending(`delete-course:${course.id}`)',
      'isActionPending(`complete-reminder:${item.id}`)',
      'isActionPending(`delete-reminder:${item.id}`)',
      "isActionPending('import-courses')",
      "isActionPending('confirm-import-courses')"
    ],
    'src/pages/communication/index.vue': [
      'const pendingAction = ref',
      'isActionPending(`save-draft:${draft.id}`)',
      'isActionPending(`copy-draft:${draft.id}`)',
      'isActionPending(`delete-parent:${parent.id}`)',
      'isActionPending(`complete-record:${record.id}`)',
      'isActionPending(`delete-record:${record.id}`)',
      "isActionPending('import-parents')",
      "isActionPending('confirm-import-parents')"
    ]
  }
  for (const [page, needles] of Object.entries(expectations)) {
    const source = await read(page)
    for (const needle of needles) {
      assertIncludes(source, needle, `${page} must prevent duplicate submits for high-traffic actions: ${needle}`)
    }
  }
}

async function testAIActionsKeepReviewGate() {
  const communication = await read('src/pages/communication/index.vue')
  assertIncludes(communication, 'review_confirmed', 'communication AI drafts must write review confirmation actions')
  assertIncludes(communication, 'review_skipped', 'communication AI drafts must write skipped review actions')
  assertIncludes(communication, 'save_record', 'communication AI drafts must audit save_record usage')

  const healing = await read('src/pages/heal/index.vue')
  assertIncludes(healing, 'review_confirmed', 'healing AI praise must write review confirmation actions')
  assertIncludes(healing, 'save_healing', 'healing AI praise must audit save_healing usage')

  const detail = await read('src/pages/profile/ai-log-detail.vue')
  assertIncludes(detail, 'confirmDraftReview', 'AI log detail must require review before copying risky output')
  assertIncludes(detail, 'review_confirmed', 'AI log detail must persist review confirmations')
  assertIncludes(detail, 'api.deleteAIGeneration(', 'AI log detail must let users delete sensitive AI records')

  const profile = await read('src/pages/profile/index.vue')
  assertIncludes(profile, 'api.deleteAIGeneration(', 'profile AI log list must let users delete sensitive AI records')
}

async function testProfileExposesMockMembershipCheckout() {
  const source = await read('src/pages/profile/index.vue')
  assertIncludes(source, 'data-testid="profile-member-card"', 'profile page must show membership directly below personal info')
  assertIncludes(source, 'openBenefitsPage()', 'profile page must navigate to the membership benefits page')
  assertIncludes(source, 'pro-badge', 'profile page must render a dedicated Pro member badge')
  assertIncludes(source, '支付：模拟微信支付', 'profile page must clearly label the current mock payment flow')

  assert.equal(
    pagesJSON.pages.some((page) => page.path === 'pages/profile/pro-benefits'),
    true,
    'pages.json must register the Pro benefits page'
  )
  assert.equal(
    pagesJSON.pages.some((page) => page.path === 'pages/profile/pro-pay'),
    true,
    'pages.json must register the Pro payment page'
  )

  const benefits = await read('src/pages/profile/pro-benefits.vue')
  assertIncludes(benefits, 'price: 20', 'Pro benefits page must show the 20 yuan monthly plan')
  assertIncludes(benefits, 'price: 200', 'Pro benefits page must show the 200 yuan yearly plan')
  assertIncludes(benefits, '模拟微信支付', 'Pro benefits page must label the payment as simulated')
  assertIncludes(benefits, 'data-testid="pro-open-pay-button"', 'Pro benefits page must expose a stable pay button target')
  assertIncludes(benefits, '/pages/profile/pro-pay?plan=', 'Pro benefits page must navigate into payment with the selected plan')

  const pay = await read('src/pages/profile/pro-pay.vue')
  assertIncludes(pay, 'api.checkout({', 'Pro payment page must call the checkout endpoint')
  assertIncludes(pay, "provider: 'wechat'", 'Pro payment page must use WeChat as the payment provider')
  assertIncludes(pay, 'mock: true', 'Pro payment page must explicitly mark simulated payment')
  assertIncludes(pay, 'data-testid="mock-wechat-pay-button"', 'Pro payment page must expose a stable mock WeChat pay target')
  assertIncludes(pay, "uni.switchTab({ url: '/pages/profile/index' })", 'Pro payment page must return to profile after success')

  assertIncludes(source, '推送：{{ notificationReady ? \'已接入\' : \'未接入\' }}', 'profile page must honestly show notification provider status')
  assertIncludes(source, '对象存储：{{ objectStorageReady ? \'已接入\' : \'未接入\' }}', 'profile page must honestly show object storage status')
}

async function testProfileExposesAccountDeletion() {
  const source = await read('src/pages/profile/index.vue')
  assertIncludes(source, 'api.deleteMe()', 'profile page must call the account deletion endpoint')
  assertIncludes(source, '删除账号与数据', 'profile page must clearly label account deletion')
  assertIncludes(source, 'confirmAction({', 'profile page must confirm before deleting account data')
  assertIncludes(source, "uni.reLaunch({ url: '/pages/login/index' })", 'profile page must return to login after account deletion')
}

async function testProfileExposesAccountExport() {
  const source = await read('src/pages/profile/index.vue')
  assertIncludes(source, 'api.exportMe()', 'profile page must call the account export endpoint')
  assertIncludes(source, '导出数据', 'profile page must expose account data export')
  assertIncludes(source, 'JSON.stringify(data, null, 2)', 'profile export must serialize a readable JSON export')
  assertIncludes(source, 'new Blob([text]', 'H5 profile export must download JSON as a file')
  assertIncludes(source, 'uni.setClipboardData({ data: text })', 'non-H5 profile export must preserve JSON through clipboard fallback')
}

async function testVoiceDictationSupportsH5AndApp() {
  const app = await read('src/App.vue')
  assertIncludes(app, '<VoiceDictation />', 'the global app shell must render realtime voice dictation')

  const component = await read('src/components/VoiceDictation.vue')
  assertIncludes(component, 'createDictationSocket(', 'voice dictation must use the cross-platform WebSocket factory')
  assertIncludes(component, 'createPCMRecorder({', 'voice dictation must use the cross-platform PCM recorder factory')
  assertIncludes(component, 'duration: 280', 'App dictation must use short chunks for near realtime feedback')

  const util = await read('src/utils/dictation.js')
  assertIncludes(util, 'class H5PCMRecorder', 'H5 voice dictation must keep Web Audio PCM capture')
  assertIncludes(util, 'class UniAppPCMChunkRecorder', 'App voice dictation must keep a native recorder implementation')
  assertIncludes(util, 'uni.connectSocket({ url', 'App voice dictation must use uni.connectSocket instead of browser-only WebSocket')
  assertIncludes(util, 'uni.getRecorderManager()', 'App voice dictation must use the native RecorderManager')
  assertIncludes(util, "format: 'PCM'", 'App recorder must request raw PCM output')
  assertIncludes(util, 'frameBytes = 1280', 'App PCM chunks must be split to Xfyun-sized 40ms frames')
  assertIncludes(util, 'readPlusFileAsArrayBuffer', 'App recorder must keep an HTML5+ file read fallback')

  const manifest = await read('src/manifest.json')
  assertIncludes(manifest, 'android.permission.RECORD_AUDIO', 'App package must request Android microphone permission')
  assertIncludes(manifest, 'NSMicrophoneUsageDescription', 'App package must explain iOS microphone usage')

  const envExample = await read('.env.example')
  assertIncludes(envExample, 'VITE_APP_API_BASE_URL=https://api.example.com/api/v1', 'App builds must document an absolute API base URL for WebSocket support')
}

async function read(path) {
  return readFile(new URL(`../${path}`, import.meta.url), 'utf8')
}

function assertIncludes(source, needle, message) {
  assert.equal(source.includes(needle), true, message)
}
