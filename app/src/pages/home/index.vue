<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">{{ summary.todayLabel || todayLabel }}</text>
        <view class="title">早上好，{{ profile.name || '微光老师' }}</view>
      </view>
      <button class="avatar-btn notify" data-testid="home-schedule-button" @tap="goSchedule">
        <text class="icon-chip"><AppIcon name="bell" /></text>
      </button>
    </view>

    <view class="hero" :class="`hero-${mood}`">
      <view class="hero-copy">
        <text class="hero-tag">{{ heroRhythm.tag }}</text>
        <view class="hero-title">{{ heroRhythm.title }}</view>
        <text class="hero-body">{{ heroRhythm.description }}</text>
        <view class="row actions">
          <button class="primary-btn action" @tap="goCommunication"><text class="btn-icon"><AppIcon name="bot" /></text>助手</button>
          <button class="ghost-btn action" data-testid="home-today-schedule-button" @tap="goSchedule"><text class="btn-icon"><AppIcon name="calendar" /></text>安排</button>
        </view>
      </view>
      <view class="teacher-art" aria-hidden="true">
        <view class="hair"></view>
        <view class="face"></view>
        <view class="body-shape"></view>
        <view class="book"></view>
        <view class="note-a"></view>
        <view class="note-b"></view>
      </view>
    </view>

    <AppState v-if="loading" type="loading" message="正在整理今天的安排..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="section-head row between">
      <text class="section-title">今日状态</text>
      <text class="caption">低打扰工作台</text>
    </view>
    <view class="mood-row">
      <view v-for="item in moods" :key="item.code" class="mood-chip" :class="{ active: item.code === mood }" :data-testid="`home-mood-${item.code}`" @tap="mood = item.code">
        <text class="mood-icon"><AppIcon :name="item.icon" /></text>
        <text>{{ item.label }}</text>
      </view>
    </view>

    <view class="bento">
      <view class="tile blue" data-testid="home-course-tile" @tap="goSchedule">
        <view class="row between"><text class="tag">课程</text><text class="icon-chip"><AppIcon name="arrowUpRight" /></text></view>
        <view><text class="num">{{ summary.coursesCount || 0 }}</text><text class="caption block">今日课程</text></view>
      </view>
      <view class="tile mint" data-testid="home-reminder-tile" @tap="goSchedule">
        <view class="row between"><text class="tag">待办</text><text class="icon-chip"><AppIcon name="listChecks" /></text></view>
        <view><text class="num">{{ summary.remindersCount || 0 }}</text><text class="caption block">待处理提醒</text></view>
      </view>
      <view class="tile pink" @tap="goCommunication">
        <view class="row between"><text class="tag">AI</text><text class="icon-chip"><AppIcon name="bot" /></text></view>
        <view><text class="tile-title">生成家校回复</text><text class="caption block">多语气版本</text></view>
      </view>
      <view class="tile peach" @tap="goHeal">
        <view class="row between"><text class="tag">恢复</text><text class="icon-chip"><AppIcon name="heartHand" /></text></view>
        <view><text class="tile-title">一分钟呼吸</text><text class="caption block">先收回注意力</text></view>
      </view>
    </view>

    <view class="next-course-card" v-if="summary.nextCourse" data-testid="home-next-course-card" @tap="goSchedule">
      <view class="next-glow"></view>
      <view class="next-course-main">
        <view class="next-course-mark">
          <AppIcon name="calendarCheck" />
        </view>
        <view class="next-course-copy">
          <view class="next-kicker-row">
            <text class="next-kicker">下一项</text>
            <text class="next-range">{{ nextCourseTimeRange }}</text>
          </view>
          <text class="next-course-title">{{ nextCourseTitle }}</text>
        </view>
        <view class="next-time-pill">
          <text class="next-time">{{ nextCourseStartTime }}</text>
        </view>
      </view>
      <view v-if="nextCourseMeta.length" class="next-meta-row">
        <view v-for="item in nextCourseMeta" :key="item.label" class="next-meta-chip">
          <AppIcon :name="item.icon" />
          <text>{{ item.label }}</text>
        </view>
      </view>
      <view class="next-course-foot">
        <text class="caption">{{ nextCourseFootnote }}</text>
        <view class="next-link">
          <text>查看安排</text>
          <AppIcon name="chevronRight" />
        </view>
      </view>
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
      <view v-for="item in visibleReminders" :key="item.id" class="task-card reminder-card">
        <view class="reminder-head">
          <view class="time-meta" :class="`status-${item.status || 'pending'}`" @tap="goSchedule">
            <text class="time-text">{{ reminderTime(item.remindAt) }}</text>
            <text class="status-text">{{ reminderStatusText(item.status) }}</text>
          </view>
          <button v-if="item.status !== 'done'" class="ghost-btn mini success" :disabled="completingReminderId === item.id" @tap="completeReminder(item)">
            {{ completingReminderId === item.id ? '处理中' : '完成' }}
          </button>
        </view>
        <view class="task-copy" @tap="goSchedule">
          <text class="task-title">{{ item.title }}</text>
          <text class="caption">{{ item.note || item.category || '暂无备注' }}</text>
        </view>
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
import AppIcon from '../../components/AppIcon.vue'

const summary = ref({ rhythm: {} })
const profile = ref({})
const loading = ref(false)
const error = ref('')
const mood = ref('steady')
const completingReminderId = ref('')
const moodRhythms = {
  steady: {
    tag: '今日教学心流',
    title: '温柔但高效',
    description: '先把节奏拆小，保持一点余裕。'
  },
  busy: {
    tag: '偏满模式',
    title: '先减压，再推进',
    description: '把今天拆成三件必要事，能延后的先放进提醒里。'
  },
  soft: {
    tag: '缓冲模式',
    title: '留一点空白',
    description: '先安排短恢复，再处理沟通和待办，别把自己排满。'
  },
  focus: {
    tag: '专注模式',
    title: '单线程推进',
    description: '关掉额外切换，优先完成一项课程或材料整理。'
  },
  warm: {
    tag: '沟通模式',
    title: '先共情，再说明',
    description: '适合处理家长反馈，用清晰温和的话减少反复解释。'
  }
}
const moods = [
  { code: 'steady', icon: 'smile', label: '平稳' },
  { code: 'busy', icon: 'activity', label: '偏满' },
  { code: 'soft', icon: 'moon', label: '缓冲' },
  { code: 'focus', icon: 'target', label: '专注' },
  { code: 'warm', icon: 'message', label: '沟通' }
]

const todayLabel = new Date().toLocaleDateString('zh-CN', { weekday: 'long', month: 'long', day: 'numeric' })
const heroRhythm = computed(() => {
  const fallback = moodRhythms[mood.value] || moodRhythms.steady
  if (mood.value !== 'steady') return fallback
  return {
    tag: fallback.tag,
    title: summary.value.rhythm?.title || fallback.title,
    description: summary.value.rhythm?.description || fallback.description
  }
})
const visibleReminders = computed(() => (summary.value.reminders || []).slice(0, 3))
const visibleFocusParents = computed(() => (summary.value.focusParents || []).slice(0, 3))
const nextCourse = computed(() => summary.value.nextCourse || null)
const nextCourseTitle = computed(() => compactJoin([nextCourse.value?.className, nextCourse.value?.title], ' · ') || '待安排课程')
const nextCourseStartTime = computed(() => nextCourse.value?.startTime || '待定')
const nextCourseTimeRange = computed(() => {
  const start = nextCourse.value?.startTime
  const end = nextCourse.value?.endTime
  if (start && end) return `${start} - ${end}`
  return start || '时间待定'
})
const nextCourseMeta = computed(() => {
  const items = []
  if (nextCourse.value?.location) items.push({ icon: 'mapPin', label: nextCourse.value.location })
  if (nextCourse.value?.note) items.push({ icon: 'bookmark', label: nextCourse.value.note })
  return items
})
const nextCourseFootnote = computed(() => {
  const start = clockMinutes(nextCourse.value?.startTime)
  const end = clockMinutes(nextCourse.value?.endTime)
  if (start == null) return '打开日程页可以补充地点和备注。'
  const now = currentMinutes()
  if (end != null && now >= start && now < end) return '这节课正在进行中。'
  if (now < start) {
    const diff = start - now
    if (diff < 60) return `${diff} 分钟后开始，先留一点准备时间。`
    const hours = Math.floor(diff / 60)
    const minutes = diff % 60
    return minutes > 0 ? `${hours} 小时 ${minutes} 分钟后开始。` : `${hours} 小时后开始。`
  }
  return '下一项课程信息已同步到日程。'
})

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

function compactJoin(values, separator = ' · ') {
  return values.map((value) => String(value || '').trim()).filter(Boolean).join(separator)
}

function clockMinutes(value) {
  const match = String(value || '').match(/^(\d{2}):(\d{2})$/)
  if (!match) return null
  return Number(match[1]) * 60 + Number(match[2])
}

function currentMinutes() {
  const now = new Date()
  return now.getHours() * 60 + now.getMinutes()
}

function riskText(value) {
  return ({ low: '低风险', medium: '中风险', high: '高风险' })[value] || '低风险'
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.avatar-btn { width: 88rpx; height: 88rpx; padding: 0; border-radius: 28rpx; background: linear-gradient(180deg, rgba(255,255,255,.96), rgba(250,253,255,.86)); border: 1px solid rgba(97,116,166,.14); box-shadow: 0 10rpx 22rpx rgba(73,91,146,.10), inset 0 1px 0 rgba(255,255,255,.96); position: relative; display: flex; align-items: center; justify-content: center; }
.avatar-btn.notify::after { content: ""; position: absolute; top: 16rpx; right: 16rpx; width: 14rpx; height: 14rpx; border-radius: 999rpx; background: #ee6e87; box-shadow: 0 0 0 5rpx rgba(255,255,255,.88); }
.hero { min-height: 430rpx; margin: 6rpx 0 24rpx; padding: 32rpx; border-radius: 38rpx; color: #23213b; display: grid; grid-template-columns: minmax(0,1fr) 210rpx; gap: 24rpx; overflow: hidden; background: radial-gradient(80% 72% at 14% 12%, rgba(255,255,255,.58), transparent 52%), linear-gradient(138deg, rgba(176,210,255,.92) 0%, rgba(198,175,255,.88) 36%, rgba(255,174,211,.82) 68%, rgba(255,225,151,.86) 100%); border: 1rpx solid rgba(255,255,255,.72); box-shadow: 0 28rpx 58rpx rgba(122,89,174,.20), inset 0 1rpx 0 rgba(255,255,255,.70); box-sizing: border-box; }
.hero-busy { background: radial-gradient(80% 72% at 14% 12%, rgba(255,255,255,.62), transparent 52%), linear-gradient(138deg, rgba(255,214,169,.92) 0%, rgba(255,176,199,.86) 46%, rgba(214,198,255,.84) 100%); }
.hero-soft { background: radial-gradient(80% 72% at 14% 12%, rgba(255,255,255,.64), transparent 52%), linear-gradient(138deg, rgba(220,236,255,.94) 0%, rgba(232,224,255,.88) 48%, rgba(220,246,237,.86) 100%); }
.hero-focus { background: radial-gradient(80% 72% at 14% 12%, rgba(255,255,255,.58), transparent 52%), linear-gradient(138deg, rgba(180,216,255,.94) 0%, rgba(153,200,229,.88) 48%, rgba(196,233,218,.86) 100%); }
.hero-warm { background: radial-gradient(80% 72% at 14% 12%, rgba(255,255,255,.58), transparent 52%), linear-gradient(138deg, rgba(255,205,224,.88) 0%, rgba(255,231,180,.86) 52%, rgba(214,232,255,.86) 100%); }
.hero-copy { min-width: 0; display: flex; flex-direction: column; justify-content: space-between; gap: 18rpx; }
.hero-tag { width: fit-content; padding: 12rpx 18rpx; border-radius: 999rpx; border: 1rpx solid rgba(255,255,255,.76); color: #5d407b; background: rgba(255,255,255,.52); font-size: 22rpx; font-weight: 900; }
.hero-title { margin-top: 4rpx; font-size: 58rpx; line-height: 1.03; font-weight: 950; color: #172039; letter-spacing: 0; }
.hero-body { max-width: 360rpx; color: rgba(38,37,62,.72); font-size: 25rpx; line-height: 1.5; }
.actions { margin-top: 8rpx; align-items: stretch; }
.action { flex: 1; min-width: 0; min-height: 76rpx; border-radius: 999rpx; }
.teacher-art { min-height: 320rpx; border-radius: 30rpx; position: relative; overflow: hidden; align-self: stretch; background: radial-gradient(80% 60% at 50% 18%, rgba(255,255,255,.62), transparent 60%), linear-gradient(160deg, rgba(255,255,255,.26), rgba(255,255,255,.08)); border: 1rpx solid rgba(255,255,255,.56); }
.teacher-art .face { position: absolute; left: 50%; top: 52rpx; transform: translateX(-50%); width: 94rpx; height: 94rpx; border-radius: 999rpx; background: linear-gradient(145deg, #ffd9bd, #ffb6c4); box-shadow: 0 10rpx 24rpx rgba(116,72,156,.16); }
.teacher-art .hair { position: absolute; width: 110rpx; height: 54rpx; left: 50%; top: 38rpx; transform: translateX(-50%); border-radius: 99rpx 99rpx 28rpx 28rpx; background: #5d5888; z-index: 1; }
.teacher-art .body-shape { position: absolute; left: 50%; top: 154rpx; transform: translateX(-50%); width: 132rpx; height: 150rpx; border-radius: 48rpx 48rpx 30rpx 30rpx; background: linear-gradient(145deg, #758ce7, #64bfd1); }
.teacher-art .book { position: absolute; right: 18rpx; bottom: 34rpx; width: 82rpx; height: 58rpx; border-radius: 18rpx; background: linear-gradient(145deg, #fff, #ffe9a7); box-shadow: 0 10rpx 20rpx rgba(84,74,151,.14); }
.teacher-art .note-a, .teacher-art .note-b { position: absolute; width: 46rpx; height: 46rpx; border-radius: 16rpx; background: rgba(255,255,255,.72); border: 1rpx solid rgba(255,255,255,.8); }
.teacher-art .note-a { left: 18rpx; top: 72rpx; transform: rotate(-8deg); }
.teacher-art .note-b { left: 28rpx; bottom: 42rpx; transform: rotate(7deg); }
.section-head { padding: 26rpx 4rpx 8rpx; }
.mood-row { display: grid; grid-template-columns: repeat(5, minmax(0,1fr)); gap: 12rpx; padding: 14rpx 0 20rpx; overflow-x: auto; }
.mood-chip { min-width: 0; height: 110rpx; border-radius: 24rpx; background: rgba(255,255,255,.61); border: 1rpx solid rgba(255,255,255,.78); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6rpx; font-size: 21rpx; font-weight: 850; color: #3f405b; box-shadow: inset 0 1rpx 0 rgba(255,255,255,.86), 0 9rpx 18rpx rgba(73,91,146,.07); }
.mood-chip.active { outline: 4rpx solid rgba(98,166,233,.30); background: rgba(255,255,255,.84); }
.mood-icon { width: 34rpx; height: 34rpx; display: flex; align-items: center; justify-content: center; font-size: 31rpx; line-height: 1; color: #50619b; }
.bento { display: grid; grid-template-columns: 1fr 1fr; gap: 20rpx; padding: 0; }
.tile { min-height: 218rpx; border-radius: 30rpx; padding: 26rpx; display: flex; flex-direction: column; justify-content: space-between; border: 1rpx solid rgba(255,255,255,.72); box-shadow: 0 14rpx 30rpx rgba(73,91,146,.10), inset 0 1rpx 0 rgba(255,255,255,.74); box-sizing: border-box; overflow: hidden; }
.tile.blue { background: linear-gradient(145deg, rgba(206,231,255,.90), rgba(220,211,255,.72)); }
.tile.mint { background: linear-gradient(145deg, rgba(202,240,229,.86), rgba(216,237,255,.74)); }
.tile.pink { background: linear-gradient(145deg, rgba(255,210,229,.82), rgba(255,232,185,.72)); }
.tile.peach { background: linear-gradient(145deg, rgba(255,239,184,.82), rgba(255,214,202,.72)); }
.num { display: block; font-size: 58rpx; line-height: 1; font-weight: 950; color: #182033; }
.tile-title, .course-title { display: block; font-size: 32rpx; line-height: 1.26; font-weight: 930; color: #182033; }
.next-course-card { position: relative; margin: 24rpx 0 6rpx; padding: 28rpx; border-radius: 32rpx; color: #172039; background: linear-gradient(145deg, rgba(255,255,255,.94), rgba(243,248,255,.78)); border: 1px solid rgba(97,116,166,.11); box-shadow: 0 16rpx 34rpx rgba(73,91,146,.09), inset 0 1px 0 rgba(255,255,255,.96); overflow: hidden; }
.next-glow { position: absolute; right: -44rpx; top: -52rpx; width: 180rpx; height: 180rpx; border-radius: 999rpx; background: radial-gradient(circle, rgba(98,103,233,.15), rgba(82,184,207,.06) 58%, transparent 72%); pointer-events: none; }
.next-course-main { position: relative; display: grid; grid-template-columns: 72rpx minmax(0,1fr) auto; gap: 18rpx; align-items: center; }
.next-course-mark { width: 72rpx; height: 72rpx; border-radius: 24rpx; color: #fff; background: linear-gradient(145deg, #6f86df, #52b8cf); display: flex; align-items: center; justify-content: center; font-size: 35rpx; box-shadow: 0 12rpx 24rpx rgba(82,111,190,.18), inset 0 1px 0 rgba(255,255,255,.26); }
.next-course-copy { min-width: 0; display: flex; flex-direction: column; gap: 9rpx; }
.next-kicker-row { min-width: 0; display: flex; align-items: center; gap: 12rpx; flex-wrap: wrap; }
.next-kicker { color: #6267e9; font-size: 23rpx; line-height: 1.2; font-weight: 950; }
.next-range { color: #8790a8; font-size: 22rpx; line-height: 1.2; font-weight: 850; font-variant-numeric: tabular-nums; }
.next-course-title { color: #172039; font-size: 34rpx; line-height: 1.24; font-weight: 950; overflow-wrap: anywhere; }
.next-time-pill { min-width: 112rpx; min-height: 64rpx; padding: 0 20rpx; border-radius: 999rpx; color: #43517b; background: rgba(255,255,255,.76); border: 1rpx solid rgba(97,116,166,.10); display: flex; align-items: center; justify-content: center; box-shadow: 0 8rpx 18rpx rgba(73,91,146,.06), inset 0 1px 0 rgba(255,255,255,.92); box-sizing: border-box; }
.next-time { font-size: 29rpx; line-height: 1; font-weight: 950; font-variant-numeric: tabular-nums; }
.next-meta-row { position: relative; display: flex; flex-wrap: wrap; gap: 12rpx; margin-top: 22rpx; }
.next-meta-chip { max-width: 100%; padding: 12rpx 16rpx; border-radius: 18rpx; color: #5e6983; background: rgba(245,249,255,.86); border: 1rpx solid rgba(97,116,166,.08); display: flex; align-items: center; gap: 8rpx; font-size: 23rpx; line-height: 1.3; font-weight: 800; box-sizing: border-box; }
.next-meta-chip .app-icon { flex: 0 0 auto; width: 25rpx; height: 25rpx; color: #6f86df; }
.next-meta-chip text { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.next-course-foot { position: relative; display: flex; align-items: center; justify-content: space-between; gap: 16rpx; margin-top: 22rpx; padding-top: 20rpx; border-top: 1rpx solid rgba(97,116,166,.08); }
.next-course-foot .caption { flex: 1; min-width: 0; margin: 0; }
.next-link { flex: 0 0 auto; color: #5263a2; display: flex; align-items: center; gap: 6rpx; font-size: 22rpx; line-height: 1.2; font-weight: 950; }
.next-link .app-icon { width: 24rpx; height: 24rpx; }
.task-list { margin: 14rpx 0 8rpx; display: flex; flex-direction: column; gap: 18rpx; }
.task-card { padding: 26rpx; border-radius: 28rpx; display: flex; align-items: center; gap: 18rpx; }
.reminder-card { align-items: stretch; flex-direction: column; gap: 18rpx; }
.reminder-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16rpx; }
.time-meta { flex: 0 0 auto; min-width: 136rpx; padding: 16rpx 18rpx; border-radius: 24rpx; display: flex; flex-direction: column; gap: 5rpx; color: #46577f; background: linear-gradient(145deg, rgba(255,255,255,.96), rgba(245,249,255,.78)); border: 1px solid rgba(97,116,166,.12); box-shadow: 0 10rpx 22rpx rgba(73,91,146,.08), inset 0 1px 0 rgba(255,255,255,.96); box-sizing: border-box; }
.time-text { font-size: 34rpx; line-height: 1; font-weight: 950; letter-spacing: 0; font-variant-numeric: tabular-nums; }
.status-text { display: flex; align-items: center; gap: 8rpx; color: #6d7894; font-size: 22rpx; line-height: 1.2; font-weight: 900; white-space: nowrap; }
.status-text::before { content: ""; width: 10rpx; height: 10rpx; border-radius: 999rpx; background: #6f86df; box-shadow: 0 0 0 5rpx rgba(111,134,223,.12); }
.time-meta.status-done { color: #2f7f6e; background: linear-gradient(145deg, rgba(242,255,250,.96), rgba(229,248,240,.76)); }
.time-meta.status-done .status-text::before { background: #3a8a76; box-shadow: 0 0 0 5rpx rgba(58,138,118,.13); }
.time-meta.status-snoozed { color: #7c679f; background: linear-gradient(145deg, rgba(250,246,255,.96), rgba(243,236,255,.76)); }
.time-meta.status-snoozed .status-text::before { background: #a990ea; box-shadow: 0 0 0 5rpx rgba(169,144,234,.13); }
.parent-task { align-items: stretch; }
.task-copy { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 10rpx; }
.task-title { font-size: 30rpx; line-height: 1.35; font-weight: 900; color: #14313b; }
.small { min-height: 62rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 54rpx; padding: 0 18rpx; font-size: 22rpx; }
@media (max-width: 360px) {
  .hero { grid-template-columns: 1fr; }
  .teacher-art { min-height: 220rpx; }
}
</style>
