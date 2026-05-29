<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">AI 回复 · 重点关注</text>
        <view class="title">沟通助手</view>
      </view>
      <view class="avatar-btn"><text class="icon-chip"><AppIcon name="bot" /></text></view>
    </view>

    <view class="card">
      <view class="row between">
        <text class="section-title">生成家校草稿</text>
        <text class="icon-chip"><AppIcon name="sparkles" /></text>
      </view>
      <textarea class="textarea" :class="{ invalid: aiFormError && (!hasText(issue) || !withinLength(issue, 1200)) }" v-model="issue" maxlength="1200" placeholder="描述学生情况、家长问题和沟通目标" aria-label="家校沟通问题描述" data-testid="parent-draft-issue-input" />
      <view class="row picker-row">
        <picker class="picker" :range="styles" @change="style = styles[$event.detail.value]"><text>{{ style }}</text></picker>
        <picker class="picker" :range="tones" @change="tone = tones[$event.detail.value]"><text>{{ tone }}</text></picker>
      </view>
      <text v-if="aiFormError" class="form-error">{{ aiFormError }}</text>
      <button class="primary-btn" :disabled="generating" @tap="generate"><text class="btn-icon"><AppIcon name="edit" /></text>{{ generating ? '生成中...' : '生成草稿' }}</button>
    </view>

    <AppState v-if="loading" type="loading" message="正在加载沟通数据..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="section-head row between">
      <text class="section-title">可选草稿</text>
      <text class="tag">自动适配</text>
    </view>
	    <view v-for="draft in drafts" :key="draft.id" class="card draft">
	      <view class="row between">
	        <text class="tag" :class="{ dangerTag: isHighRiskDraft(draft) }">{{ draft.version }} · {{ safetyText(draft.safety) }}</text>
	        <view class="row action-row">
          <button class="ghost-btn small" :disabled="isActionPending(`save-draft:${draft.id}`)" @tap="saveDraftAsRecord(draft)">
            <text class="btn-icon"><AppIcon name="bookmark" /></text>
            {{ isActionPending(`save-draft:${draft.id}`) ? '处理中...' : '存记录' }}
          </button>
          <button class="ghost-btn small" :disabled="isActionPending(`copy-draft:${draft.id}`)" @tap="copyDraft(draft)">
            <text class="btn-icon"><AppIcon name="copy" /></text>
            {{ isActionPending(`copy-draft:${draft.id}`) ? '复制中...' : '复制' }}
          </button>
	        </view>
	      </view>
	      <text class="caption block">{{ aiSourceText(draft) }}</text>
	      <view v-if="needsReview(draft)" class="safety-note">
	        <text class="block">{{ draftSafetyReason(draft) }}</text>
	        <text v-if="draftSafetySignals(draft)" class="caption block">触发信号：{{ draftSafetySignals(draft) }}</text>
	      </view>
	      <text class="body">{{ draft.content }}</text>
	    </view>
    <AppState
      v-if="!generating && drafts.length === 0"
      type="empty"
      title="还没有生成草稿"
      message="写下沟通背景，系统会生成几版可审核的话术。"
    />

    <view class="section-head row between">
      <text class="section-title">重点关注</text>
      <view class="row action-row">
        <button class="ghost-btn small" @tap="downloadParentTemplate"><text class="btn-icon"><AppIcon name="table" /></text>模板</button>
        <button class="ghost-btn small" :disabled="isActionPending('import-parents')" @tap="importParents">
          <text class="btn-icon"><AppIcon name="upload" /></text>
          {{ isActionPending('import-parents') ? '预览中...' : '导入名单' }}
        </button>
        <button class="primary-btn small" :disabled="hasPendingAction" @tap="openParentForm"><text class="btn-icon"><AppIcon name="plus" /></text>新增家长</button>
      </view>
    </view>
    <view class="search-bar">
      <text class="search-icon"><AppIcon name="search" /></text>
      <input class="input search-input" v-model="parentQuery" confirm-type="search" placeholder="搜索学生、家长、班级或备注" aria-label="搜索家长档案" data-testid="parent-search-input" @confirm="searchParents" />
      <button class="ghost-btn small" :disabled="loadingParents" @tap="searchParents">搜索</button>
    </view>
    <view v-for="parent in parents" :key="parent.id" class="card parent-card" @tap="openParent(parent)">
      <view class="row between">
        <view class="parent-title">
          <text class="section-title">{{ parent.parentName }}</text>
          <text class="caption block">{{ parent.studentName }} · {{ parent.className }} · {{ riskText(parent.riskLevel) }}</text>
        </view>
        <view class="row action-row">
          <button class="ghost-btn mini" :disabled="hasPendingAction" @tap.stop="selectParent(parent)"><text class="btn-icon"><AppIcon name="message" /></text>记录</button>
          <button class="ghost-btn mini" :disabled="hasPendingAction" @tap.stop="editParent(parent)"><text class="btn-icon"><AppIcon name="edit" /></text>编辑</button>
          <button class="ghost-btn mini danger" :disabled="isActionPending(`delete-parent:${parent.id}`)" @tap.stop="removeParent(parent)">
            <text class="btn-icon"><AppIcon name="trash" /></text>
            {{ isActionPending(`delete-parent:${parent.id}`) ? '删除中...' : '删除' }}
          </button>
        </view>
      </view>
      <text class="body">{{ parent.nextAction || parent.importantNotes || '暂无下一步跟进' }}</text>
    </view>
    <AppState
      v-if="!loading && !error && parents.length === 0"
      type="empty"
      title="还没有重点关注对象"
      message="新增家长档案或导入名单后，可以沉淀沟通记录。"
    />
    <button v-if="parentsHasMore" class="ghost-btn load-more" :disabled="loadingParents" @tap="loadMoreParents">
      {{ loadingParents ? '加载中...' : '加载更多关注对象' }}
    </button>

    <view class="section-head row between">
      <text class="section-title">沟通记录</text>
      <button class="primary-btn small" :disabled="hasPendingAction" @tap="openRecordForm()"><text class="btn-icon"><AppIcon name="plus" /></text>手动添加</button>
    </view>
    <view class="search-bar">
      <text class="search-icon"><AppIcon name="search" /></text>
      <input class="input search-input" v-model="recordQuery" confirm-type="search" placeholder="搜索原因、摘要、结果或风险" aria-label="搜索沟通记录" data-testid="record-search-input" @confirm="searchRecords" />
      <button class="ghost-btn small" :disabled="loadingRecords" @tap="searchRecords">搜索</button>
    </view>
    <view v-for="record in records" :key="record.id" class="card">
      <view class="row between">
        <text class="tag">{{ record.channel }} · {{ record.riskLevel }}</text>
        <view class="row action-row">
          <button v-if="canCompleteFollowUp(record)" class="ghost-btn mini success" :disabled="isActionPending(`complete-record:${record.id}`)" @tap="completeRecordFollowUp(record)">
            <text class="btn-icon"><AppIcon name="check" /></text>
            {{ isActionPending(`complete-record:${record.id}`) ? '完成中...' : '完成跟进' }}
          </button>
          <button class="ghost-btn mini" :disabled="hasPendingAction" @tap="editRecord(record)"><text class="btn-icon"><AppIcon name="edit" /></text>编辑</button>
          <button class="ghost-btn mini danger" :disabled="isActionPending(`delete-record:${record.id}`)" @tap="removeRecord(record.id)">
            <text class="btn-icon"><AppIcon name="trash" /></text>
            {{ isActionPending(`delete-record:${record.id}`) ? '删除中...' : '删除' }}
          </button>
        </view>
      </view>
      <text class="caption block">{{ followUpStatusText(record) }}</text>
      <text class="section-title">{{ record.student }} · {{ record.reason }}</text>
      <text class="body">{{ record.summary }}</text>
      <text class="caption">{{ record.result }}</text>
    </view>
    <AppState
      v-if="!loading && !error && records.length === 0"
      type="empty"
      title="暂无沟通记录"
      message="选择一个家长后，可以手动添加记录或把 AI 草稿沉淀下来。"
    />
    <button v-if="recordsHasMore" class="ghost-btn load-more" :disabled="loadingRecords" @tap="loadMoreRecords">
      {{ loadingRecords ? '加载中...' : '加载更多沟通记录' }}
    </button>

    <view v-if="parentFormOpen || recordFormOpen" class="modal-mask" @tap="closeForms">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">{{ parentFormOpen ? (editingParentId ? '编辑家长档案' : '新增家长档案') : (editingRecordId ? '编辑沟通记录' : '添加沟通记录') }}</text>
          <button class="ghost-btn mini" @tap="closeForms">关闭</button>
        </view>

        <view v-if="parentFormOpen" class="form-grid">
          <input class="input" :class="{ invalid: parentFormError && !hasText(parentForm.studentName) }" v-model="parentForm.studentName" maxlength="60" placeholder="学生姓名" aria-label="学生姓名" data-testid="parent-student-input" />
          <input class="input" :class="{ invalid: parentFormError && !hasText(parentForm.className) }" v-model="parentForm.className" maxlength="80" placeholder="班级" aria-label="学生班级" data-testid="parent-class-input" />
          <input class="input" :class="{ invalid: parentFormError && !hasText(parentForm.parentName) }" v-model="parentForm.parentName" maxlength="80" placeholder="家长称呼" aria-label="家长称呼" data-testid="parent-name-input" />
          <input class="input" :class="{ invalid: parentFormError && !hasText(parentForm.relationship) }" v-model="parentForm.relationship" maxlength="40" placeholder="关系" aria-label="家长关系" data-testid="parent-relationship-input" />
          <input class="input" :class="{ invalid: parentFormError && !withinLength(parentForm.contact, 80) }" v-model="parentForm.contact" maxlength="80" placeholder="联系方式" aria-label="家长联系方式" data-testid="parent-contact-input" />
          <picker class="input" :range="styles" @change="parentForm.communicationStyle = styles[$event.detail.value]"><text>{{ parentForm.communicationStyle || '沟通风格' }}</text></picker>
          <view class="risk-row">
            <button v-for="risk in risks" :key="risk.value" class="risk-btn" :class="{ active: parentForm.riskLevel === risk.value }" @tap="parentForm.riskLevel = risk.value">{{ risk.label }}</button>
          </view>
          <textarea class="textarea short" :class="{ invalid: parentFormError && !withinLength(parentForm.importantNotes, 2000) }" v-model="parentForm.importantNotes" maxlength="2000" placeholder="重点备注" aria-label="重点备注" data-testid="parent-notes-input" />
          <textarea class="textarea short" :class="{ invalid: parentFormError && !withinLength(parentForm.nextAction, 1000) }" v-model="parentForm.nextAction" maxlength="1000" placeholder="下一步跟进" aria-label="下一步跟进" data-testid="parent-next-action-input" />
          <text v-if="parentFormError" class="form-error">{{ parentFormError }}</text>
          <button class="primary-btn" :disabled="saving" @tap="submitParent">{{ saving ? '保存中...' : (editingParentId ? '更新家长档案' : '保存家长档案') }}</button>
        </view>

        <view v-else class="form-grid">
          <picker class="input" :range="parentNames" @change="pickRecordParent"><text>{{ recordForm.student || '选择学生/家长' }}</text></picker>
          <input class="input" :class="{ invalid: recordFormError && !hasText(recordForm.student) }" v-model="recordForm.student" maxlength="60" placeholder="学生姓名" aria-label="沟通记录学生姓名" data-testid="record-student-input" />
          <input class="input" :class="{ invalid: recordFormError && !hasText(recordForm.channel) }" v-model="recordForm.channel" maxlength="40" placeholder="沟通渠道，例如 微信" aria-label="沟通渠道" data-testid="record-channel-input" />
          <input class="input" :class="{ invalid: recordFormError && !hasText(recordForm.reason) }" v-model="recordForm.reason" maxlength="200" placeholder="沟通原因" aria-label="沟通原因" data-testid="record-reason-input" />
          <textarea class="textarea short" :class="{ invalid: recordFormError && !hasText(recordForm.summary) }" v-model="recordForm.summary" maxlength="2000" placeholder="沟通摘要" aria-label="沟通摘要" data-testid="record-summary-input" />
          <textarea class="textarea short" :class="{ invalid: recordFormError && !withinLength(recordForm.result, 2000) }" v-model="recordForm.result" maxlength="2000" placeholder="沟通结果" aria-label="沟通结果" data-testid="record-result-input" />
          <view class="row follow-row">
            <input class="input half" :class="{ invalid: recordFormError && !validISODate(recordForm.followUpDate) }" v-model="recordForm.followUpDate" placeholder="跟进日期 2026-05-30" aria-label="跟进日期" data-testid="record-follow-up-date-input" />
            <input class="input half" :class="{ invalid: recordFormError && !validClock(recordForm.followUpTime) }" v-model="recordForm.followUpTime" placeholder="跟进时间 17:00" aria-label="跟进时间" data-testid="record-follow-up-time-input" />
          </view>
          <view class="risk-row">
            <button v-for="risk in risks" :key="risk.value" class="risk-btn" :class="{ active: recordForm.riskLevel === risk.value }" @tap="recordForm.riskLevel = risk.value">{{ risk.label }}</button>
          </view>
          <text v-if="recordFormError" class="form-error">{{ recordFormError }}</text>
          <button class="primary-btn" :disabled="saving" @tap="submitRecord">{{ saving ? '保存中...' : '保存沟通记录' }}</button>
        </view>
      </view>
    </view>

    <view v-if="importDialogOpen" class="modal-mask" @tap="closeImportDialog">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">名单导入预览</text>
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
        <button class="primary-btn" :disabled="saving || isActionPending('confirm-import-parents') || !importReadyCount" @tap="confirmImportParents">
          {{ isActionPending('confirm-import-parents') ? '导入中...' : `确认导入 ${importReadyCount} 条` }}
        </button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api, listItems, listPageInfo } from '../../api/client'
import { chooseImportFile, confirmAction, ensureLoggedIn, errorMessage, hasText, isHighRiskSafety, showToast, trimmed, validClock, withinLength } from '../../utils/ui'
import AppState from '../../components/AppState.vue'
import AppIcon from '../../components/AppIcon.vue'

const issue = ref('')
const styles = ['容易焦虑', '比较敏感', '沟通积极', '关注成绩']
const tones = ['温和', '正式', '简短', '坚定但礼貌']
const style = ref(styles[0])
const tone = ref(tones[0])
const drafts = ref([])
const parents = ref([])
const records = ref([])
const activeParent = ref(null)
const loading = ref(false)
const loadingParents = ref(false)
const loadingRecords = ref(false)
const generating = ref(false)
const saving = ref(false)
const error = ref('')
const aiFormError = ref('')
const parentFormError = ref('')
const recordFormError = ref('')
const parentQuery = ref('')
const recordQuery = ref('')
const pageSize = 20
const parentsOffset = ref(0)
const recordsOffset = ref(0)
const parentsHasMore = ref(false)
const recordsHasMore = ref(false)
const parentFormOpen = ref(false)
const recordFormOpen = ref(false)
const editingParentId = ref('')
const editingRecordId = ref('')
const parentForm = ref(defaultParentForm())
const recordForm = ref(defaultRecordForm())
const pendingAIRecordDraft = ref(null)
const importDialogOpen = ref(false)
const importFile = ref(null)
const importPreview = ref([])
const importErrors = ref([])
const importFailureCsv = ref('')
const importReadyCount = ref(0)
const importSummary = ref('')
const pendingAction = ref('')
const risks = [
  { value: 'low', label: '低风险' },
  { value: 'medium', label: '中风险' },
  { value: 'high', label: '高风险' }
]
const parentNames = computed(() => parents.value.map((item) => `${item.studentName} · ${item.parentName}`))
const hasPendingAction = computed(() => !!pendingAction.value)

onShow(async () => {
  if (!ensureLoggedIn(api)) return
  await load()
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    await loadParents({ reset: true })
    if (!activeParent.value && parents.value.length) activeParent.value = parents.value[0]
    await loadRecords({ reset: true })
  } catch (err) {
    error.value = errorMessage(err, '沟通数据加载失败')
  } finally {
    loading.value = false
  }
}

async function loadParents({ reset = false } = {}) {
  if (loadingParents.value) return
  loadingParents.value = true
  try {
    const offset = reset ? 0 : parentsOffset.value
    const response = await api.parents({ q: parentQuery.value, limit: pageSize, offset })
    const data = listItems(response)
    const page = listPageInfo(response, pageSize, offset)
    parents.value = reset ? data : [...parents.value, ...data]
    parentsOffset.value = page.nextOffset
    parentsHasMore.value = page.hasMore
    if (reset && activeParent.value && !parents.value.some((item) => item.id === activeParent.value.id)) {
      activeParent.value = parents.value[0] || null
    }
  } finally {
    loadingParents.value = false
  }
}

async function loadRecords({ reset = false } = {}) {
  if (loadingRecords.value) return
  loadingRecords.value = true
  try {
    const offset = reset ? 0 : recordsOffset.value
    const response = await api.records(activeParent.value?.id, { q: recordQuery.value, limit: pageSize, offset })
    const data = listItems(response)
    const page = listPageInfo(response, pageSize, offset)
    records.value = reset ? data : [...records.value, ...data]
    recordsOffset.value = page.nextOffset
    recordsHasMore.value = page.hasMore
  } finally {
    loadingRecords.value = false
  }
}

async function searchParents() {
  try {
    await loadParents({ reset: true })
  } catch (err) {
    showToast(errorMessage(err, '关注对象搜索失败'))
  }
}

async function searchRecords() {
  try {
    await loadRecords({ reset: true })
  } catch (err) {
    showToast(errorMessage(err, '沟通记录搜索失败'))
  }
}

async function loadMoreParents() {
  try {
    await loadParents()
  } catch (err) {
    showToast(errorMessage(err, '关注对象加载失败'))
  }
}

async function loadMoreRecords() {
  try {
    await loadRecords()
  } catch (err) {
    showToast(errorMessage(err, '沟通记录加载失败'))
  }
}

async function selectParent(parent) {
  activeParent.value = parent
  try {
    await loadRecords({ reset: true })
    showToast(`已切换到 ${parent.studentName}`)
  } catch (err) {
    showToast(errorMessage(err, '沟通记录加载失败'))
  }
}

function openParent(parent) {
  activeParent.value = parent
  uni.navigateTo({ url: `/pages/communication/parent-detail?id=${encodeURIComponent(parent.id)}` })
}

function openParentForm() {
  parentForm.value = defaultParentForm()
  parentFormError.value = ''
  recordFormError.value = ''
  editingParentId.value = ''
  parentFormOpen.value = true
  recordFormOpen.value = false
}

function editParent(parent) {
  if (!parent?.id) return
  editingParentId.value = parent.id
  parentForm.value = parentFormFromParent(parent)
  activeParent.value = parent
  parentFormError.value = ''
  recordFormError.value = ''
  parentFormOpen.value = true
  recordFormOpen.value = false
}

function openRecordForm(record = null) {
  const parent = activeParent.value || parents.value[0]
  editingRecordId.value = record?.id || ''
  recordForm.value = record ? recordFormFromRecord(record, parent) : defaultRecordForm(parent)
  parentFormError.value = ''
  recordFormError.value = ''
  recordFormOpen.value = true
  parentFormOpen.value = false
}

function closeForms() {
  parentFormOpen.value = false
  recordFormOpen.value = false
  editingParentId.value = ''
  editingRecordId.value = ''
  pendingAIRecordDraft.value = null
  parentFormError.value = ''
  recordFormError.value = ''
}

async function submitParent() {
  if (saving.value) return
  const validation = validateParentForm(parentForm.value)
  if (validation) {
    parentFormError.value = validation
    return
  }
  saving.value = true
  parentFormError.value = ''
  try {
    const parent = editingParentId.value
      ? await api.updateParent(editingParentId.value, cleanParentPayload(parentForm.value))
      : await api.createParent(cleanParentPayload(parentForm.value))
    const wasEditing = !!editingParentId.value
    parents.value = wasEditing
      ? parents.value.map((item) => item.id === parent.id ? parent : item)
      : [parent, ...parents.value]
    activeParent.value = parent
    closeForms()
    showToast(wasEditing ? '已更新家长档案' : '已新增家长档案')
  } catch (err) {
    parentFormError.value = errorMessage(err, '家长档案保存失败')
    showToast(parentFormError.value)
  } finally {
    saving.value = false
  }
}

async function removeParent(parent) {
  if (!parent?.id) return
  const action = `delete-parent:${parent.id}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmAction({
    title: '删除家长档案',
    content: `确认删除 ${parent.studentName || parent.parentName || '这位家长'} 的档案吗？关联沟通记录会保留为历史记录。`,
    confirmText: '删除'
  })
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  try {
    await api.deleteParent(parent.id)
    parents.value = parents.value.filter((item) => item.id !== parent.id)
    if (activeParent.value?.id === parent.id) {
      activeParent.value = parents.value[0] || null
      await loadRecords({ reset: true })
    }
    showToast('已删除家长档案')
  } catch (err) {
    showToast(errorMessage(err, '家长档案删除失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function importParents() {
  const action = 'import-parents'
  if (isActionPending(action)) return
  pendingAction.value = action
  try {
    const uploadFile = await chooseImportFile()
    importFile.value = uploadFile
    const result = await api.importParents(uploadFile, { preview: true })
    showImportPreview(result, '名单')
  } catch (err) {
    showToast(errorMessage(err, '名单预览失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function confirmImportParents() {
  const action = 'confirm-import-parents'
  if (saving.value || isActionPending(action) || !importFile.value) return
  saving.value = true
  pendingAction.value = action
  try {
    const result = await api.importParents(importFile.value)
    showImportPreview(result, '名单')
    showToast(`已导入 ${result.imported || 0} 位家长，跳过 ${result.skipped || 0} 条`)
    await load()
  } catch (err) {
    showToast(errorMessage(err, '名单导入失败'))
  } finally {
    saving.value = false
    clearPendingAction(action)
  }
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
  const student = values['学生姓名'] || values['学生'] || values.student || values.studentname || '未命名学生'
  const parent = values['家长姓名'] || values['家长'] || values.parent || values.parentname || '家长'
  const className = values['班级'] || values.class || values.classname || ''
  return `${student} · ${parent}${className ? ` · ${className}` : ''}`
}

function copyFailureCsv() {
  uni.setClipboardData({ data: importFailureCsv.value })
  showToast('已复制失败行 CSV')
}

function downloadParentTemplate() {
  downloadTemplate(api.parentImportTemplateURL(), '名单模板字段：学生姓名、班级、家长姓名、关系、联系方式、沟通风格、风险等级、重点备注、下一步')
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
      ...cleanRecordPayload(recordForm.value),
      followUpAt: recordFollowUpISO(recordForm.value),
      followUpStatus: recordForm.value.followUpStatus || 'pending',
      followedUpAt: recordForm.value.followedUpAt || null
    }
    const saved = editingRecordId.value
      ? await api.updateRecord(editingRecordId.value, payload)
      : await api.createRecord(payload)
    const wasEditing = !!editingRecordId.value
    const aiDraft = !wasEditing ? pendingAIRecordDraft.value : null
    records.value = editingRecordId.value
      ? records.value.map((item) => item.id === saved.id ? saved : item)
      : [saved, ...records.value]
    closeForms()
    showToast(wasEditing ? '已更新沟通记录' : '已添加沟通记录')
    if (aiDraft) {
      try {
        await auditAIAction(aiDraft, 'save_record', {
          note: '保存为沟通记录',
          metadata: { recordId: saved.id, surface: 'communication' }
        })
      } catch (err) {
        showToast(errorMessage(err, '记录已保存，AI 审计写入失败'))
      }
    }
  } catch (err) {
    recordFormError.value = errorMessage(err, '沟通记录保存失败')
    showToast(recordFormError.value)
  } finally {
    saving.value = false
  }
}

function editRecord(record) {
  openRecordForm(record)
}

async function removeRecord(id) {
  const action = `delete-record:${id}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmAction({
    title: '删除沟通记录',
    content: '删除后这条沟通记录将无法在当前列表中查看，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  try {
    await api.deleteRecord(id)
    records.value = records.value.filter((item) => item.id !== id)
    showToast('已删除沟通记录')
  } catch (err) {
    showToast(errorMessage(err, '删除沟通记录失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function completeRecordFollowUp(record) {
  const action = `complete-record:${record.id}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmAction({
    title: '完成跟进',
    content: `确认已完成 ${record.student || '这位学生'} 的本次跟进吗？`,
    confirmText: '完成'
  })
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  try {
    const updated = await api.completeRecordFollowUp(record.id)
    records.value = records.value.map((item) => item.id === updated.id ? updated : item)
    showToast('已标记为完成跟进')
  } catch (err) {
    showToast(errorMessage(err, '完成跟进失败'))
  } finally {
    clearPendingAction(action)
  }
}

async function generate() {
  if (generating.value) return
  if (!hasText(issue.value)) {
    aiFormError.value = '请先写下沟通背景'
    return
  }
  if (!withinLength(issue.value, 1200)) {
    aiFormError.value = '沟通背景最多 1200 个字'
    return
  }
  generating.value = true
  aiFormError.value = ''
  try {
    drafts.value = await api.parentDrafts({ issue: trimmed(issue.value), parentStyle: style.value, tone: tone.value, studentName: activeParent.value?.studentName || '' })
    showToast('已生成多个沟通版本')
  } catch (err) {
    aiFormError.value = errorMessage(err, '草稿生成失败')
    showToast(aiFormError.value)
  } finally {
    generating.value = false
  }
}

async function copyDraft(draft) {
  const action = `copy-draft:${draft?.id || ''}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmDraftAction(draft, '确认复制', '先不复制')
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  try {
    await auditAIAction(draft, 'copy', { note: '复制家校草稿', metadata: { surface: 'communication' } })
  } catch (err) {
    showToast(errorMessage(err, '审计记录保存失败，请稍后再复制'))
    clearPendingAction(action)
    return
  }
  copy(draft.content)
  clearPendingAction(action)
}

async function saveDraftAsRecord(draft) {
  const action = `save-draft:${draft?.id || ''}`
  if (isActionPending(action)) return
  pendingAction.value = action
  const confirmed = await confirmDraftAction(draft, '继续存记录', '先复核')
  if (!confirmed) {
    clearPendingAction(action)
    return
  }
  const parent = activeParent.value || parents.value[0] || null
  openRecordForm()
  recordForm.value = {
    ...recordForm.value,
    parentId: parent?.id || '',
    student: parent?.studentName || '',
    channel: '微信',
    reason: trimmed(issue.value) || 'AI 家校沟通草稿',
    summary: draft.content || '',
    result: draftRecordResult(draft),
    riskLevel: draftRecordRisk(draft, parent),
    ...followUpFields(defaultFollowUpDate())
  }
  pendingAIRecordDraft.value = draft
  recordFormError.value = ''
  showToast('已填入沟通记录，请复核后保存')
  clearPendingAction(action)
}

async function confirmDraftAction(draft, confirmText, cancelText) {
  if (!needsReview(draft)) return true
  const confirmed = await confirmAction({
    title: isHighRiskDraft(draft) ? '安全优先' : '需要人工复核',
    content: draftSafetyReason(draft),
    confirmText,
    cancelText
  })
  try {
    await auditAIAction(draft, confirmed ? 'review_confirmed' : 'review_skipped', {
      note: confirmed ? '教师确认复核后继续使用草稿' : '教师暂缓使用需复核草稿',
      metadata: {
        surface: 'communication',
        safety: draft?.safety || '',
        safetyLevel: draft?.safetyLevel || '',
        safetyReason: draftSafetyReason(draft),
        safetySignals: Array.isArray(draft?.safetySignals) ? draft.safetySignals : [],
        source: draft?.source || ''
      }
    })
  } catch (err) {
    showToast(errorMessage(err, 'AI 复核审计写入失败'))
  }
  return confirmed
}

function copy(content) {
  uni.setClipboardData({ data: content })
  showToast('已复制草稿')
}

async function auditAIAction(draft, action, options = {}) {
  if (!draft?.generationId) return null
  return api.createAIAction(draft.generationId, {
    action,
    draftId: draft.id || '',
    note: options.note || '',
    metadata: options.metadata || {}
  })
}

function isActionPending(action) {
  return pendingAction.value === action
}

function clearPendingAction(action) {
  if (pendingAction.value === action) pendingAction.value = ''
}

function isHighRiskDraft(draft) {
  return isHighRiskSafety(draft?.safety)
}

function needsReview(draft) {
  return !!draft?.reviewRequired || !!draft?.fallback || isHighRiskDraft(draft)
}

function aiSourceText(draft) {
  if (draft?.source === 'risk_guardrail') return '安全规则生成 · 请按流程复核'
  if (draft?.fallback) return '本地降级模板 · 模型暂不可用'
  if (draft?.provider === 'llm') return `模型生成 · ${draft.source || '实时生成'}`
  return '本地模板 · 适合调试和基础参考'
}

function reviewText(draft) {
  if (draft?.fallback) return '模型服务暂不可用，当前内容来自本地降级模板。请人工复核后再复制或发送。'
  if (isHighRiskDraft(draft)) return '该草稿涉及安全或专业边界，请先人工复核并按学校流程处理。'
  return 'AI 草稿仅供教师复核，发送前需要结合真实情境人工确认。'
}

function draftSafetyReason(draft) {
  return draft?.safetyReason || draft?.safetyNote || reviewText(draft)
}

function draftSafetySignals(draft) {
  if (!Array.isArray(draft?.safetySignals) || draft.safetySignals.length === 0) return ''
  return draft.safetySignals.join('、')
}

function safetyText(value) {
  return ({
    teacher_review_required: '需教师审核',
    crisis_support_required: '危机支持',
    student_safety_review_required: '学生安全',
    medical_review_required: '专业复核',
    self_care: '自我照护'
  })[value] || '需教师审核'
}

function riskText(value) {
  return ({ low: '低风险', medium: '中风险', high: '高风险' })[value] || '低风险'
}

function draftRecordResult(draft) {
  return `${safetyText(draft?.safety)} · ${aiSourceText(draft)}。发送前已提示人工复核。`
}

function draftRecordRisk(draft, parent) {
  if (isHighRiskDraft(draft)) return 'high'
  if (draft?.fallback || draft?.reviewRequired) return parent?.riskLevel || 'medium'
  return parent?.riskLevel || 'low'
}

function pickRecordParent(event) {
  const parent = parents.value[event.detail.value]
  if (!parent) return
  activeParent.value = parent
  recordForm.value.parentId = parent.id
  recordForm.value.student = parent.studentName
  recordForm.value.riskLevel = parent.riskLevel || 'low'
}

function defaultParentForm() {
  return {
    studentName: '',
    className: '',
    parentName: '',
    relationship: '家长',
    contact: '',
    communicationStyle: '沟通积极',
    riskLevel: 'low',
    importantNotes: '',
    nextAction: ''
  }
}

function parentFormFromParent(parent = {}) {
  return {
    studentName: parent.studentName || '',
    className: parent.className || '',
    parentName: parent.parentName || '',
    relationship: parent.relationship || '家长',
    contact: parent.contact || '',
    communicationStyle: parent.communicationStyle || '沟通积极',
    riskLevel: parent.riskLevel || 'low',
    importantNotes: parent.importantNotes || '',
    nextAction: parent.nextAction || ''
  }
}

function defaultRecordForm(parent = null) {
  return {
    parentId: parent?.id || '',
    student: parent?.studentName || '',
    channel: '微信',
    reason: '',
    summary: '',
    result: '',
    riskLevel: parent?.riskLevel || 'low',
    followUpStatus: 'pending',
    followedUpAt: null,
    ...followUpFields(defaultFollowUpDate())
  }
}

function recordFormFromRecord(record, parent = null) {
	return {
		parentId: record.parentId || '',
		student: record.student || parent?.studentName || '',
		channel: record.channel || '微信',
    reason: record.reason || '',
    summary: record.summary || '',
    result: record.result || '',
    riskLevel: record.riskLevel || parent?.riskLevel || 'low',
    followUpStatus: record.followUpStatus || 'pending',
    followedUpAt: record.followedUpAt || null,
    ...followUpFields(record.followUpAt || defaultFollowUpDate())
  }
}

function canCompleteFollowUp(record) {
  return !!record?.id && record.followUpStatus !== 'done'
}

function followUpStatusText(record) {
  if (record?.followUpStatus === 'done') {
    const doneAt = formatDateTime(record.followedUpAt)
    return doneAt ? `已跟进 · ${doneAt}` : '已跟进'
  }
  const followAt = formatDateTime(record?.followUpAt)
  return followAt ? `待跟进 · ${followAt}` : '未设置跟进时间'
}

function validateParentForm(form) {
  if (!hasText(form.studentName) || !hasText(form.className) || !hasText(form.parentName) || !hasText(form.relationship)) return '请补全学生、班级、家长称呼和关系'
  if (!withinLength(form.studentName, 60) || !withinLength(form.parentName, 80)) return '学生姓名最多 60 个字，家长称呼最多 80 个字'
  if (!withinLength(form.className, 80) || !withinLength(form.relationship, 40)) return '班级最多 80 个字，关系最多 40 个字'
  if (!withinLength(form.contact, 80) || !withinLength(form.communicationStyle, 80)) return '联系方式和沟通风格最多 80 个字'
  if (!withinLength(form.importantNotes, 2000)) return '重点备注最多 2000 个字'
  if (!withinLength(form.nextAction, 1000)) return '下一步跟进最多 1000 个字'
  return ''
}

function validateRecordForm(form) {
  if (!hasText(form.student) || !hasText(form.channel) || !hasText(form.reason) || !hasText(form.summary)) return '请补全学生、渠道、原因和摘要'
  if (!withinLength(form.student, 60) || !withinLength(form.channel, 40)) return '学生姓名最多 60 个字，渠道最多 40 个字'
  if (!withinLength(form.reason, 200)) return '沟通原因最多 200 个字'
  if (!withinLength(form.summary, 2000) || !withinLength(form.result, 2000)) return '沟通摘要和结果最多 2000 个字'
  if (!validISODate(form.followUpDate)) return '跟进日期请使用 2026-05-30 这样的格式'
  if (!validClock(form.followUpTime)) return '跟进时间请使用 17:00 这样的 24 小时格式'
  return ''
}

function cleanParentPayload(form) {
  return {
    ...form,
    studentName: trimmed(form.studentName),
    className: trimmed(form.className),
    parentName: trimmed(form.parentName),
    relationship: trimmed(form.relationship),
    contact: trimmed(form.contact),
    communicationStyle: trimmed(form.communicationStyle),
    importantNotes: trimmed(form.importantNotes),
    nextAction: trimmed(form.nextAction)
  }
}

function cleanRecordPayload(form) {
  return {
    parentId: form.parentId || null,
    student: trimmed(form.student),
    channel: trimmed(form.channel),
    reason: trimmed(form.reason),
    summary: trimmed(form.summary),
    result: trimmed(form.result),
    riskLevel: form.riskLevel || 'low'
  }
}

function recordFollowUpISO(form) {
  return new Date(`${trimmed(form.followUpDate)}T${trimmed(form.followUpTime)}:00`).toISOString()
}

function formatDateTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
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
.header, .section-head { padding: 0 4rpx 14rpx; }
.avatar-btn { width: 88rpx; height: 88rpx; padding: 0; border-radius: 28rpx; background: linear-gradient(180deg, rgba(255,255,255,.96), rgba(250,253,255,.86)); border: 1px solid rgba(97,116,166,.14); box-shadow: 0 10rpx 22rpx rgba(73,91,146,.10), inset 0 1px 0 rgba(255,255,255,.96); display: flex; align-items: center; justify-content: center; }
.picker-row { margin: 20rpx 0; }
.picker { flex: 1; }
.draft { border: 2rpx solid rgba(169,144,234,.28); }
.parent-card { display: flex; flex-direction: column; gap: 18rpx; }
.parent-title { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6rpx; }
.half { flex: 1; min-width: 0; }
.small { min-height: 62rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 54rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
.success { color: #059669; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
.search-bar { display: flex; gap: 12rpx; align-items: center; }
.search-input { flex: 1; min-width: 0; }
.load-more { margin: 8rpx 0 28rpx; }
.state-card { display: flex; flex-direction: column; gap: 18rpx; }
.compact-state { margin: 0; padding: 22rpx; }
.preview-row { padding: 22rpx; border-radius: 22rpx; background: rgba(255,255,255,.58); border: 1rpx solid rgba(255,255,255,.78); display: flex; flex-direction: column; gap: 8rpx; }
.preview-row.invalid { background: rgba(255,241,242,.88); }
	.dangerTag { background: rgba(255,241,242,.95); color: #b95c61; }
.safety-note { display: block; padding: 18rpx 20rpx; border-radius: 20rpx; background: rgba(255,241,242,.88); color: #9f4b52; font-size: 24rpx; line-height: 1.5; font-weight: 800; }
.retry { align-self: flex-start; }
.form-grid { display: flex; flex-direction: column; gap: 18rpx; }
.textarea.short { min-height: 120rpx; }
.risk-row { display: flex; gap: 14rpx; }
.follow-row { align-items: stretch; }
.risk-btn { flex: 1; min-height: 68rpx; border-radius: 22rpx; background: rgba(255,255,255,.64); color: #506075; font-size: 24rpx; font-weight: 900; border: 1rpx solid rgba(255,255,255,.78); }
.risk-btn.active { color: #fff; background: linear-gradient(135deg,#6f86df,#52b8cf); }
</style>
