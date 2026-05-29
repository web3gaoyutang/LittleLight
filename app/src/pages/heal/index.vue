<template>
  <view class="page-wrap">
    <view class="header"><text class="caption">呼吸 · AI夸夸 · 声音</text><view class="title">短恢复</view></view>
    <view class="breath card">
      <text class="section-title">一分钟呼吸</text>
      <text class="caption">把注意力交还给自己</text>
      <view class="breath-core" :class="{ active: breathing }"><view class="inner">≋</view></view>
      <button class="primary-btn breath-btn" @tap="toggleBreath">{{ breathing ? `呼吸中 ${leftText}` : '开始 01:00' }}</button>
    </view>
    <view class="card">
      <view class="row between"><view><text class="section-title">AI夸夸</text><text class="caption block">写下今天发生了什么</text></view><text class="tag" :class="{ dangerTag: praiseHighRisk }">{{ praiseSafetyText }}</text></view>
      <textarea class="textarea" v-model="content" maxlength="1000" placeholder="比如：今天课很多，还处理了家长反馈..." aria-label="AI 夸夸内容" data-testid="praise-content-input" />
      <button class="primary-btn praise-btn" :disabled="saving" @tap="makePraise">{{ saving ? '生成中...' : '生成一句抱抱' }}</button>
	      <view class="reply" :class="{ dangerReply: praiseNeedsReview }">
	        <text class="caption block">{{ praiseSourceText }}</text>
	        <view v-if="praiseNeedsReview" class="safety-note">
	          <text class="block">{{ praiseSafetyNote }}</text>
	          <text v-if="praiseSafetySignals" class="caption block">触发信号：{{ praiseSafetySignals }}</text>
	        </view>
	        <text class="body">{{ reply }}</text>
	      </view>
    </view>

    <AppState v-if="loading" type="loading" message="正在加载疗愈记录..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="loadEntries" />

    <view class="section-head row between">
      <text class="section-title">疗愈记录</text>
      <text class="tag">{{ entries.length }} 条</text>
    </view>
    <view class="search-bar">
      <input class="input search-input" v-model="entryQuery" confirm-type="search" placeholder="搜索记录、心情或 AI 回复" aria-label="搜索疗愈记录" data-testid="healing-search-input" @confirm="searchEntries" />
      <button class="ghost-btn mini" :disabled="loadingMore" @tap="searchEntries">搜索</button>
    </view>
    <view v-for="entry in entries" :key="entry.id" class="card entry-card">
      <view class="row between">
        <view>
          <text class="tag">{{ entryLabel(entry.type) }}</text>
          <text class="caption block">{{ formatTime(entry.createdAt) }}</text>
        </view>
        <button class="ghost-btn mini danger" @tap="removeEntry(entry.id)">删除</button>
      </view>
      <text class="body">{{ entry.aiReply || entry.content || '完成一次短恢复' }}</text>
      <text v-if="entry.content && entry.aiReply" class="caption block source-text">{{ entry.content }}</text>
    </view>
    <AppState
      v-if="!loading && !error && entries.length === 0"
      type="empty"
      title="还没有疗愈记录"
      message="完成一次呼吸或生成一句 AI 夸夸，都会留在这里。"
    />
    <button v-if="hasMore" class="ghost-btn load-more" :disabled="loadingMore" @tap="loadMoreEntries">
      {{ loadingMore ? '加载中...' : '加载更多疗愈记录' }}
    </button>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api, listItems, listPageInfo } from '../../api/client'
import { confirmAction, ensureLoggedIn, errorMessage, isHighRiskSafety, showToast, trimmed } from '../../utils/ui'
import AppState from '../../components/AppState.vue'

const breathing = ref(false)
const left = ref(60)
let timer = null
const content = ref('')
const reply = ref('你今天已经完成了很多复杂判断。先把肩膀放下来，剩下的事情可以一件一件处理。')
const entries = ref([])
const loading = ref(false)
const loadingMore = ref(false)
const saving = ref(false)
const error = ref('')
const entryQuery = ref('')
const praiseSafety = ref('self_care')
const praiseMeta = ref({})
const pageSize = 20
const offset = ref(0)
const hasMore = ref(false)
const leftText = computed(() => `00:${String(left.value).padStart(2, '0')}`)
const praiseHighRisk = computed(() => isHighRiskSafety(praiseSafety.value))
const praiseNeedsReview = computed(() => praiseHighRisk.value || !!praiseMeta.value.reviewRequired || !!praiseMeta.value.fallback)
const praiseSafetyText = computed(() => safetyText(praiseSafety.value))
const praiseSafetyNote = computed(() => praiseMeta.value.safetyReason || praiseMeta.value.safetyNote || safetyNote(praiseSafety.value, praiseMeta.value))
const praiseSafetySignals = computed(() => Array.isArray(praiseMeta.value.safetySignals) ? praiseMeta.value.safetySignals.join('、') : '')
const praiseSourceText = computed(() => aiSourceText(praiseMeta.value))

onShow(loadEntries)

async function loadEntries() {
  if (!ensureLoggedIn(api)) return
  loading.value = true
  error.value = ''
  try {
    await fetchEntries({ reset: true })
  } catch (err) {
    error.value = errorMessage(err, '疗愈记录加载失败')
  } finally {
    loading.value = false
  }
}

async function fetchEntries({ reset = false } = {}) {
  if (loadingMore.value) return
  loadingMore.value = true
  try {
    const currentOffset = reset ? 0 : offset.value
    const response = await api.healingEntries('', { q: entryQuery.value, limit: pageSize, offset: currentOffset })
    const data = listItems(response)
    const page = listPageInfo(response, pageSize, currentOffset)
    entries.value = reset ? data : [...entries.value, ...data]
    offset.value = page.nextOffset
    hasMore.value = page.hasMore
  } finally {
    loadingMore.value = false
  }
}

async function searchEntries() {
  try {
    await fetchEntries({ reset: true })
  } catch (err) {
    showToast(errorMessage(err, '疗愈记录搜索失败'))
  }
}

async function loadMoreEntries() {
  try {
    await fetchEntries()
  } catch (err) {
    showToast(errorMessage(err, '疗愈记录加载失败'))
  }
}

function toggleBreath() {
  if (breathing.value) {
    clearInterval(timer); breathing.value = false; left.value = 60; return
  }
  breathing.value = true; left.value = 60
  timer = setInterval(() => {
    left.value -= 1
    if (left.value <= 0) {
      completeBreath()
    }
  }, 1000)
}

async function completeBreath() {
  clearInterval(timer)
  breathing.value = false
  left.value = 60
  try {
    const entry = await api.healingEntry({
      type: 'breath',
      mood: 'calm',
      content: '完成 1 分钟呼吸练习',
      aiReply: '已完成一次短恢复。'
    })
    entries.value.unshift(entry)
    showToast('一分钟呼吸完成')
  } catch (err) {
    showToast(errorMessage(err, '呼吸记录保存失败'))
  }
}

async function makePraise() {
  if (saving.value) return
  const input = trimmed(content.value)
  if (!input) {
    showToast('先写一点今天发生的事')
    return
  }
  saving.value = true
  try {
	    const data = await api.praise({ persona: '温柔前辈', content: input })
	    reply.value = data.content
	    praiseSafety.value = data.safety || 'self_care'
	    praiseMeta.value = data || {}
	    if (praiseNeedsReview.value) {
	      const confirmed = await confirmAction({
	        title: '需要优先保障安全',
	        content: praiseSafetyNote.value,
	        confirmText: '保存记录',
	        cancelText: '只查看'
	      })
      await auditReviewAction(data, confirmed)
      if (!confirmed) {
        showToast('已生成安全提示，未保存记录')
        return
      }
    }
    const entry = await api.healingEntry({ type: 'praise', mood: 'warm', content: input, aiReply: data.content })
    entries.value.unshift(entry)
    showToast(praiseHighRisk.value ? '已保存安全提示记录' : '已生成新的 AI 夸夸')
    try {
      await auditAIAction(data, 'save_healing', {
        note: praiseHighRisk.value ? '保存安全提示记录' : '保存 AI 夸夸记录',
        metadata: { entryId: entry.id, surface: 'healing' }
      })
    } catch (err) {
      showToast(errorMessage(err, '记录已保存，AI 审计写入失败'))
    }
  } catch (err) {
    showToast(errorMessage(err, 'AI 夸夸生成失败'))
  } finally {
    saving.value = false
  }
}

async function removeEntry(id) {
  const confirmed = await confirmAction({
    title: '删除疗愈记录',
    content: '删除后这条短恢复记录会从历史中移除，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) return
  try {
    await api.deleteHealingEntry(id)
    entries.value = entries.value.filter((item) => item.id !== id)
    showToast('已删除疗愈记录')
  } catch (err) {
    showToast(errorMessage(err, '删除疗愈记录失败'))
  }
}

function entryLabel(type) {
  const labels = { breath: '呼吸', praise: 'AI夸夸', treehole: '树洞', sound: '声音' }
  return labels[type] || '记录'
}

function safetyText(value) {
  return ({
    self_care: '抱抱',
    teacher_review_required: '需复核',
    crisis_support_required: '安全优先',
    student_safety_review_required: '安全优先',
    medical_review_required: '专业支持'
  })[value] || '抱抱'
}

function safetyNote(value, meta = {}) {
  if (meta.fallback) return '模型服务暂不可用，当前内容来自本地降级模板。请只作为自我照护参考。'
  return ({
    crisis_support_required: '这已经超出普通情绪鼓励范围。请先确保自己安全，尽快联系身边可信任的人、学校负责人或当地紧急救助渠道。',
    student_safety_review_required: '这涉及学生安全或暴力风险，请先按学校流程联系负责人或专业支持，再做后续处理。',
    medical_review_required: '这涉及医学或心理专业边界，请避免自行判断诊断或用药，优先联系专业支持。'
  })[value] || '这条内容需要先人工复核，再决定是否保存或继续使用。'
}

function aiSourceText(meta = {}) {
  if (meta.source === 'risk_guardrail') return '安全规则生成 · 请优先处理安全'
  if (meta.fallback) return '本地降级模板 · 模型暂不可用'
  if (meta.provider === 'llm') return `模型生成 · ${meta.source || '实时生成'}`
  return '本地模板 · 适合调试和基础参考'
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

async function auditReviewAction(draft, confirmed) {
  try {
    await auditAIAction(draft, confirmed ? 'review_confirmed' : 'review_skipped', {
      note: confirmed ? '教师确认复核后保存疗愈记录' : '教师只查看需复核内容',
      metadata: {
        surface: 'healing',
        safety: draft?.safety || '',
        safetyLevel: draft?.safetyLevel || '',
        safetyReason: draft?.safetyReason || draft?.safetyNote || '',
        safetySignals: Array.isArray(draft?.safetySignals) ? draft.safetySignals : [],
        source: draft?.source || ''
      }
    })
  } catch (err) {
    showToast(errorMessage(err, 'AI 复核审计写入失败'))
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
.block { display: block; }
.breath { min-height: 720rpx; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 28rpx; }
.breath-core { width: 300rpx; height: 300rpx; border-radius: 999rpx; display: flex; align-items: center; justify-content: center; background: rgba(121,144,255,.14); }
.breath-core.active { animation: breathe 7.2s ease-in-out infinite; }
.inner { width: 150rpx; height: 150rpx; border-radius: 999rpx; background: linear-gradient(135deg,#0891b2,#059669); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 42rpx; }
.breath-btn, .praise-btn { width: 100%; margin-top: 24rpx; }
.reply { margin-top: 24rpx; padding: 28rpx; border-radius: 30rpx; background: rgba(255,255,255,.68); display: flex; flex-direction: column; gap: 16rpx; }
.dangerReply { background: rgba(255,241,242,.88); }
.dangerTag { background: rgba(255,241,242,.95); color: #b95c61; }
.safety-note { display: block; padding: 18rpx 20rpx; border-radius: 20rpx; background: rgba(255,255,255,.72); color: #9f4b52; font-size: 24rpx; line-height: 1.5; font-weight: 800; }
.entry-card { display: flex; flex-direction: column; gap: 18rpx; }
.source-text { margin-top: 6rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
.search-bar { padding: 0 32rpx 12rpx; display: flex; gap: 12rpx; align-items: center; }
.search-input { flex: 1; min-width: 0; }
.load-more { margin: 8rpx 32rpx 28rpx; }
@keyframes breathe { 0%,100% { transform: scale(.86); opacity:.9 } 50% { transform: scale(1.08); opacity:.52 } }
</style>
