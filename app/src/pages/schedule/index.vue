<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">课程 · 待办 · 提醒</text>
        <view class="title">今日安排</view>
      </view>
      <button class="bell" data-testid="schedule-quick-reminder-button" @tap="openReminderForm">+</button>
    </view>

    <view class="week-row">
      <view v-for="day in days" :key="day.iso" class="day" :class="{ active: selectedDay === day.iso }" @tap="selectDay(day)">
        <text>{{ day.label }}</text>
        <text>{{ day.date }}</text>
      </view>
    </view>

    <view class="section-head row between">
      <text class="section-title">本周日程</text>
      <view class="row action-row">
        <button class="ghost-btn small" @tap="downloadCourseTemplate">模板</button>
        <button class="ghost-btn small" :disabled="isActionPending('import-courses')" @tap="importSchedule">
          {{ isActionPending('import-courses') ? '预览中...' : '导入课表' }}
        </button>
      </view>
    </view>
    <view class="section-head row between compact">
      <text class="caption">当日课程</text>
      <button class="primary-btn small" data-testid="course-add-button" @tap="openCourseForm">添加日程</button>
    </view>
    <AppState v-if="loading" type="loading" message="正在加载日程..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />
    <view v-for="course in courses" :key="course.id" class="card">
      <view class="row between">
        <text class="tag">{{ course.startTime }} - {{ course.endTime }}</text>
        <view class="row action-row">
          <button class="ghost-btn mini" :disabled="hasPendingAction" @tap="editCourse(course)">编辑</button>
          <button class="ghost-btn mini danger" :disabled="isActionPending(`delete-course:${course.id}`)" @tap="removeCourse(course.id)">
            {{ isActionPending(`delete-course:${course.id}`) ? '删除中...' : '删除' }}
          </button>
        </view>
      </view>
      <text class="section-title">{{ course.className }} · {{ course.title }}</text>
      <text class="body">{{ course.note }} · {{ course.location }}</text>
    </view>
    <AppState
      v-if="!loading && !error && courses.length === 0"
      type="empty"
      title="当天还没有课程"
      message="添加日程或导入课表后，会同步到首页工作台。"
    />

    <view class="section-head row between">
      <text class="section-title">待办事项</text>
      <button class="primary-btn small" data-testid="reminder-add-button" @tap="openReminderForm">添加待办</button>
    </view>
    <view v-for="item in reminders" :key="item.id" class="card">
      <view class="row between">
        <text class="tag">{{ formatTime(item.remindAt) }} · {{ statusText(item.status) }}</text>
        <view class="row action-row">
          <button class="ghost-btn mini" :disabled="hasPendingAction" @tap="editReminder(item)">编辑</button>
          <button v-if="canSnoozeReminder(item)" class="ghost-btn mini" :disabled="hasPendingAction" @tap="openSnoozeForm(item)">延后</button>
          <button v-if="canCompleteReminder(item)" class="ghost-btn mini success" :disabled="isActionPending(`complete-reminder:${item.id}`)" @tap="complete(item.id)">
            {{ isActionPending(`complete-reminder:${item.id}`) ? '完成中...' : '完成' }}
          </button>
          <button class="ghost-btn mini danger" :disabled="isActionPending(`delete-reminder:${item.id}`)" @tap="removeReminder(item.id)">
            {{ isActionPending(`delete-reminder:${item.id}`) ? '删除中...' : '删除' }}
          </button>
        </view>
      </view>
      <text class="section-title">{{ item.title }}</text>
      <text class="body">{{ reminderDescription(item) }}</text>
    </view>
    <AppState
      v-if="!loading && !error && reminders.length === 0"
      type="empty"
      title="没有待办压着你"
      message="可以添加一个低打扰提醒，把后续跟进放到系统里。"
    />

    <view v-if="courseFormOpen || reminderFormOpen" class="modal-mask" @tap="closeForms">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">{{ courseFormOpen ? (editingCourseId ? '编辑课程' : '添加课程') : (editingReminderId ? '编辑待办' : '添加待办') }}</text>
          <button class="ghost-btn mini" @tap="closeForms">关闭</button>
        </view>

        <view v-if="courseFormOpen" class="form-grid">
          <input class="input" :class="{ invalid: courseFormError && !hasText(courseForm.title) }" v-model="courseForm.title" maxlength="80" placeholder="课程名称" aria-label="课程名称" data-testid="course-title-input" />
          <input class="input" :class="{ invalid: courseFormError && !hasText(courseForm.className) }" v-model="courseForm.className" maxlength="80" placeholder="班级" aria-label="课程班级" data-testid="course-class-input" />
          <input class="input" :class="{ invalid: courseFormError && !withinLength(courseForm.location, 120) }" v-model="courseForm.location" maxlength="120" placeholder="地点" aria-label="课程地点" data-testid="course-location-input" />
          <view class="row">
            <input class="input half" :class="{ invalid: courseFormError && !validClock(courseForm.startTime) }" v-model="courseForm.startTime" placeholder="开始 09:30" aria-label="课程开始时间" data-testid="course-start-input" />
            <input class="input half" :class="{ invalid: courseFormError && (!validClock(courseForm.endTime) || !clockBefore(courseForm.startTime, courseForm.endTime)) }" v-model="courseForm.endTime" placeholder="结束 10:15" aria-label="课程结束时间" data-testid="course-end-input" />
          </view>
          <textarea class="textarea short" :class="{ invalid: courseFormError && !withinLength(courseForm.note, 1000) }" v-model="courseForm.note" maxlength="1000" placeholder="备注" aria-label="课程备注" data-testid="course-note-input" />
          <text v-if="courseFormError" class="form-error">{{ courseFormError }}</text>
          <button class="primary-btn" :disabled="saving" data-testid="course-save-button" @tap="submitCourse">{{ saving ? '保存中...' : (editingCourseId ? '更新课程' : '保存课程') }}</button>
        </view>

        <view v-else class="form-grid">
          <input class="input" :class="{ invalid: reminderFormError && !hasText(reminderForm.title) }" v-model="reminderForm.title" maxlength="120" placeholder="待办标题" aria-label="待办标题" data-testid="reminder-title-input" />
          <input class="input" :class="{ invalid: reminderFormError && !validClock(reminderForm.time) }" v-model="reminderForm.time" placeholder="提醒时间 17:00" aria-label="待办提醒时间" data-testid="reminder-time-input" />
          <input class="input" :class="{ invalid: reminderFormError && !withinLength(reminderForm.category, 40) }" v-model="reminderForm.category" maxlength="40" placeholder="分类，例如 回访/个人事项" aria-label="待办分类" data-testid="reminder-category-input" />
          <textarea class="textarea short" :class="{ invalid: reminderFormError && !withinLength(reminderForm.note, 1000) }" v-model="reminderForm.note" maxlength="1000" placeholder="备注" aria-label="待办备注" data-testid="reminder-note-input" />
          <text v-if="reminderFormError" class="form-error">{{ reminderFormError }}</text>
          <button class="primary-btn" :disabled="saving" data-testid="reminder-save-button" @tap="submitReminder">{{ saving ? '保存中...' : (editingReminderId ? '更新待办' : '保存待办') }}</button>
        </view>
      </view>
    </view>

    <view v-if="snoozeFormOpen" class="modal-mask" @tap="closeSnoozeForm">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">延后待办</text>
          <button class="ghost-btn mini" @tap="closeSnoozeForm">关闭</button>
        </view>
        <view class="form-grid">
          <text class="body snooze-summary">{{ snoozeForm.title || '选择新的提醒时间' }}</text>
          <view class="row">
            <input class="input half" :class="{ invalid: snoozeFormError && !validISODate(snoozeForm.date) }" v-model="snoozeForm.date" placeholder="日期 2026-05-30" aria-label="延后日期" data-testid="snooze-date-input" />
            <input class="input half" :class="{ invalid: snoozeFormError && !validClock(snoozeForm.time) }" v-model="snoozeForm.time" placeholder="时间 17:30" aria-label="延后时间" data-testid="snooze-time-input" />
          </view>
          <text v-if="snoozeFormError" class="form-error">{{ snoozeFormError }}</text>
          <button class="primary-btn" :disabled="saving" @tap="submitSnooze">{{ saving ? '保存中...' : '确认延后' }}</button>
        </view>
      </view>
    </view>

    <view v-if="importDialogOpen" class="modal-mask" @tap="closeImportDialog">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">课表导入预览</text>
          <button class="ghost-btn mini" @tap="closeImportDialog">关闭</button>
        </view>
        <text class="body">{{ importSummary }}</text>
        <view v-for="item in importPreview" :key="item.row" class="preview-row" :class="{ invalid: item.status === 'invalid' }">
          <text class="tag">第 {{ item.row }} 行 · {{ item.status === 'ready' ? '可导入' : '需修正' }}</text>
          <text class="body">{{ previewText(item) }}</text>
          <text v-if="item.message" class="caption block">{{ item.message }}</text>
        </view>
        <view v-if="importErrors.length" class="state-card compact-state">
          <text class="section-title">失败提示</text>
          <text v-for="error in importErrors" :key="error" class="caption block">{{ error }}</text>
          <button v-if="importFailureCsv" class="ghost-btn small" @tap="copyFailureCsv">复制失败行 CSV</button>
        </view>
        <button class="primary-btn" :disabled="saving || isActionPending('confirm-import-courses') || !importReadyCount" @tap="confirmImportSchedule">
          {{ isActionPending('confirm-import-courses') ? '导入中...' : `确认导入 ${importReadyCount} 条` }}
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { chooseImportFile, clockBefore, confirmAction, ensureLoggedIn, errorMessage, hasText, showToast, trimmed, validClock, withinLength } from '../../utils/ui'
import AppState from '../../components/AppState.vue'

const today = new Date()
const selectedDay = ref(toISODate(today))
const weekday = ref(today.getDay())
const courses = ref([])
const reminders = ref([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const courseFormOpen = ref(false)
const reminderFormOpen = ref(false)
const snoozeFormOpen = ref(false)
const editingCourseId = ref('')
const editingReminderId = ref('')
const snoozeReminderId = ref('')
const courseForm = ref(defaultCourseForm())
const reminderForm = ref(defaultReminderForm())
const snoozeForm = ref(defaultSnoozeForm())
const courseFormError = ref('')
const reminderFormError = ref('')
const snoozeFormError = ref('')
const days = buildWeek(today)
const importDialogOpen = ref(false)
const importFile = ref(null)
const importPreview = ref([])
const importErrors = ref([])
const importFailureCsv = ref('')
const importReadyCount = ref(0)
const importSummary = ref('')
const pendingAction = ref('')
const hasPendingAction = computed(() => !!pendingAction.value)

onShow(load)

async function load() {
  if (!ensureLoggedIn(api)) return
  loading.value = true
  error.value = ''
  try {
    courses.value = await api.courses(weekday.value)
    reminders.value = await api.reminders(selectedDay.value)
  } catch (err) {
    error.value = errorMessage(err, '日程加载失败')
  } finally {
    loading.value = false
  }
}

async function selectDay(day) {
  selectedDay.value = day.iso
  weekday.value = day.weekday
  await load()
}

function openCourseForm() {
  courseForm.value = defaultCourseForm()
  courseFormError.value = ''
  reminderFormError.value = ''
  closeSnoozeForm()
  editingCourseId.value = ''
  courseFormOpen.value = true
  reminderFormOpen.value = false
}

function openReminderForm() {
  reminderForm.value = defaultReminderForm()
  courseFormError.value = ''
  reminderFormError.value = ''
  closeSnoozeForm()
  editingReminderId.value = ''
  reminderFormOpen.value = true
  courseFormOpen.value = false
}

function closeForms() {
  courseFormOpen.value = false
  reminderFormOpen.value = false
  editingCourseId.value = ''
  editingReminderId.value = ''
  courseFormError.value = ''
  reminderFormError.value = ''
}

function openSnoozeForm(item) {
  closeForms()
  snoozeReminderId.value = item.id
  snoozeForm.value = defaultSnoozeForm(item)
  snoozeFormError.value = ''
  snoozeFormOpen.value = true
}

function closeSnoozeForm() {
  snoozeFormOpen.value = false
  snoozeReminderId.value = ''
  snoozeFormError.value = ''
}

function editCourse(course) {
  editingCourseId.value = course.id
  courseForm.value = {
    title: course.title || '',
    className: course.className || '',
    location: course.location || '',
    startTime: course.startTime || '10:00',
    endTime: course.endTime || '10:45',
    note: course.note || ''
  }
  courseFormError.value = ''
  reminderFormError.value = ''
  courseFormOpen.value = true
  reminderFormOpen.value = false
}

function editReminder(item) {
  editingReminderId.value = item.id
  reminderForm.value = {
    title: item.title || '',
    category: item.category || '个人事项',
    time: timeInputValue(item.remindAt) || '17:00',
    note: item.note || '',
    status: item.status || 'pending'
  }
  courseFormError.value = ''
  reminderFormError.value = ''
  reminderFormOpen.value = true
  courseFormOpen.value = false
}

async function submitCourse() {
  if (saving.value) return
  const validation = validateCourseForm()
  if (validation) {
    courseFormError.value = validation
    return
  }
  saving.value = true
  courseFormError.value = ''
  try {
    const payload = {
      title: trimmed(courseForm.value.title),
      className: trimmed(courseForm.value.className),
      location: trimmed(courseForm.value.location),
      startTime: trimmed(courseForm.value.startTime),
      endTime: trimmed(courseForm.value.endTime),
      note: trimmed(courseForm.value.note),
      weekday: weekday.value
    }
    const item = editingCourseId.value
      ? await api.updateCourse(editingCourseId.value, payload)
      : await api.createCourse(payload)
    courses.value = editingCourseId.value
      ? courses.value.map((course) => course.id === item.id ? item : course)
      : [...courses.value, item]
    const wasEditing = !!editingCourseId.value
    closeForms()
    showToast(wasEditing ? '已更新课程' : '已添加到当日课程')
  } catch (err) {
    courseFormError.value = errorMessage(err, '课程保存失败')
    showToast(courseFormError.value)
  } finally {
    saving.value = false
  }
}

async function removeCourse(id) {
  const action = `delete-course:${id}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmAction({
    title: '删除课程',
    content: '删除后该课程会从当天安排中移除，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  try {
    await api.deleteCourse(id)
    showToast('已删除课程')
    await load()
  } catch (err) {
    showToast(errorMessage(err, '课程删除失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function submitReminder() {
  if (saving.value) return
  const validation = validateReminderForm()
  if (validation) {
    reminderFormError.value = validation
    return
  }
  saving.value = true
  reminderFormError.value = ''
  try {
    const payload = cleanReminderPayload(reminderForm.value, !!editingReminderId.value)
    const item = editingReminderId.value
      ? await api.updateReminder(editingReminderId.value, payload)
      : await api.createReminder(payload)
    reminders.value = editingReminderId.value
      ? reminders.value.map((reminder) => reminder.id === item.id ? item : reminder)
      : [item, ...reminders.value]
    const wasEditing = !!editingReminderId.value
    closeForms()
    showToast(wasEditing ? '已更新待办' : '已添加待办')
  } catch (err) {
    reminderFormError.value = errorMessage(err, '待办保存失败')
    showToast(reminderFormError.value)
  } finally {
    saving.value = false
  }
}

function cleanReminderPayload(form, includeStatus = false) {
  const remindAt = new Date(`${selectedDay.value}T${trimmed(form.time)}:00`)
  const payload = {
    title: trimmed(form.title),
    category: trimmed(form.category) || '个人事项',
    note: trimmed(form.note),
    remindAt: remindAt.toISOString()
  }
  if (includeStatus) {
    payload.status = form.status || 'pending'
  }
  return payload
}

async function complete(id) {
  const action = `complete-reminder:${id}`
  if (isActionPending(action)) return
  pendingAction.value = action
  try {
    await api.completeReminder(id)
    showToast('已完成提醒')
    await load()
  } catch (err) {
    showToast(errorMessage(err, '提醒更新失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function submitSnooze() {
  if (saving.value) return
  const validation = validateSnoozeForm()
  if (validation) {
    snoozeFormError.value = validation
    return
  }
  saving.value = true
  snoozeFormError.value = ''
  try {
    const until = snoozeUntilDate().toISOString()
    await api.snoozeReminder(snoozeReminderId.value, until)
    const message = `${snoozeForm.value.date} ${snoozeForm.value.time}`
    closeSnoozeForm()
    showToast(`已延后到 ${message}`)
    await load()
  } catch (err) {
    snoozeFormError.value = errorMessage(err, '延后失败')
    showToast(snoozeFormError.value)
  } finally {
    saving.value = false
  }
}

async function removeReminder(id) {
  const action = `delete-reminder:${id}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmAction({
    title: '删除待办',
    content: '删除后该提醒不会再出现，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  try {
    await api.deleteReminder(id)
    showToast('已删除提醒')
    await load()
  } catch (err) {
    showToast(errorMessage(err, '删除提醒失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function importSchedule() {
  const action = 'import-courses'
  if (isActionPending(action)) return
  pendingAction.value = action
  try {
    const uploadFile = await chooseImportFile()
    importFile.value = uploadFile
    const result = await api.importCourses(uploadFile, { preview: true })
    showImportPreview(result, '课表')
  } catch (err) {
    showToast(errorMessage(err, '课表预览失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function confirmImportSchedule() {
  const action = 'confirm-import-courses'
  if (saving.value || isActionPending(action) || !importFile.value) return
  saving.value = true
  pendingAction.value = action
  try {
    const result = await api.importCourses(importFile.value)
    showImportPreview(result, '课表')
    showToast(`已导入 ${result.imported || 0} 条课表，跳过 ${result.skipped || 0} 条`)
    await load()
  } catch (err) {
    showToast(errorMessage(err, '课表导入失败'))
  } finally {
    saving.value = false
    clearPendingAction(action)
  }
}

function isActionPending(action) {
  return pendingAction.value === action
}

function clearPendingAction(action) {
  if (pendingAction.value === action) pendingAction.value = ''
}

function showImportPreview(result, label) {
  importPreview.value = result.preview || []
  importErrors.value = result.errors || []
  importFailureCsv.value = result.failureCsv || ''
  importReadyCount.value = importPreview.value.filter((item) => item.status === 'ready').length
  importSummary.value = `${label}文件中 ${importReadyCount.value} 条可导入，${result.skipped || 0} 条需跳过。`
  importDialogOpen.value = true
}

function closeImportDialog() {
  importDialogOpen.value = false
}

function previewText(item) {
  const values = item.values || {}
  return values['课程名称'] || values['课程'] || values.title || values.course || values['班级'] || values.class || '已读取该行'
}

function copyFailureCsv() {
  uni.setClipboardData({ data: importFailureCsv.value })
  showToast('已复制失败行 CSV')
}

function downloadCourseTemplate() {
  downloadTemplate(api.courseImportTemplateURL(), '课程模板字段：课程名称、班级、地点、星期、开始时间、结束时间、备注')
}

function downloadTemplate(url, fallbackText) {
  // #ifdef H5
  const link = document.createElement('a')
  link.href = url
  link.download = ''
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  showToast('已下载导入模板')
  // #endif
  // #ifndef H5
  uni.setClipboardData({ data: fallbackText })
  showToast('已复制模板字段')
  // #endif
}

function formatTime(value) {
  return new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function timeInputValue(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function statusText(status) {
  return ({ pending: '待处理', done: '已完成', snoozed: '已延后' })[status] || '待处理'
}

function canCompleteReminder(item) {
  return item?.status !== 'done'
}

function canSnoozeReminder(item) {
  return item?.status !== 'done'
}

function reminderDescription(item) {
  if (item?.status === 'done' && item.doneAt) {
    return `${item.note || '已完成'} · 完成于 ${formatTime(item.doneAt)}`
  }
  return item?.note || '暂无备注'
}

function defaultCourseForm() {
  return { title: '', className: '', location: '', startTime: '10:00', endTime: '10:45', note: '' }
}

function defaultReminderForm() {
  return { title: '', category: '个人事项', time: '17:00', note: '' }
}

function defaultSnoozeForm(item = null) {
  const date = new Date(Date.now() + 30 * 60 * 1000)
  return { title: item?.title || '', date: toISODate(date), time: timeInputValue(date) || '17:30' }
}

function validateCourseForm() {
  const form = courseForm.value
  if (!hasText(form.title) || !hasText(form.className)) return '请填写课程名称和班级'
  if (!validClock(form.startTime) || !validClock(form.endTime)) return '开始和结束时间请使用 09:30 这样的 24 小时格式'
  if (!clockBefore(form.startTime, form.endTime)) return '结束时间需要晚于开始时间'
  if (!withinLength(form.title, 80) || !withinLength(form.className, 80)) return '课程名称和班级最多 80 个字'
  if (!withinLength(form.location, 120)) return '地点最多 120 个字'
  if (!withinLength(form.note, 1000)) return '备注最多 1000 个字'
  return ''
}

function validateReminderForm() {
  const form = reminderForm.value
  if (!hasText(form.title)) return '请填写待办标题'
  if (!validClock(form.time)) return '提醒时间请使用 17:00 这样的 24 小时格式'
  if (!withinLength(form.title, 120)) return '待办标题最多 120 个字'
  if (!withinLength(form.category, 40)) return '分类最多 40 个字'
  if (!withinLength(form.note, 1000)) return '备注最多 1000 个字'
  return ''
}

function validateSnoozeForm() {
  const form = snoozeForm.value
  if (!snoozeReminderId.value) return '未找到要延后的待办'
  if (!validISODate(form.date)) return '延后日期请使用 2026-05-30 这样的格式'
  if (!validClock(form.time)) return '延后时间请使用 17:30 这样的 24 小时格式'
  const until = snoozeUntilDate()
  if (Number.isNaN(until.getTime())) return '延后时间无效'
  if (until.getTime() <= Date.now()) return '延后时间需要晚于当前时间'
  return ''
}

function snoozeUntilDate() {
  return new Date(`${trimmed(snoozeForm.value.date)}T${trimmed(snoozeForm.value.time)}:00`)
}

function buildWeek(base) {
  const labels = ['日', '一', '二', '三', '四', '五', '六']
  const start = new Date(base)
  start.setDate(base.getDate() - base.getDay())
  return labels.map((label, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    return { label, date: String(date.getDate()).padStart(2, '0'), weekday: index, iso: toISODate(date) }
  })
}

function toISODate(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function validISODate(value) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(trimmed(value))) return false
  const date = new Date(`${trimmed(value)}T00:00:00`)
  return !Number.isNaN(date.getTime()) && toISODate(date) === trimmed(value)
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header, .section-head { padding: 0 32rpx 12rpx; }
.compact { padding-bottom: 0; }
.bell { width: 88rpx; height: 88rpx; border-radius: 28rpx; background: rgba(255,255,255,.86); font-size: 42rpx; color: #0e7490; }
.week-row { display: flex; gap: 16rpx; padding: 16rpx 32rpx; overflow-x: auto; }
.day { min-width: 88rpx; height: 116rpx; border-radius: 28rpx; background: rgba(255,255,255,.76); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10rpx; color: #506075; font-weight: 900; }
.day.active { color: #fff; background: linear-gradient(145deg,#0891b2,#059669); }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
.success { color: #059669; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
.state-card { display: flex; flex-direction: column; gap: 18rpx; }
.compact-state { margin: 0; padding: 22rpx; }
.preview-row { padding: 22rpx; border-radius: 22rpx; background: rgba(236,254,255,.72); display: flex; flex-direction: column; gap: 8rpx; }
.preview-row.invalid { background: rgba(255,241,242,.88); }
.block { display: block; }
.retry { align-self: flex-start; }
.snooze-summary { padding: 18rpx 20rpx; border-radius: 20rpx; background: rgba(236,254,255,.72); }
.modal-mask { position: fixed; inset: 0; z-index: 20; background: rgba(18,32,56,.34); display: flex; align-items: flex-end; padding: 32rpx; box-sizing: border-box; }
.modal-card { width: 100%; padding: 32rpx; border-radius: 32rpx; background: #fff; box-shadow: 0 -20rpx 60rpx rgba(24,32,51,.16); display: flex; flex-direction: column; gap: 24rpx; }
.form-grid { display: flex; flex-direction: column; gap: 18rpx; }
.half { flex: 1; min-width: 0; }
.textarea.short { min-height: 120rpx; }
</style>
