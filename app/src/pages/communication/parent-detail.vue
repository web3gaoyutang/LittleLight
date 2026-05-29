<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">学生 · 班级 · 风险 · 跟进</text>
        <view class="title">{{ form.parentName || '家长档案' }}</view>
      </view>
      <button class="ghost-btn small" @tap="load">刷新</button>
    </view>

    <AppState v-if="loading" type="loading" message="正在加载家长档案..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="card">
      <view class="grid">
        <view class="field">
          <text class="caption">学生</text>
          <input class="input" :class="{ invalid: formError && !hasText(form.studentName) }" v-model="form.studentName" maxlength="60" placeholder="学生姓名" aria-label="学生姓名" data-testid="detail-parent-student-input" />
        </view>
        <view class="field">
          <text class="caption">班级</text>
          <input class="input" :class="{ invalid: formError && !hasText(form.className) }" v-model="form.className" maxlength="80" placeholder="班级" aria-label="学生班级" data-testid="detail-parent-class-input" />
        </view>
        <view class="field">
          <text class="caption">家长称呼</text>
          <input class="input" :class="{ invalid: formError && !hasText(form.parentName) }" v-model="form.parentName" maxlength="80" placeholder="家长称呼" aria-label="家长称呼" data-testid="detail-parent-name-input" />
        </view>
        <view class="field">
          <text class="caption">关系</text>
          <input class="input" :class="{ invalid: formError && !hasText(form.relationship) }" v-model="form.relationship" maxlength="40" placeholder="关系" aria-label="家长关系" data-testid="detail-parent-relationship-input" />
        </view>
        <view class="field">
          <text class="caption">联系方式</text>
          <input class="input" :class="{ invalid: formError && !withinLength(form.contact, 80) }" v-model="form.contact" maxlength="80" placeholder="手机号或微信" aria-label="家长联系方式" data-testid="detail-parent-contact-input" />
        </view>
        <view class="field">
          <text class="caption">沟通风格</text>
          <picker class="picker" :range="styles" @change="form.communicationStyle = styles[$event.detail.value]">
            <text>{{ form.communicationStyle || '选择风格' }}</text>
          </picker>
        </view>
      </view>

      <view class="field">
        <text class="caption">风险等级</text>
        <view class="risk-row">
          <button v-for="risk in risks" :key="risk.value" class="risk-btn" :class="{ active: form.riskLevel === risk.value }" @tap="form.riskLevel = risk.value">
            {{ risk.label }}
          </button>
        </view>
      </view>

      <view class="field">
        <text class="caption">重点备注</text>
        <textarea class="textarea" :class="{ invalid: formError && !withinLength(form.importantNotes, 2000) }" v-model="form.importantNotes" maxlength="2000" placeholder="睡眠、作业、情绪、家庭沟通背景等" aria-label="重点备注" data-testid="detail-parent-notes-input" />
      </view>
      <view class="field">
        <text class="caption">下一步动作</text>
        <textarea class="textarea short" :class="{ invalid: formError && !withinLength(form.nextAction, 1000) }" v-model="form.nextAction" maxlength="1000" placeholder="下一次跟进要做什么" aria-label="下一步跟进" data-testid="detail-parent-next-action-input" />
      </view>
      <text v-if="formError" class="form-error">{{ formError }}</text>
      <button class="primary-btn save-btn" :disabled="saving" @tap="save">{{ saving ? '保存中...' : '保存档案' }}</button>
    </view>

    <view class="section-head row between">
      <text class="section-title">沟通记录</text>
      <button class="primary-btn small" @tap="openRecordForm">添加记录</button>
    </view>
    <view class="search-bar">
      <input class="input search-input" v-model="recordQuery" confirm-type="search" placeholder="搜索原因、摘要、结果或风险" aria-label="搜索沟通记录" data-testid="detail-record-search-input" @confirm="searchRecords" />
      <button class="ghost-btn small" :disabled="loadingRecords" @tap="searchRecords">搜索</button>
    </view>
    <AppState v-if="loadingRecords && records.length === 0" type="loading" message="正在加载沟通记录..." />
    <view v-for="record in records" :key="record.id" class="card record">
      <view class="row between">
        <text class="tag">{{ record.channel }} · {{ riskText(record.riskLevel) }}</text>
        <view class="row action-row">
          <button v-if="canCompleteFollowUp(record)" class="ghost-btn mini success" @tap="completeRecordFollowUp(record)">完成跟进</button>
          <button class="ghost-btn mini" @tap="editRecord(record)">编辑</button>
          <button class="ghost-btn mini danger" @tap="removeRecord(record.id)">删除</button>
        </view>
      </view>
      <text class="caption">{{ followUpStatusText(record) }}</text>
      <text class="section-title">{{ record.reason }}</text>
      <text class="body">{{ record.summary }}</text>
      <text class="caption">{{ record.result }}</text>
    </view>
    <AppState
      v-if="!loading && !loadingRecords && !error && records.length === 0"
      type="empty"
      title="暂无沟通记录"
      message="保存档案后，可以添加一次跟进记录。"
    />
    <button v-if="recordsHasMore" class="ghost-btn load-more" :disabled="loadingRecords" @tap="loadMoreRecords">
      {{ loadingRecords ? '加载中...' : '加载更多沟通记录' }}
    </button>

    <view v-if="recordFormOpen" class="modal-mask" @tap="closeRecordForm">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">{{ editingRecordId ? '编辑沟通记录' : '添加沟通记录' }}</text>
          <button class="ghost-btn mini" @tap="closeRecordForm">关闭</button>
        </view>
        <view class="form-grid">
          <input class="input" :class="{ invalid: recordFormError && !hasText(recordForm.student) }" v-model="recordForm.student" maxlength="60" placeholder="学生姓名" aria-label="沟通记录学生姓名" data-testid="detail-record-student-input" />
          <input class="input" :class="{ invalid: recordFormError && !hasText(recordForm.channel) }" v-model="recordForm.channel" maxlength="40" placeholder="沟通渠道，例如 微信" aria-label="沟通渠道" data-testid="detail-record-channel-input" />
          <input class="input" :class="{ invalid: recordFormError && !hasText(recordForm.reason) }" v-model="recordForm.reason" maxlength="200" placeholder="沟通原因" aria-label="沟通原因" data-testid="detail-record-reason-input" />
          <textarea class="textarea short" :class="{ invalid: recordFormError && !hasText(recordForm.summary) }" v-model="recordForm.summary" maxlength="2000" placeholder="沟通摘要" aria-label="沟通摘要" data-testid="detail-record-summary-input" />
          <textarea class="textarea short" :class="{ invalid: recordFormError && !withinLength(recordForm.result, 2000) }" v-model="recordForm.result" maxlength="2000" placeholder="沟通结果" aria-label="沟通结果" data-testid="detail-record-result-input" />
          <view class="row follow-row">
            <input class="input half" :class="{ invalid: recordFormError && !validISODate(recordForm.followUpDate) }" v-model="recordForm.followUpDate" placeholder="跟进日期 2026-05-30" aria-label="跟进日期" data-testid="detail-record-follow-up-date-input" />
            <input class="input half" :class="{ invalid: recordFormError && !validClock(recordForm.followUpTime) }" v-model="recordForm.followUpTime" placeholder="跟进时间 17:00" aria-label="跟进时间" data-testid="detail-record-follow-up-time-input" />
          </view>
          <view class="risk-row">
            <button v-for="risk in risks" :key="risk.value" class="risk-btn" :class="{ active: recordForm.riskLevel === risk.value }" @tap="recordForm.riskLevel = risk.value">
              {{ risk.label }}
            </button>
          </view>
          <text v-if="recordFormError" class="form-error">{{ recordFormError }}</text>
          <button class="primary-btn" :disabled="saving" @tap="submitRecord">{{ saving ? '保存中...' : (editingRecordId ? '更新沟通记录' : '保存沟通记录') }}</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { api, listItems, listPageInfo } from '../../api/client'
import { confirmAction, ensureLoggedIn, errorMessage, hasText, showToast, trimmed, validClock, withinLength } from '../../utils/ui'
import AppState from '../../components/AppState.vue'

const id = ref('')
const form = ref({})
const records = ref([])
const loading = ref(false)
const loadingRecords = ref(false)
const saving = ref(false)
const error = ref('')
const formError = ref('')
const recordQuery = ref('')
const recordsOffset = ref(0)
const recordsHasMore = ref(false)
const pageSize = 20
const recordFormOpen = ref(false)
const recordFormError = ref('')
const editingRecordId = ref('')
const recordForm = ref(defaultRecordForm())
const styles = ['容易焦虑', '比较敏感', '沟通积极', '关注成绩']
const risks = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' }
]

onLoad(async (query) => {
  id.value = query?.id || ''
  await load()
})

async function load() {
  if (!ensureLoggedIn(api)) return
  if (!id.value) return
  loading.value = true
  error.value = ''
  try {
    form.value = await api.parent(id.value)
    await fetchRecords({ reset: true })
    formError.value = ''
  } catch (err) {
    error.value = errorMessage(err, '家长档案加载失败')
  } finally {
    loading.value = false
  }
}

async function fetchRecords({ reset = false } = {}) {
  if (loadingRecords.value) return
  loadingRecords.value = true
  try {
    const offset = reset ? 0 : recordsOffset.value
    const response = await api.records(id.value, { q: recordQuery.value, limit: pageSize, offset })
    const data = listItems(response)
    const page = listPageInfo(response, pageSize, offset)
    records.value = reset ? data : [...records.value, ...data]
    recordsOffset.value = page.nextOffset
    recordsHasMore.value = page.hasMore
  } finally {
    loadingRecords.value = false
  }
}

async function searchRecords() {
  try {
    await fetchRecords({ reset: true })
  } catch (err) {
    showToast(errorMessage(err, '沟通记录搜索失败'))
  }
}

async function loadMoreRecords() {
  try {
    await fetchRecords()
  } catch (err) {
    showToast(errorMessage(err, '沟通记录加载失败'))
  }
}

async function save() {
  if (saving.value) return
  const validation = validateParentForm(form.value)
  if (validation) {
    formError.value = validation
    return
  }
  saving.value = true
  formError.value = ''
  try {
    form.value = await api.updateParent(id.value, cleanParentPayload(form.value))
    showToast('已保存家长档案')
  } catch (err) {
    formError.value = errorMessage(err, '档案保存失败')
    showToast(formError.value)
  } finally {
    saving.value = false
  }
}

function openRecordForm() {
  editingRecordId.value = ''
  recordForm.value = defaultRecordForm(form.value)
  recordFormError.value = ''
  recordFormOpen.value = true
}

function closeRecordForm() {
  recordFormOpen.value = false
  recordFormError.value = ''
  editingRecordId.value = ''
}

async function submitRecord() {
  if (saving.value) return
  const validation = validateRecordForm(recordForm.value)
  if (validation) {
    recordFormError.value = validation
    return
  }
  saving.value = true
  recordFormError.value = ''
  try {
    const payload = {
      parentId: id.value,
      student: trimmed(recordForm.value.student),
      channel: trimmed(recordForm.value.channel),
      reason: trimmed(recordForm.value.reason),
      summary: trimmed(recordForm.value.summary),
      result: trimmed(recordForm.value.result),
      riskLevel: recordForm.value.riskLevel || 'low',
      followUpAt: recordFollowUpISO(recordForm.value),
      followUpStatus: recordForm.value.followUpStatus || 'pending',
      followedUpAt: recordForm.value.followedUpAt || null
    }
    const record = editingRecordId.value
      ? await api.updateRecord(editingRecordId.value, payload)
      : await api.createRecord(payload)
    records.value = editingRecordId.value
      ? records.value.map((item) => item.id === record.id ? record : item)
      : [record, ...records.value]
    if (!editingRecordId.value) recordsOffset.value += 1
    const wasEditing = !!editingRecordId.value
    closeRecordForm()
    showToast(wasEditing ? '已更新沟通记录' : '已添加沟通记录')
  } catch (err) {
    recordFormError.value = errorMessage(err, '沟通记录保存失败')
    showToast(recordFormError.value)
  } finally {
    saving.value = false
  }
}

function editRecord(record) {
  editingRecordId.value = record.id
  recordForm.value = recordFormFromRecord(record, form.value)
  recordFormError.value = ''
  recordFormOpen.value = true
}

async function removeRecord(recordId) {
  const confirmed = await confirmAction({
    title: '删除沟通记录',
    content: '删除后这条记录会从家长档案中移除，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) return
  try {
    await api.deleteRecord(recordId)
    records.value = records.value.filter((item) => item.id !== recordId)
    recordsOffset.value = Math.max(0, recordsOffset.value - 1)
    showToast('已删除沟通记录')
  } catch (err) {
    showToast(errorMessage(err, '删除沟通记录失败'))
  }
}

async function completeRecordFollowUp(record) {
  const confirmed = await confirmAction({
    title: '完成跟进',
    content: `确认已完成 ${record.student || form.value.studentName || '这位学生'} 的本次跟进吗？`,
    confirmText: '完成'
  })
  if (!confirmed) return
  try {
    const updated = await api.completeRecordFollowUp(record.id)
    records.value = records.value.map((item) => item.id === updated.id ? updated : item)
    showToast('已标记为完成跟进')
  } catch (err) {
    showToast(errorMessage(err, '完成跟进失败'))
  }
}

function riskText(value) {
  return ({ low: '低风险', medium: '中风险', high: '高风险' })[value] || '低风险'
}

function formatTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function followUpStatusText(record) {
  if (record?.followUpStatus === 'done') {
    const doneAt = formatTime(record.followedUpAt)
    return doneAt ? `已跟进 · ${doneAt}` : '已跟进'
  }
  const followAt = formatTime(record?.followUpAt)
  return followAt ? `待跟进 · ${followAt}` : '未设置跟进时间'
}

function canCompleteFollowUp(record) {
  return !!record?.id && record.followUpStatus !== 'done'
}

function defaultRecordForm(parent = {}) {
  return {
    parentId: id.value,
    student: parent.studentName || '',
    channel: '微信',
    reason: parent.nextAction || '',
    summary: '',
    result: '',
    riskLevel: parent.riskLevel || 'low',
    followUpStatus: 'pending',
    followedUpAt: null,
    ...followUpFields(defaultFollowUpDate())
  }
}

function recordFormFromRecord(record, parent = {}) {
  return {
    parentId: id.value,
    student: record.student || parent.studentName || '',
    channel: record.channel || '微信',
    reason: record.reason || '',
    summary: record.summary || '',
    result: record.result || '',
    riskLevel: record.riskLevel || parent.riskLevel || 'low',
    followUpStatus: record.followUpStatus || 'pending',
    followedUpAt: record.followedUpAt || null,
    ...followUpFields(record.followUpAt || defaultFollowUpDate())
  }
}

function validateParentForm(value) {
  if (!hasText(value.studentName) || !hasText(value.className) || !hasText(value.parentName) || !hasText(value.relationship)) return '请补全学生、班级、家长称呼和关系'
  if (!withinLength(value.studentName, 60) || !withinLength(value.parentName, 80)) return '学生姓名最多 60 个字，家长称呼最多 80 个字'
  if (!withinLength(value.className, 80) || !withinLength(value.relationship, 40)) return '班级最多 80 个字，关系最多 40 个字'
  if (!withinLength(value.contact, 80) || !withinLength(value.communicationStyle, 80)) return '联系方式和沟通风格最多 80 个字'
  if (!withinLength(value.importantNotes, 2000)) return '重点备注最多 2000 个字'
  if (!withinLength(value.nextAction, 1000)) return '下一步动作最多 1000 个字'
  return ''
}

function validateRecordForm(value) {
  if (!hasText(value.student) || !hasText(value.channel) || !hasText(value.reason) || !hasText(value.summary)) return '请补全学生、渠道、原因和摘要'
  if (!withinLength(value.student, 60) || !withinLength(value.channel, 40)) return '学生姓名最多 60 个字，渠道最多 40 个字'
  if (!withinLength(value.reason, 200)) return '沟通原因最多 200 个字'
  if (!withinLength(value.summary, 2000) || !withinLength(value.result, 2000)) return '沟通摘要和结果最多 2000 个字'
  if (!validISODate(value.followUpDate)) return '跟进日期请使用 2026-05-30 这样的格式'
  if (!validClock(value.followUpTime)) return '跟进时间请使用 17:00 这样的 24 小时格式'
  return ''
}

function cleanParentPayload(value) {
  return {
    ...value,
    studentName: trimmed(value.studentName),
    className: trimmed(value.className),
    parentName: trimmed(value.parentName),
    relationship: trimmed(value.relationship),
    contact: trimmed(value.contact),
    communicationStyle: trimmed(value.communicationStyle),
    importantNotes: trimmed(value.importantNotes),
    nextAction: trimmed(value.nextAction)
  }
}

function recordFollowUpISO(value) {
  return new Date(`${trimmed(value.followUpDate)}T${trimmed(value.followUpTime)}:00`).toISOString()
}

function followUpFields(value) {
  const date = parseDate(value)
  return {
    followUpDate: toISODate(date),
    followUpTime: `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  }
}

function defaultFollowUpDate() {
  return new Date(Date.now() + 24 * 60 * 60 * 1000)
}

function parseDate(value) {
  const date = value ? new Date(value) : defaultFollowUpDate()
  return Number.isNaN(date.getTime()) ? defaultFollowUpDate() : date
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
.grid { display: grid; grid-template-columns: 1fr; gap: 18rpx; }
.field { display: flex; flex-direction: column; gap: 10rpx; margin-bottom: 22rpx; }
.input, .picker { min-height: 82rpx; }
.textarea.short { min-height: 120rpx; }
.form-grid { display: flex; flex-direction: column; gap: 18rpx; }
.half { flex: 1; min-width: 0; }
.risk-row { display: flex; gap: 14rpx; }
.follow-row { align-items: stretch; }
.risk-btn { flex: 1; min-height: 70rpx; border-radius: 24rpx; background: rgba(255,255,255,.72); color: #53607a; font-weight: 900; }
.risk-btn.active { background: linear-gradient(135deg,#0891b2,#059669); color: #fff; }
.save-btn { width: 100%; margin-top: 10rpx; }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.success { color: #059669; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
.record { display: flex; flex-direction: column; gap: 14rpx; }
.search-bar { padding: 0 32rpx 12rpx; display: flex; gap: 12rpx; align-items: center; }
.search-input { flex: 1; min-width: 0; }
.load-more { margin: 8rpx 32rpx 28rpx; }
.modal-mask { position: fixed; inset: 0; z-index: 20; background: rgba(18,32,56,.34); display: flex; align-items: flex-end; padding: 32rpx; box-sizing: border-box; }
.modal-card { width: 100%; max-height: 88vh; overflow-y: auto; padding: 32rpx; border-radius: 32rpx; background: #fff; box-shadow: 0 -20rpx 60rpx rgba(24,32,51,.16); display: flex; flex-direction: column; gap: 24rpx; }
</style>
