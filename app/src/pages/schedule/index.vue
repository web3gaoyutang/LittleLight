<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">课程 · 待办 · 提醒</text>
        <view class="title">今日安排</view>
      </view>
      <button class="avatar-btn" data-testid="schedule-quick-reminder-button" @tap="openReminderForm">
        <text class="icon-chip"><AppIcon name="plus" /></text>
      </button>
    </view>

    <view class="week-row">
      <view
        v-for="day in days"
        :key="day.iso"
        class="day"
        :class="{ active: selectedDay === day.iso, today: day.iso === todayISO }"
        @tap="selectDay(day)"
      >
        <text>{{ day.label }}</text>
        <text>{{ day.date }}</text>
        <view v-if="day.iso === todayISO" class="today-dot"></view>
      </view>
    </view>

    <view class="section-head row between">
      <text class="section-title">本周日程</text>
      <view class="row action-row">
        <button class="ghost-btn small" @tap="downloadCourseTemplate"><text class="btn-icon"><AppIcon name="table" /></text>模板</button>
        <button class="ghost-btn small" :disabled="isActionPending('import-courses')" @tap="importSchedule">
          <text class="btn-icon"><AppIcon name="upload" /></text>{{ isActionPending('import-courses') ? '预览中...' : '导入课表' }}
        </button>
      </view>
    </view>
    <view class="section-head row between compact daily-head">
      <view class="daily-title">
        <text class="section-title">当日课程</text>
        <text class="course-count">{{ courseCountText }}</text>
      </view>
      <button class="primary-btn add-course-btn" data-testid="course-add-button" @tap="openCourseForm"><text class="btn-icon"><AppIcon name="calendar" /></text>添加日程</button>
    </view>
    <AppState v-if="loading" type="loading" message="正在加载日程..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />
    <view v-if="courses.length" class="course-timeline">
      <view v-for="course in courses" :key="course.id" class="timeline-item" :class="[courseStatusClass(course), { featured: isFeaturedCourse(course) }]">
        <view class="timeline-axis">
          <view class="timeline-dot"></view>
        </view>
        <view class="course-card">
          <view class="course-top row between">
            <view class="course-time">
              <text class="course-time-main">{{ course.startTime }}</text>
              <text class="course-time-period">{{ coursePeriod(course.startTime) }}</text>
            </view>
            <view class="course-meta">
              <text class="course-class-chip">{{ course.className || '课程' }}</text>
              <text class="course-status-chip">{{ courseStatusText(course) }}</text>
            </view>
          </view>
          <text class="course-name">{{ course.title }}</text>
          <view v-if="course.location" class="course-location">
            <AppIcon name="mapPin" />
            <text>{{ course.location }}</text>
          </view>
          <view v-if="course.note" class="course-note">
            <view class="note-accent"></view>
            <view class="note-copy">
              <text class="note-title">{{ courseTimeRange(course) }}</text>
              <text class="note-body">{{ course.note }}</text>
            </view>
          </view>
          <view class="row action-row course-actions">
            <button class="ghost-btn mini" :disabled="hasPendingAction" @tap="editCourse(course)"><text class="btn-icon"><AppIcon name="edit" /></text>编辑</button>
            <button class="ghost-btn mini danger" :disabled="isActionPending(`delete-course:${course.id}`)" @tap="removeCourse(course.id)">
              <text class="btn-icon"><AppIcon name="trash" /></text>{{ isActionPending(`delete-course:${course.id}`) ? '删除中...' : '删除' }}
            </button>
          </view>
        </view>
      </view>
    </view>
    <AppState
      v-if="!loading && !error && courses.length === 0"
      type="empty"
      title="当天还没有课程"
      message="添加日程或导入课表后，会同步到首页工作台。"
    />

    <view class="section-head row between">
      <text class="section-title">待办事项</text>
      <button class="primary-btn small" data-testid="reminder-add-button" @tap="openReminderForm"><text class="btn-icon"><AppIcon name="plus" /></text>添加待办</button>
    </view>
    <view v-for="item in reminders" :key="item.id" class="card schedule-card todo-card">
      <view class="todo-head">
        <view class="time-meta" :class="`status-${item.status || 'pending'}`">
          <text class="time-text">{{ formatTime(item.remindAt) }}</text>
          <text class="status-text">{{ statusText(item.status) }}</text>
        </view>
        <view class="row action-row">
          <button class="ghost-btn mini" :disabled="hasPendingAction" @tap="editReminder(item)"><text class="btn-icon"><AppIcon name="edit" /></text>编辑</button>
          <button v-if="canSnoozeReminder(item)" class="ghost-btn mini" :disabled="hasPendingAction" @tap="openSnoozeForm(item)"><text class="btn-icon"><AppIcon name="clock" /></text>延后</button>
          <button v-if="canCompleteReminder(item)" class="ghost-btn mini success" :disabled="isActionPending(`complete-reminder:${item.id}`)" @tap="complete(item.id)">
            <text class="btn-icon"><AppIcon name="check" /></text>{{ isActionPending(`complete-reminder:${item.id}`) ? '完成中...' : '完成' }}
          </button>
          <button class="ghost-btn mini danger" :disabled="isActionPending(`delete-reminder:${item.id}`)" @tap="removeReminder(item.id)">
            <text class="btn-icon"><AppIcon name="trash" /></text>{{ isActionPending(`delete-reminder:${item.id}`) ? '删除中...' : '删除' }}
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
import { computed, onBeforeUnmount, ref } from 'vue'
import { onHide, onShow, onUnload } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { appendDictationText, onDictationText } from '../../utils/dictation'
import { chooseImportFile, clockBefore, confirmAction, ensureLoggedIn, errorMessage, hasText, showToast, trimmed, validClock, withinLength } from '../../utils/ui'
import AppState from '../../components/AppState.vue'
import AppIcon from '../../components/AppIcon.vue'

const today = new Date()
const todayISO = ref(toISODate(today))
const selectedDay = ref(todayISO.value)
const weekday = ref(today.getDay())
const currentMinutes = ref(todayMinutes())
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
const completedCourseCount = computed(() => courses.value.filter((course) => courseStatusCode(course) === 'done').length)
const courseCountText = computed(() => {
  const total = courses.value.length
  const done = completedCourseCount.value
  return done > 0 ? `共 ${total} 节课 · ${done} 已完成` : `共 ${total} 节课`
})
const offDictationText = onDictationText(handleDictationText)
const featuredCourseId = computed(() => {
  if (selectedDay.value !== todayISO.value) return ''
  const sortedCourses = [...courses.value].sort((a, b) => (clockMinutes(a.startTime) ?? 0) - (clockMinutes(b.startTime) ?? 0))
  const active = sortedCourses.find((course) => courseStatusCode(course) === 'active')
  if (active?.id) return active.id
  const upcoming = sortedCourses.find((course) => courseStatusCode(course) === 'upcoming')
  return upcoming?.id || ''
})
let clockTimer = null

onShow(() => {
  refreshClock()
  startClockTimer()
  load()
})

onHide(stopClockTimer)
onUnload(stopClockTimer)
onBeforeUnmount(() => offDictationText?.())

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

function handleDictationText(payload) {
  if (payload?.target !== 'schedule') return
  if (!reminderFormOpen.value) openReminderForm()
  if (!reminderForm.value.title) {
    const firstLine = String(payload.text || '').split(/\n|。|\.|！|!|？|\?/)[0].trim()
    reminderForm.value.title = firstLine.slice(0, 120) || '语音待办'
  }
  reminderForm.value.note = appendDictationText(reminderForm.value.note, payload.text)
  showToast('已插入待办草稿')
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

function coursePeriod(value) {
  const hour = Number(String(value || '').split(':')[0])
  if (Number.isNaN(hour)) return ''
  return hour < 12 ? 'AM' : 'PM'
}

function courseTimeRange(course) {
  return `${course.startTime || '--:--'} - ${course.endTime || '--:--'}`
}

function isFeaturedCourse(course) {
  return !!course?.id && course.id === featuredCourseId.value
}

function courseStatus(course) {
  const code = courseStatusCode(course)
  return {
    code,
    text: ({ done: '已完成', active: '进行中', upcoming: '待开始' })[code] || '待开始'
  }
}

function courseStatusCode(course) {
  const relation = selectedDateRelation()
  if (relation < 0) return 'done'
  if (relation > 0) return 'upcoming'

  const start = clockMinutes(course?.startTime)
  const end = clockMinutes(course?.endTime)
  if (start == null || end == null) return 'upcoming'
  if (currentMinutes.value >= end) return 'done'
  if (currentMinutes.value >= start) return 'active'
  return 'upcoming'
}

function courseStatusText(course) {
  return courseStatus(course).text
}

function courseStatusClass(course) {
  return `status-${courseStatus(course).code}`
}

function selectedDateRelation() {
  if (selectedDay.value < todayISO.value) return -1
  if (selectedDay.value > todayISO.value) return 1
  return 0
}

function refreshClock() {
  todayISO.value = toISODate(new Date())
  currentMinutes.value = todayMinutes()
}

function startClockTimer() {
  stopClockTimer()
  clockTimer = setInterval(refreshClock, 60 * 1000)
}

function stopClockTimer() {
  if (!clockTimer) return
  clearInterval(clockTimer)
  clockTimer = null
}

function clockMinutes(value) {
  const match = String(value || '').match(/^(\d{2}):(\d{2})$/)
  if (!match) return null
  return Number(match[1]) * 60 + Number(match[2])
}

function todayMinutes() {
  const now = new Date()
  return now.getHours() * 60 + now.getMinutes()
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
.avatar-btn { width: 88rpx; height: 88rpx; padding: 0; border-radius: 28rpx; background: linear-gradient(180deg, rgba(255,255,255,.96), rgba(250,253,255,.86)); border: 1px solid rgba(97,116,166,.14); box-shadow: 0 10rpx 22rpx rgba(73,91,146,.10), inset 0 1px 0 rgba(255,255,255,.96); display: flex; align-items: center; justify-content: center; }
.section-head { padding: 28rpx 4rpx 10rpx; }
.compact { padding-top: 10rpx; }
.week-row { display: flex; gap: 14rpx; margin-right: -32rpx; padding: 12rpx 32rpx 20rpx 0; overflow-x: auto; scrollbar-width: none; }
.week-row::-webkit-scrollbar { display: none; }
.day { position: relative; flex: 0 0 94rpx; height: 118rpx; border-radius: 26rpx; background: rgba(255,255,255,.62); border: 1rpx solid rgba(255,255,255,.78); box-shadow: inset 0 1rpx 0 rgba(255,255,255,.86); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8rpx; color: #585b77; font-size: 23rpx; font-weight: 900; }
.day.today { color: #46577f; border-color: rgba(98,103,233,.20); box-shadow: 0 10rpx 22rpx rgba(98,103,233,.08), inset 0 1rpx 0 rgba(255,255,255,.90); }
.day.active { color: #fff; background: linear-gradient(145deg, #6f86df, #52b8cf); box-shadow: 0 12rpx 24rpx rgba(89,102,209,.20); }
.today-dot { position: absolute; left: 50%; bottom: 12rpx; width: 10rpx; height: 10rpx; margin-left: -5rpx; border-radius: 999rpx; background: #6267e9; box-shadow: 0 0 0 5rpx rgba(98,103,233,.11); }
.day.active .today-dot { background: rgba(255,255,255,.96); box-shadow: 0 0 0 5rpx rgba(255,255,255,.18); }
.small { min-height: 62rpx; padding: 0 22rpx; font-size: 24rpx; }
.mini { min-height: 54rpx; padding: 0 16rpx; font-size: 22rpx; }
.btn-icon { width: 32rpx; height: 32rpx; background: rgba(255,255,255,.46); font-size: 19rpx; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
.daily-head { align-items: center; }
.daily-title { display: flex; align-items: center; gap: 16rpx; min-width: 0; }
.course-count { flex: 0 0 auto; padding: 7rpx 18rpx; border-radius: 999rpx; color: #9aa1b4; background: rgba(255,255,255,.66); font-size: 21rpx; font-weight: 900; }
.add-course-btn { min-height: 66rpx; padding: 0 26rpx; border-radius: 999rpx; font-size: 26rpx; box-shadow: 0 12rpx 24rpx rgba(74,111,190,.20), inset 0 1px 0 rgba(255,255,255,.32); }
.course-timeline { position: relative; display: flex; flex-direction: column; gap: 18rpx; margin: 16rpx 0 8rpx; }
.course-timeline::before { content: ""; position: absolute; left: 13rpx; top: 22rpx; bottom: 22rpx; width: 2rpx; border-radius: 999rpx; background: rgba(111,134,223,.15); }
.timeline-item { position: relative; display: grid; grid-template-columns: 34rpx minmax(0,1fr); gap: 16rpx; align-items: start; }
.timeline-axis { position: relative; min-height: 44rpx; display: flex; justify-content: center; padding-top: 28rpx; }
.timeline-dot { width: 14rpx; height: 14rpx; border-radius: 999rpx; background: #d6dce8; border: 4rpx solid rgba(255,255,255,.92); box-shadow: 0 0 0 1px rgba(99,113,152,.12); box-sizing: content-box; }
.timeline-item.featured .timeline-dot { width: 16rpx; height: 16rpx; background: #6267e9; box-shadow: 0 0 0 12rpx rgba(98,103,233,.10); }
.timeline-item.status-active .timeline-dot { background: #6267e9; box-shadow: 0 0 0 12rpx rgba(98,103,233,.10); }
.timeline-item.status-done .timeline-dot { background: #75b99d; box-shadow: 0 0 0 8rpx rgba(117,185,157,.10); }
.course-card { position: relative; display: flex; flex-direction: column; gap: 16rpx; padding: 28rpx; border-radius: 28rpx; background: linear-gradient(145deg, rgba(255,255,255,.94), rgba(255,255,255,.74)); border: 1px solid rgba(97,116,166,.10); box-shadow: 0 14rpx 34rpx rgba(73,91,146,.08), inset 0 1px 0 rgba(255,255,255,.96); overflow: hidden; }
.timeline-item.featured .course-card { border-color: rgba(98,103,233,.24); background: linear-gradient(145deg, rgba(255,255,255,.96), rgba(245,247,255,.86)); box-shadow: 0 18rpx 42rpx rgba(98,103,233,.12), inset 0 1px 0 rgba(255,255,255,.96); }
.timeline-item.featured .course-card::before { content: ""; position: absolute; left: 0; top: 24rpx; bottom: 24rpx; width: 8rpx; border-radius: 0 999rpx 999rpx 0; background: linear-gradient(180deg,#6267e9,#6f86df); }
.timeline-item.status-done .course-card { background: linear-gradient(145deg, rgba(255,255,255,.78), rgba(244,251,248,.62)); border-color: rgba(117,185,157,.16); box-shadow: 0 10rpx 24rpx rgba(73,91,146,.05), inset 0 1px 0 rgba(255,255,255,.88); }
.course-top { align-items: flex-start; gap: 16rpx; }
.course-time { display: flex; align-items: baseline; gap: 8rpx; min-width: 0; color: #16223a; font-variant-numeric: tabular-nums; }
.course-time-main { font-size: 36rpx; line-height: 1; font-weight: 950; letter-spacing: 0; }
.course-time-period { color: #7d85a0; font-size: 20rpx; line-height: 1; font-weight: 900; }
.timeline-item.featured .course-time { color: #525ce8; }
.timeline-item.status-done .course-time { color: #78928b; }
.course-meta { flex: 0 1 auto; min-width: 0; display: flex; align-items: center; justify-content: flex-end; gap: 8rpx; flex-wrap: wrap; }
.course-class-chip { flex: 0 0 auto; max-width: 230rpx; padding: 8rpx 16rpx; border-radius: 12rpx; color: #7b7f95; background: rgba(249,250,252,.92); border: 1px solid rgba(97,116,166,.10); font-size: 22rpx; line-height: 1.2; font-weight: 850; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.timeline-item.featured .course-class-chip { color: #6267e9; background: rgba(239,240,255,.92); border-color: rgba(98,103,233,.18); }
.course-status-chip { flex: 0 0 auto; padding: 8rpx 15rpx; border-radius: 999rpx; color: #6d7894; background: rgba(245,249,255,.88); border: 1px solid rgba(97,116,166,.10); font-size: 21rpx; line-height: 1.2; font-weight: 900; white-space: nowrap; }
.timeline-item.status-active .course-status-chip { color: #fff; background: linear-gradient(135deg,#6267e9,#52b8cf); border-color: rgba(98,103,233,.20); box-shadow: 0 8rpx 18rpx rgba(98,103,233,.16); }
.timeline-item.status-upcoming .course-status-chip { color: #52627d; background: rgba(245,249,255,.92); }
.timeline-item.status-done .course-class-chip { color: #5d8176; background: rgba(238,250,245,.88); border-color: rgba(117,185,157,.16); }
.timeline-item.status-done .course-status-chip { color: #2f7f6e; background: rgba(235,250,243,.92); border-color: rgba(117,185,157,.18); }
.course-name { display: block; color: #172039; font-size: 34rpx; line-height: 1.22; font-weight: 950; overflow-wrap: anywhere; }
.timeline-item.status-done .course-name { color: #506963; }
.course-location { display: flex; align-items: center; gap: 10rpx; color: #7b8195; font-size: 25rpx; line-height: 1.35; font-weight: 750; }
.course-location .app-icon { flex: 0 0 auto; width: 27rpx; height: 27rpx; color: #6f86df; }
.timeline-item.status-done .course-location .app-icon { color: #75b99d; }
.course-note { position: relative; display: flex; gap: 16rpx; margin-top: 4rpx; padding: 20rpx 22rpx; border-radius: 18rpx; background: rgba(255,255,255,.82); border: 1px solid rgba(97,116,166,.10); box-shadow: 0 8rpx 20rpx rgba(73,91,146,.06); overflow: hidden; }
.note-accent { flex: 0 0 6rpx; align-self: stretch; border-radius: 999rpx; background: linear-gradient(180deg,#6267e9,#6f86df); }
.timeline-item.status-done .note-accent { background: linear-gradient(180deg,#75b99d,#4ea489); }
.note-copy { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 8rpx; }
.note-title { color: #46577f; font-size: 23rpx; line-height: 1.2; font-weight: 950; }
.note-body { color: #7b8195; font-size: 24rpx; line-height: 1.45; overflow-wrap: anywhere; }
.course-actions { justify-content: flex-end; margin-top: 2rpx; }
.todo-card { background: linear-gradient(145deg, rgba(255,255,255,.78), rgba(255,248,224,.58)); }
.todo-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16rpx; }
.todo-head .action-row { flex: 1; min-width: 0; padding-top: 2rpx; }
.time-meta { flex: 0 0 auto; min-width: 136rpx; padding: 16rpx 18rpx; border-radius: 24rpx; display: flex; flex-direction: column; gap: 5rpx; color: #46577f; background: linear-gradient(145deg, rgba(255,255,255,.96), rgba(245,249,255,.78)); border: 1px solid rgba(97,116,166,.12); box-shadow: 0 10rpx 22rpx rgba(73,91,146,.08), inset 0 1px 0 rgba(255,255,255,.96); box-sizing: border-box; }
.time-text { font-size: 34rpx; line-height: 1; font-weight: 950; letter-spacing: 0; font-variant-numeric: tabular-nums; }
.status-text { display: flex; align-items: center; gap: 8rpx; color: #6d7894; font-size: 22rpx; line-height: 1.2; font-weight: 900; white-space: nowrap; }
.status-text::before { content: ""; width: 10rpx; height: 10rpx; border-radius: 999rpx; background: #6f86df; box-shadow: 0 0 0 5rpx rgba(111,134,223,.12); }
.time-meta.status-done { color: #2f7f6e; background: linear-gradient(145deg, rgba(242,255,250,.96), rgba(229,248,240,.76)); }
.time-meta.status-done .status-text::before { background: #3a8a76; box-shadow: 0 0 0 5rpx rgba(58,138,118,.13); }
.time-meta.status-snoozed { color: #7c679f; background: linear-gradient(145deg, rgba(250,246,255,.96), rgba(243,236,255,.76)); }
.time-meta.status-snoozed .status-text::before { background: #a990ea; box-shadow: 0 0 0 5rpx rgba(169,144,234,.13); }
.state-card { display: flex; flex-direction: column; gap: 18rpx; }
.compact-state { margin: 0; padding: 22rpx; }
.preview-row { padding: 24rpx; border-radius: 24rpx; background: rgba(255,255,255,.58); border: 1rpx solid rgba(255,255,255,.78); display: flex; flex-direction: column; gap: 8rpx; }
.preview-row.invalid { background: rgba(255,241,242,.88); }
.snooze-summary { padding: 20rpx 22rpx; border-radius: 22rpx; background: rgba(255,255,255,.58); border: 1rpx solid rgba(255,255,255,.78); }
@media (max-width: 380px) {
  .todo-head { flex-direction: column; align-items: stretch; }
  .todo-head .action-row { justify-content: flex-start; }
  .time-meta { width: fit-content; }
}
</style>
