<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">{{ formatTime(log.createdAt) || 'AI 审计记录' }}</text>
        <view class="title">{{ scenarioText(log.scenario) }}</view>
      </view>
      <view class="row action-row">
        <button class="ghost-btn small" @tap="load">刷新</button>
        <button v-if="log.id" class="ghost-btn small danger" :disabled="deleting" @tap="deleteLog">
          {{ deleting ? '删除中...' : '删除' }}
        </button>
      </view>
    </view>

    <AppState v-if="loading" type="loading" message="正在加载 AI 记录..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <AppState
      v-if="!loading && !error && !log.id"
      type="empty"
      title="没有找到这条记录"
      message="可能已被清理，或当前账号没有访问权限。"
    />

    <view v-if="log.id" class="card status-card" :class="{ danger: isHighRisk(log.safetyLabel) }">
      <view class="row between">
        <text class="section-title">安全标记</text>
        <text class="tag">{{ safetyText(log.safetyLabel) }}</text>
      </view>
      <text class="body">{{ safetyNote(log.safetyLabel) }}</text>
    </view>

    <view v-if="log.id" class="card meta-card">
      <view class="meta-row">
        <text class="caption">场景</text>
        <text class="body">{{ scenarioText(log.scenario) }}</text>
      </view>
      <view class="meta-row">
        <text class="caption">生成时间</text>
        <text class="body">{{ formatTime(log.createdAt) || '未知' }}</text>
      </view>
      <view class="meta-row">
        <text class="caption">Token</text>
        <text class="body">{{ log.tokenUsage || 0 }}</text>
      </view>
    </view>

    <view v-if="inputItems.length" class="section-head row between">
      <text class="section-title">输入</text>
      <text class="caption">{{ inputItems.length }} 项</text>
    </view>
    <view v-for="item in inputItems" :key="item.key" class="card detail-card">
      <text class="tag">{{ inputLabel(item.key) }}</text>
      <text class="body detail-text">{{ item.value }}</text>
    </view>

    <view v-if="drafts.length" class="section-head row between">
      <text class="section-title">输出</text>
      <button class="ghost-btn small" @tap="copyAllOutputs">复制全部</button>
    </view>
    <view v-for="draft in drafts" :key="draft.key" class="card detail-card" :class="{ danger: isHighRisk(draft.safety) }">
      <view class="row between">
        <text class="tag">{{ draft.version || draft.key }}</text>
        <text class="caption">{{ safetyText(draft.safety) }}</text>
      </view>
      <text v-if="sourceText(draft)" class="caption block">{{ sourceText(draft) }}</text>
      <view v-if="draftSafetyReason(draft)" class="safety-note">
        <text class="block">{{ draftSafetyReason(draft) }}</text>
        <text v-if="draftSafetySignals(draft)" class="caption block">触发信号：{{ draftSafetySignals(draft) }}</text>
      </view>
      <text class="body detail-text">{{ draft.content }}</text>
      <button class="ghost-btn small copy-btn" @tap="copyDraft(draft)">复制这条</button>
    </view>

    <view v-if="actions.length" class="section-head row between">
      <text class="section-title">使用记录</text>
      <text class="caption">{{ actions.length }} 次</text>
    </view>
    <view v-for="action in actions" :key="action.id" class="card action-card">
      <view class="row between">
        <text class="tag">{{ actionText(action.action) }}</text>
        <text class="caption">{{ formatTime(action.createdAt) }}</text>
      </view>
      <text v-if="action.note" class="body">{{ action.note }}</text>
      <text v-if="action.draftId" class="caption block">草稿：{{ action.draftId }}</text>
    </view>

    <view v-if="rawOutputText" class="section-head row between">
      <text class="section-title">原始输出</text>
      <button class="ghost-btn small" @tap="copyText(rawOutputText)">复制</button>
    </view>
    <view v-if="rawOutputText" class="card raw-card">
      <text class="raw-text">{{ rawOutputText }}</text>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { confirmAction, ensureLoggedIn, errorMessage, isHighRiskSafety, showToast } from '../../utils/ui'
import AppState from '../../components/AppState.vue'

const id = ref('')
const log = ref({})
const loading = ref(false)
const error = ref('')
const deleting = ref(false)

const inputItems = computed(() => objectItems(log.value.input))
const drafts = computed(() => outputDrafts(log.value.output))
const actions = computed(() => Array.isArray(log.value.actions) ? log.value.actions : [])
const rawOutputText = computed(() => {
  if (!log.value.output) return ''
  if (drafts.value.length) return ''
  return stringify(log.value.output)
})

onLoad(async (query) => {
  id.value = query?.id || ''
  await load()
})

async function load() {
  if (!ensureLoggedIn(api)) return
  if (!id.value) {
    error.value = '缺少 AI 记录 ID'
    return
  }
  loading.value = true
  error.value = ''
  try {
    log.value = await api.aiGeneration(id.value)
  } catch (err) {
    error.value = errorMessage(err, 'AI 记录加载失败')
  } finally {
    loading.value = false
  }
}

async function deleteLog() {
  if (deleting.value || !log.value.id) return
  const confirmed = await confirmAction({
    title: '删除 AI 记录',
    content: '删除后这条 AI 输入、输出和使用审计会从当前账号移除，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) return
  deleting.value = true
  try {
    await api.deleteAIGeneration(log.value.id)
    showToast('已删除 AI 记录')
    uni.navigateBack({
      fail: () => uni.switchTab({ url: '/pages/profile/index' })
    })
  } catch (err) {
    showToast(errorMessage(err, 'AI 记录删除失败'))
  } finally {
    deleting.value = false
  }
}

function objectItems(value) {
  if (!value || typeof value !== 'object') return []
  return Object.entries(value)
    .filter(([, item]) => item !== undefined && item !== null && item !== '')
    .map(([key, item]) => ({ key, value: stringify(item) }))
}

function outputDrafts(output) {
  if (!output || typeof output !== 'object') return []
  if (Array.isArray(output.drafts)) return output.drafts.map((item, index) => normalizeDraft(item, `版本 ${index + 1}`))
  if (output.draft) return [normalizeDraft(output.draft, '生成内容')]
  if (output.content) return [normalizeDraft(output, '生成内容')]
  return []
}

function normalizeDraft(item, fallbackKey) {
  return {
    key: item?.id || fallbackKey,
    id: item?.id || '',
    generationId: item?.generationId || log.value.id || '',
    version: item?.version || fallbackKey,
    content: item?.content || stringify(item),
    safety: item?.safety || log.value.safetyLabel,
    provider: item?.provider || '',
    source: item?.source || '',
    fallback: !!item?.fallback,
    reviewRequired: !!item?.reviewRequired,
    safetyNote: item?.safetyNote || '',
    safetyLevel: item?.safetyLevel || '',
    safetyReason: item?.safetyReason || '',
    safetySignals: Array.isArray(item?.safetySignals) ? item.safetySignals : []
  }
}

function sourceText(draft) {
  const parts = []
  if (draft.provider) parts.push(providerText(draft.provider))
  if (draft.source) parts.push(draft.source)
  if (draft.fallback) parts.push('降级生成')
  if (draft.reviewRequired) parts.push('需复核')
  return parts.join(' · ')
}

function providerText(value) {
  return ({ llm: '模型生成', mock: '本地模板', safety_rules: '安全规则' })[value] || value
}

function scenarioText(scenario) {
  return ({ parent_drafts: '家校草稿', praise: 'AI夸夸' })[scenario] || 'AI 生成'
}

function inputLabel(key) {
  return ({ issue: '沟通背景', parentStyle: '家长风格', tone: '语气', studentName: '学生', persona: '角色', content: '输入内容', mood: '心情' })[key] || key
}

function safetyText(value) {
  return ({
    self_care: '自我照护',
    teacher_review_required: '需教师审核',
    crisis_support_required: '危机支持',
    student_safety_review_required: '学生安全',
    medical_review_required: '专业复核'
  })[value] || value || '未标记'
}

function safetyNote(value) {
  return ({
    self_care: '适合作为自我照护参考，仍建议结合当下状态判断。',
    teacher_review_required: '发送或保存前需要教师结合真实情境复核。',
    crisis_support_required: '涉及危机信号，请优先确保安全并联系可信任的人、学校负责人或当地紧急救助渠道。',
    student_safety_review_required: '涉及学生安全或暴力风险，请先按学校流程联系负责人或专业支持。',
    medical_review_required: '涉及医学或心理专业边界，请避免自行诊断或给出用药建议。'
  })[value] || '这条记录暂无额外安全说明。'
}

function draftSafetyReason(draft) {
  return draft?.safetyReason || draft?.safetyNote || ''
}

function draftSafetySignals(draft) {
  if (!Array.isArray(draft?.safetySignals) || draft.safetySignals.length === 0) return ''
  return draft.safetySignals.join('、')
}

function isHighRisk(value) {
  return isHighRiskSafety(value)
}

async function copyAllOutputs() {
  const readyDrafts = drafts.value.filter((item) => item.content)
  if (!readyDrafts.length) return
  for (const draft of readyDrafts) {
    const confirmed = await confirmDraftReview(draft)
    if (!confirmed) return
  }
  for (const draft of readyDrafts) {
    const audited = await auditCopyAction(draft, '从 AI 记录详情复制全部')
    if (!audited) return
  }
  copyText(readyDrafts.map((item) => item.content).join('\n\n'))
}

async function copyDraft(draft) {
  if (!draft?.content) return
  const confirmed = await confirmDraftReview(draft)
  if (!confirmed) return
  if (draft.generationId) {
    const audited = await auditCopyAction(draft, '从 AI 记录详情复制')
    if (!audited) return
  }
  copyText(draft.content)
}

async function auditCopyAction(draft, note) {
  const generationId = draft?.generationId || log.value.id
  if (!generationId) return true
  try {
    await api.createAIAction(generationId, {
      action: 'copy',
      draftId: draft.id || draft.key || '',
      note,
      metadata: { surface: 'ai_log_detail' }
    })
    await load()
    return true
  } catch (err) {
    showToast(errorMessage(err, '审计记录保存失败，请稍后再复制'))
    return false
  }
}

async function confirmDraftReview(draft) {
  if (!needsReview(draft)) return true
  const confirmed = await confirmAction({
    title: isHighRisk(draft.safety) ? '安全优先' : '需要人工复核',
    content: draftSafetyReason(draft) || safetyNote(draft.safety),
    confirmText: '确认复核',
    cancelText: '先不复制'
  })
  try {
    await api.createAIAction(draft.generationId || log.value.id, {
      action: confirmed ? 'review_confirmed' : 'review_skipped',
      draftId: draft.id || draft.key || '',
      note: confirmed ? '从 AI 记录详情确认复核' : '从 AI 记录详情暂缓复制',
      metadata: {
        surface: 'ai_log_detail',
        safety: draft.safety || '',
        safetyLevel: draft.safetyLevel || '',
        safetyReason: draftSafetyReason(draft),
        safetySignals: Array.isArray(draft.safetySignals) ? draft.safetySignals : [],
        source: draft.source || ''
      }
    })
    await load()
  } catch (err) {
    showToast(errorMessage(err, 'AI 复核审计写入失败'))
    return false
  }
  return confirmed
}

function needsReview(draft) {
  return !!draft?.reviewRequired || !!draft?.fallback || isHighRisk(draft?.safety) || draft?.safety === 'teacher_review_required' || draft?.safetyLevel === 'review'
}

function copyText(value) {
  if (!value) return
  uni.setClipboardData({ data: value })
  showToast('已复制')
}

function actionText(value) {
  return ({
    copy: '已复制',
    save_record: '已存沟通记录',
    save_healing: '已存疗愈记录',
    review_confirmed: '已确认复核',
    review_skipped: '跳过复核'
  })[value] || value || '使用动作'
}

function stringify(value) {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch (err) {
    return String(value)
  }
}

function formatTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
	.header, .section-head { padding: 0 32rpx 12rpx; }
	.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
	.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
	.danger { color: #b95c61; }
	.detail-card, .status-card, .action-card { display: flex; flex-direction: column; gap: 18rpx; }
.meta-card { display: flex; flex-direction: column; gap: 16rpx; }
.meta-row { display: flex; justify-content: space-between; gap: 24rpx; align-items: flex-start; }
.detail-text { white-space: pre-wrap; word-break: break-word; }
.block { display: block; }
	.status-card.danger, .detail-card.danger { background: rgba(255,241,242,.9); }
.safety-note { display: block; padding: 18rpx 20rpx; border-radius: 20rpx; background: rgba(255,255,255,.72); color: #9f4b52; font-size: 24rpx; line-height: 1.5; font-weight: 800; }
.copy-btn { align-self: flex-start; }
.raw-card { background: rgba(247,252,253,.92); }
.raw-text { font-size: 22rpx; line-height: 1.5; color: #415267; white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
</style>
