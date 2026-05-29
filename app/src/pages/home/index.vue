<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">{{ summary.todayLabel || todayLabel }}</text>
        <view class="title">早上好，{{ profile.name || '微光老师' }}</view>
      </view>
      <button class="bell" data-testid="home-schedule-button" @tap="goSchedule">⌁</button>
    </view>

    <view class="hero card">
      <text class="tag">今日教学心流</text>
      <view class="hero-title">{{ summary.rhythm?.title || '温柔但高效' }}</view>
      <text class="body">{{ summary.rhythm?.description || '先把节奏拆小，保持一点余裕。' }}</text>
      <view class="row actions">
        <button class="primary-btn action" @tap="goCommunication">AI 助手</button>
        <button class="ghost-btn action" data-testid="home-today-schedule-button" @tap="goSchedule">今日安排</button>
      </view>
    </view>

    <AppState v-if="loading" type="loading" message="正在整理今天的安排..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="section-head row between">
      <text class="section-title">今日状态</text>
      <text class="caption">低打扰工作台</text>
    </view>
    <view class="mood-row">
      <view v-for="item in moods" :key="item.code" class="mood-chip" :class="{ active: item.code === mood }" @tap="mood = item.code">
        <text class="mood-icon">{{ item.icon }}</text>
        <text>{{ item.label }}</text>
      </view>
    </view>

    <view class="bento">
      <view class="tile blue" data-testid="home-course-tile" @tap="goSchedule"><text class="tag">课程</text><text class="num">{{ summary.coursesCount || 0 }}</text><text class="caption">今日课程</text></view>
      <view class="tile mint" data-testid="home-reminder-tile" @tap="goSchedule"><text class="tag">待办</text><text class="num">{{ summary.remindersCount || 0 }}</text><text class="caption">待处理提醒</text></view>
      <view class="tile pink" @tap="goCommunication"><text class="tag">AI</text><text class="tile-title">生成家校回复</text><text class="caption">多语气版本</text></view>
      <view class="tile peach" @tap="goHeal"><text class="tag">恢复</text><text class="tile-title">一分钟呼吸</text><text class="caption">先收回注意力</text></view>
    </view>

    <view class="card" v-if="summary.nextCourse">
      <view class="row between"><text class="section-title">下一项</text><text class="tag">{{ summary.nextCourse.startTime }}</text></view>
      <view class="course-title">{{ summary.nextCourse.className }} · {{ summary.nextCourse.title }}</view>
      <text class="body">{{ summary.nextCourse.note }} · {{ summary.nextCourse.location }}</text>
    </view>
    <AppState
      v-else-if="!loading && !error"
      type="empty"
      title="今天还没有课程"
      message="可以先去日程页添加课程或导入课表。"
    />

    <view class="section-head row between">
      <text class="section-title">待处理提醒</text>
      <button class="ghost-btn small" data-testid="home-manage-schedule-button" @tap="goSchedule">管理日程</button>
    </view>
    <view v-if="visibleReminders.length" class="task-list">
      <view v-for="item in visibleReminders" :key="item.id" class="task-card">
        <view class="task-copy" @tap="goSchedule">
          <text class="tag">{{ reminderTime(item.remindAt) }} · {{ reminderStatusText(item.status) }}</text>
          <text class="task-title">{{ item.title }}</text>
          <text class="caption">{{ item.note || item.category || '暂无备注' }}</text>
        </view>
        <button v-if="item.status !== 'done'" class="ghost-btn mini success" :disabled="completingReminderId === item.id" @tap="completeReminder(item)">
          {{ completingReminderId === item.id ? '处理中' : '完成' }}
        </button>
      </view>
    </view>
    <AppState
      v-else-if="!loading && !error"
      type="empty"
      title="暂时没有待处理提醒"
      message="今天可以少背一点清单，保持节奏就好。"
    />

    <view class="section-head row between">
      <text class="section-title">今日跟进</text>
      <text class="caption">{{ summary.followUpsCount || 0 }} 位家长</text>
    </view>
    <view v-if="visibleFocusParents.length" class="task-list">
      <view v-for="parent in visibleFocusParents" :key="parent.id" class="task-card parent-task" @tap="openParent(parent)">
        <view class="task-copy">
          <text class="tag">{{ riskText(parent.riskLevel) }}</text>
          <text class="task-title">{{ parent.studentName }} · {{ parent.parentName }}</text>
          <text class="caption">{{ parent.nextAction || parent.importantNotes || '打开沟通助手补一条跟进记录' }}</text>
        </view>
        <button class="ghost-btn mini" @tap.stop="openParent(parent)">记录</button>
      </view>
    </view>
    <AppState
      v-else-if="!loading && !error"
      type="empty"
      title="没有到期跟进"
      message="新的沟通记录设置跟进时间后，会在这里提醒。"
    />
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { ensureLoggedIn, errorMessage, showToast } from '../../utils/ui'
import AppState from '../../components/AppState.vue'

const summary = ref({ rhythm: {} })
const profile = ref({})
const loading = ref(false)
const error = ref('')
const mood = ref('steady')
const completingReminderId = ref('')
const moods = [
  { code: 'steady', icon: '平', label: '平稳' },
  { code: 'busy', icon: '忙', label: '偏满' },
  { code: 'soft', icon: '缓', label: '缓冲' },
  { code: 'focus', icon: '专', label: '专注' },
  { code: 'warm', icon: '沟', label: '沟通' }
]

const todayLabel = new Date().toLocaleDateString('zh-CN', { weekday: 'long', month: 'long', day: 'numeric' })
const visibleReminders = computed(() => (summary.value.reminders || []).slice(0, 3))
const visibleFocusParents = computed(() => (summary.value.focusParents || []).slice(0, 3))

onShow(load)

async function load() {
  if (!ensureLoggedIn(api)) return
  loading.value = true
  error.value = ''
  try {
    const [nextSummary, nextProfile] = await Promise.all([api.dashboard(), api.me()])
    summary.value = nextSummary
    profile.value = nextProfile
  } catch (err) {
    error.value = errorMessage(err, '首页加载失败')
  } finally {
    loading.value = false
  }
}
function goSchedule() { uni.switchTab({ url: '/pages/schedule/index' }) }
function goHeal() { uni.switchTab({ url: '/pages/heal/index' }) }
function goCommunication() { uni.switchTab({ url: '/pages/communication/index' }) }
function openParent(parent) {
  if (!parent?.id) {
    goCommunication()
    return
  }
  uni.navigateTo({ url: `/pages/communication/parent-detail?id=${encodeURIComponent(parent.id)}` })
}

async function completeReminder(item) {
  if (!item?.id || completingReminderId.value) return
  completingReminderId.value = item.id
  try {
    await api.completeReminder(item.id)
    showToast('已完成提醒')
    await load()
  } catch (err) {
    showToast(errorMessage(err, '提醒更新失败'))
  } finally {
    completingReminderId.value = ''
  }
}

function reminderTime(value) {
  if (!value) return '今天'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '今天'
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function reminderStatusText(status) {
  return ({ pending: '待处理', done: '已完成', snoozed: '已延后' })[status] || '待处理'
}

function riskText(value) {
  return ({ low: '低风险', medium: '中风险', high: '高风险' })[value] || '低风险'
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header { padding: 0 32rpx 12rpx; }
.bell { width: 88rpx; height: 88rpx; border-radius: 28rpx; background: rgba(255,255,255,.86); color: #0e7490; }
.hero { background: linear-gradient(134deg,rgba(236,254,255,.98),rgba(248,251,255,.94) 44%,rgba(241,248,238,.96)); }
.hero-title { margin: 20rpx 0 12rpx; font-size: 56rpx; font-weight: 950; color: #172039; }
.actions { margin-top: 28rpx; }
.action { flex: 1; }
.section-head { padding: 16rpx 32rpx 0; }
.mood-row { display: flex; gap: 16rpx; padding: 20rpx 32rpx; overflow-x: auto; }
.mood-chip { min-width: 112rpx; height: 112rpx; border-radius: 28rpx; background: rgba(255,255,255,.78); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8rpx; font-size: 22rpx; font-weight: 800; color: #31505b; }
.mood-chip.active { outline: 4rpx solid rgba(8,145,178,.28); }
.mood-icon { font-size: 32rpx; }
.bento { display: grid; grid-template-columns: 1fr 1fr; gap: 24rpx; padding: 0 32rpx; }
.tile { min-height: 190rpx; border-radius: 32rpx; padding: 28rpx; display: flex; flex-direction: column; justify-content: space-between; box-shadow: 0 20rpx 44rpx rgba(20,78,99,.10); }
.tile.blue { background: linear-gradient(145deg,rgba(207,250,254,.94),rgba(232,247,255,.82)); }
.tile.mint { background: linear-gradient(145deg,rgba(209,250,229,.90),rgba(236,254,255,.82)); }
.tile.pink { background: linear-gradient(145deg,rgba(255,242,229,.90),rgba(236,254,255,.78)); }
.tile.peach { background: linear-gradient(145deg,rgba(241,248,238,.92),rgba(255,248,232,.84)); }
.num { font-size: 56rpx; font-weight: 950; color: #182033; }
.tile-title, .course-title { font-size: 32rpx; font-weight: 900; color: #182033; }
.task-list { margin: 16rpx 32rpx 8rpx; display: flex; flex-direction: column; gap: 18rpx; }
.task-card { padding: 24rpx; border-radius: 28rpx; background: rgba(255,255,255,.86); box-shadow: 0 18rpx 42rpx rgba(20,78,99,.08); display: flex; align-items: center; gap: 18rpx; }
.parent-task { align-items: stretch; }
.task-copy { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 10rpx; }
.task-title { font-size: 30rpx; line-height: 1.35; font-weight: 900; color: #14313b; }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.success { color: #059669; }
</style>
