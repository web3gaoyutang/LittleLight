<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">账号 · 素材库</text>
        <view class="title">我的微光</view>
      </view>
      <button class="ghost-btn small" @tap="openProfileForm"><text class="btn-icon"><AppIcon name="settings" /></text>编辑资料</button>
    </view>

    <view class="card profile">
      <view class="avatar">{{ avatarText }}</view>
      <view class="profile-main">
        <text class="section-title">{{ profile.name || '未设置姓名' }}</text>
        <text class="body block">{{ profile.school || '未设置学校' }} · {{ profile.subject || '学科' }} · {{ roleText }}</text>
        <view class="row tag-row">
          <text class="tag">{{ reminderText }}</text>
          <text class="tag">{{ proText }}</text>
        </view>
      </view>
    </view>

    <view class="card login-card">
      <view class="row between">
        <view>
          <text class="section-title">微信登录状态</text>
          <text class="body block">{{ loginText }}</text>
        </view>
        <view class="row action-row">
          <button v-if="showDevLogin" class="primary-btn small" @tap="mockWechatLogin"><text class="btn-icon"><AppIcon name="wechat" /></text>{{ loginButtonText }}</button>
          <button class="ghost-btn small danger" :disabled="loggingOut" @tap="logout"><text class="btn-icon"><AppIcon name="logout" /></text>{{ loggingOut ? '退出中...' : '退出' }}</button>
        </view>
      </view>
    </view>

    <AppState v-if="loading" type="loading" message="正在加载账号数据..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="material">
      <view class="row between">
        <text class="tag">素材</text>
        <text class="icon-chip"><AppIcon name="bookmark" /></text>
      </view>
      <view class="row between">
        <view class="material-copy">
          <text class="section-title">常用素材库</text>
          <text class="body">常用沟通模板、AI夸夸用语、班级反馈句式。</text>
        </view>
        <button class="primary-btn small" @tap="openFavoriteForm"><text class="btn-icon"><AppIcon name="plus" /></text>添加收藏</button>
      </view>
    </view>

    <view class="section-head row between">
      <text class="section-title">我的收藏</text>
      <text class="caption">{{ favorites.length }} 条</text>
    </view>
    <view class="search-bar">
      <text class="search-icon"><AppIcon name="search" /></text>
      <input class="input search-input" v-model="favoriteQuery" confirm-type="search" placeholder="搜索标题、内容或类型" aria-label="搜索收藏素材" data-testid="favorite-search-input" @confirm="searchFavorites" />
      <button class="ghost-btn small" :disabled="loadingFavorites" @tap="searchFavorites">搜索</button>
    </view>
    <view v-for="item in favorites" :key="item.id" class="card">
      <view class="row between">
        <text class="tag">{{ favoriteTypeText(item.type) }}</text>
        <button class="ghost-btn mini danger" @tap="removeFavorite(item.id)"><text class="btn-icon"><AppIcon name="trash" /></text>删除</button>
      </view>
      <text class="section-title">{{ item.title }}</text>
      <text class="body">{{ item.content }}</text>
    </view>
    <AppState
      v-if="!loading && !error && favorites.length === 0"
      type="empty"
      title="还没有收藏"
      message="可以把常用沟通模板和 AI 夸夸保存到这里。"
    />
    <button v-if="favoritesHasMore" class="ghost-btn load-more" :disabled="loadingFavorites" @tap="loadMoreFavorites">
      {{ loadingFavorites ? '加载中...' : '加载更多收藏' }}
    </button>

    <view v-if="favoriteFormOpen" class="modal-mask" @tap="closeFavoriteForm">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">添加收藏素材</text>
          <button class="ghost-btn mini" @tap="closeFavoriteForm">关闭</button>
        </view>
        <view class="form-grid">
          <picker class="input" :range="favoriteTypeLabels" @change="setFavoriteType">
            <text>{{ favoriteTypeText(favoriteForm.type) }}</text>
          </picker>
          <input class="input" :class="{ invalid: favoriteFormError && !hasText(favoriteForm.title) }" v-model="favoriteForm.title" maxlength="80" placeholder="素材标题" aria-label="素材标题" data-testid="favorite-title-input" />
          <textarea class="textarea short" :class="{ invalid: favoriteFormError && !hasText(favoriteForm.content) }" v-model="favoriteForm.content" maxlength="2000" placeholder="输入常用话术、夸夸句或班级反馈内容" aria-label="素材内容" data-testid="favorite-content-input" />
          <text v-if="favoriteFormError" class="form-error">{{ favoriteFormError }}</text>
          <button class="primary-btn" :disabled="saving" @tap="submitFavorite">{{ saving ? '保存中...' : '保存收藏' }}</button>
        </view>
      </view>
    </view>

    <view class="section-head row between">
      <text class="section-title">AI 生成记录</text>
      <text class="caption">{{ aiLogs.length }} 条</text>
    </view>
    <view class="search-bar">
      <text class="search-icon"><AppIcon name="search" /></text>
      <input class="input search-input" v-model="aiLogQuery" confirm-type="search" placeholder="搜索输入、输出或安全标记" aria-label="搜索 AI 生成记录" data-testid="ai-log-search-input" @confirm="searchAiLogs" />
      <button class="ghost-btn small" :disabled="loadingAiLogs" @tap="searchAiLogs">搜索</button>
    </view>
    <view v-for="log in aiLogs" :key="log.id" class="card ai-log-card" @tap="openAiLog(log)">
      <view class="row between">
        <text class="tag">{{ scenarioText(log.scenario) }}</text>
        <text class="caption">{{ formatTime(log.createdAt) }}</text>
      </view>
      <text class="body">{{ outputPreview(log) }}</text>
      <view class="row between ai-log-foot">
        <text class="caption block">安全标记：{{ safetyText(log.safetyLabel) }}</text>
        <view class="row action-row">
          <button class="ghost-btn mini" @tap.stop="openAiLog(log)"><text class="btn-icon"><AppIcon name="file" /></text>详情</button>
          <button class="ghost-btn mini danger" @tap.stop="removeAiLog(log)"><text class="btn-icon"><AppIcon name="trash" /></text>删除</button>
        </view>
      </view>
    </view>
    <AppState
      v-if="!loading && !error && aiLogs.length === 0"
      type="empty"
      title="暂无 AI 生成记录"
      message="生成家校草稿或 AI 夸夸后，会在这里留下复盘记录。"
    />
    <button v-if="aiLogsHasMore" class="ghost-btn load-more" :disabled="loadingAiLogs" @tap="loadMoreAiLogs">
      {{ loadingAiLogs ? '加载中...' : '加载更多 AI 记录' }}
    </button>

    <view class="section-head row between">
      <text class="section-title">数据与权益</text>
      <button class="ghost-btn small" @tap="showBenefits"><text class="btn-icon"><AppIcon name="crown" /></text>查看权益</button>
    </view>
    <view class="card boundary-card" @tap="showBenefits">
      <view class="row between">
        <text class="section-title">微光 Pro</text>
        <text class="tag" :class="{ mutedTag: !checkoutReady }">{{ entitlementText }}</text>
      </view>
      <text class="body">{{ entitlementMessage }}</text>
      <view class="boundary-meta">
        <text class="caption">支付：{{ checkoutReady ? '已接入' : '未接入' }}</text>
        <button class="ghost-btn mini" @tap.stop="showBenefits"><text class="btn-icon"><AppIcon name="info" /></text>查看状态</button>
      </view>
    </view>
    <view class="card boundary-card" @tap="showNotificationStatus">
      <view class="row between">
        <text class="section-title">提醒与通知</text>
        <text class="tag" :class="{ mutedTag: !notificationReady }">{{ notificationText }}</text>
      </view>
      <text class="body">{{ notificationMessage }}</text>
      <view class="boundary-meta">
        <text class="caption">推送：{{ notificationReady ? '已接入' : '未接入' }}</text>
        <button class="ghost-btn mini" @tap.stop="toggleReminderPolicy"><text class="btn-icon"><AppIcon name="bell" /></text>切换策略</button>
      </view>
    </view>
    <view class="card boundary-card" @tap="showSyncStatus">
      <view class="row between">
        <text class="section-title">云同步状态</text>
        <text class="tag" :class="{ mutedTag: !objectStorageReady }">{{ syncText }}</text>
      </view>
      <text class="body">{{ syncMessage }}</text>
      <view class="boundary-meta">
        <text class="caption">对象存储：{{ objectStorageReady ? '已接入' : '未接入' }}</text>
        <button class="ghost-btn mini" @tap.stop="showSyncStatus"><text class="btn-icon"><AppIcon name="cloud" /></text>刷新状态</button>
      </view>
    </view>
    <view class="card boundary-card danger-zone">
      <view class="row between">
        <text class="section-title">账号与数据</text>
        <text class="tag danger">敏感操作</text>
      </view>
      <text class="body">删除当前账号会移除课程、提醒、家长档案、沟通记录、疗愈记录、AI 生成记录、收藏和服务端 session。</text>
      <view class="boundary-meta">
        <text class="caption">删除后重新微信登录会创建新账号</text>
        <view class="row action-row">
          <button class="ghost-btn mini" :disabled="exportingAccount" @tap.stop="exportAccountData">
            <text class="btn-icon"><AppIcon name="export" /></text>
            {{ exportingAccount ? '导出中...' : '导出数据' }}
          </button>
          <button class="ghost-btn mini danger" :disabled="deletingAccount" @tap.stop="deleteAccount">
            <text class="btn-icon"><AppIcon name="trash" /></text>
            {{ deletingAccount ? '删除中...' : '删除账号' }}
          </button>
        </view>
      </view>
    </view>

    <view v-if="profileFormOpen" class="modal-mask" @tap="closeProfileForm">
      <view class="modal-card" @tap.stop>
        <view class="row between">
          <text class="section-title">编辑教师资料</text>
          <button class="ghost-btn mini" @tap="closeProfileForm">关闭</button>
        </view>
        <view class="form-grid">
          <input class="input" :class="{ invalid: profileFormError && !hasText(profileForm.name) }" v-model="profileForm.name" maxlength="40" placeholder="姓名" aria-label="教师姓名" data-testid="profile-name-input" />
          <input class="input" :class="{ invalid: profileFormError && !withinLength(profileForm.school, 80) }" v-model="profileForm.school" maxlength="80" placeholder="学校" aria-label="学校" data-testid="profile-school-input" />
          <input class="input" :class="{ invalid: profileFormError && !withinLength(profileForm.stage, 40) }" v-model="profileForm.stage" maxlength="40" placeholder="学段" aria-label="学段" data-testid="profile-stage-input" />
          <input class="input" :class="{ invalid: profileFormError && !withinLength(profileForm.subject, 40) }" v-model="profileForm.subject" maxlength="40" placeholder="学科" aria-label="学科" data-testid="profile-subject-input" />
          <button class="role-toggle" :class="{ active: profileForm.isHeadTeacher }" @tap="profileForm.isHeadTeacher = !profileForm.isHeadTeacher">
            {{ profileForm.isHeadTeacher ? '当前身份：班主任' : '当前身份：任课老师' }}
          </button>
          <text v-if="profileFormError" class="form-error">{{ profileFormError }}</text>
          <button class="primary-btn" :disabled="saving" @tap="saveProfile">{{ saving ? '保存中...' : '保存资料' }}</button>
        </view>
      </view>
    </view>

    <view v-if="boundaryDialogOpen" class="modal-mask" @tap="closeBoundaryDialog">
      <view class="modal-card boundary-dialog" @tap.stop>
        <view class="row between">
          <text class="section-title">{{ boundaryDialogTitle }}</text>
          <button class="ghost-btn mini" @tap="closeBoundaryDialog">关闭</button>
        </view>
        <text class="body">{{ boundaryDialogMessage }}</text>
        <view class="boundary-list">
          <view v-for="item in boundaryDialogItems" :key="item.label" class="boundary-row">
            <text class="tag">{{ item.status }}</text>
            <view class="boundary-copy">
              <text class="body strong">{{ item.label }}</text>
              <text class="caption">{{ item.detail }}</text>
            </view>
          </view>
        </view>
        <button class="primary-btn" @tap="closeBoundaryDialog">{{ boundaryDialogActionText }}</button>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api, listItems, listPageInfo } from '../../api/client'
import { confirmAction, ensureLoggedIn, errorMessage, hasText, showToast, trimmed, withinLength } from '../../utils/ui'
import AppState from '../../components/AppState.vue'
import AppIcon from '../../components/AppIcon.vue'

const profile = ref({})
const favorites = ref([])
const aiLogs = ref([])
const wechatSession = ref(null)
const loading = ref(false)
const loadingFavorites = ref(false)
const loadingAiLogs = ref(false)
const saving = ref(false)
const loggingOut = ref(false)
const deletingAccount = ref(false)
const exportingAccount = ref(false)
const error = ref('')
const profileFormOpen = ref(false)
const favoriteFormOpen = ref(false)
const profileForm = ref({})
const favoriteForm = ref(defaultFavoriteForm())
const profileFormError = ref('')
const favoriteFormError = ref('')
const favoriteQuery = ref('')
const aiLogQuery = ref('')
const favoritesOffset = ref(0)
const favoritesHasMore = ref(false)
const favoritesPageSize = 20
const aiLogsOffset = ref(0)
const aiLogsHasMore = ref(false)
const aiLogsPageSize = 20
const entitlements = ref({})
const notificationSettings = ref({})
const sync = ref({})
const boundaryDialogOpen = ref(false)
const boundaryDialogTitle = ref('')
const boundaryDialogMessage = ref('')
const boundaryDialogItems = ref([])
const boundaryDialogActionText = ref('知道了')
const showDevLogin = api.isDevAuthAvailable()

const avatarText = computed(() => (profile.value.name || '微').slice(0, 1))
const roleText = computed(() => profile.value.isHeadTeacher ? `${profile.value.stage || '学段'}班主任` : (profile.value.stage || '任课老师'))
const proText = computed(() => ({ free: '免费版', trial: 'Pro 试用', pro: 'Pro 已开通', expired: 'Pro 已过期' })[profile.value.proStatus] || '免费版')
const reminderText = computed(() => profile.value.reminderPolicy === 'low_interrupt' ? '低打扰提醒' : '普通提醒')
const loginText = computed(() => {
  if (wechatSession.value?.openId) return `已登录：${wechatSession.value.openId}`
  return showDevLogin ? '本地调试可使用开发登录；生产环境仅保留真实微信授权。' : '当前未检测到有效微信登录，请重新登录。'
})
const loginButtonText = computed(() => wechatSession.value?.openId ? '开发重登' : '开发登录')
const entitlementText = computed(() => ({ free: '免费版', pro: 'Pro 权益' })[entitlements.value.plan] || proText.value)
const entitlementMessage = computed(() => entitlements.value.checkoutMessage || 'AI 沟通、云同步、导入课表、素材库与高级提醒。')
const notificationText = computed(() => notificationSettings.value.reminderPolicy === 'normal' ? '普通提醒' : '低打扰提醒')
const notificationMessage = computed(() => notificationSettings.value.message || '点击切换普通提醒与低打扰提醒策略。')
const syncText = computed(() => ({ local_persistence: '账号内保存', not_configured: '未配置' })[sync.value.cloudSyncStatus] || '待检查')
const syncMessage = computed(() => sync.value.message || '点击刷新云同步、对象存储和账号数据状态。')
const checkoutReady = computed(() => entitlements.value.checkoutStatus === 'ready')
const notificationReady = computed(() => notificationSettings.value.providerStatus === 'ready')
const objectStorageReady = computed(() => sync.value.objectStorageStatus === 'ready')
const favoriteTypeOptions = [
  { value: 'communication_template', label: '沟通模板' },
  { value: 'ai_praise', label: 'AI夸夸' },
  { value: 'class_feedback', label: '班级反馈' }
]
const favoriteTypeLabels = favoriteTypeOptions.map((item) => item.label)

onShow(load)

async function load() {
  if (!ensureLoggedIn(api)) return
  loading.value = true
  error.value = ''
  try {
    wechatSession.value = api.currentWechatSession()
    profile.value = await api.me()
    await Promise.all([fetchFavorites({ reset: true }), fetchAiLogs({ reset: true }), loadBoundaries()])
  } catch (err) {
    error.value = errorMessage(err, '账号数据加载失败')
  } finally {
    loading.value = false
  }
}

async function fetchFavorites({ reset = false } = {}) {
  if (loadingFavorites.value) return
  loadingFavorites.value = true
  try {
    const offset = reset ? 0 : favoritesOffset.value
    const response = await api.favorites('', { q: favoriteQuery.value, limit: favoritesPageSize, offset })
    const data = listItems(response)
    const page = listPageInfo(response, favoritesPageSize, offset)
    favorites.value = reset ? data : [...favorites.value, ...data]
    favoritesOffset.value = page.nextOffset
    favoritesHasMore.value = page.hasMore
  } finally {
    loadingFavorites.value = false
  }
}

async function searchFavorites() {
  try {
    await fetchFavorites({ reset: true })
  } catch (err) {
    showToast(errorMessage(err, '收藏搜索失败'))
  }
}

async function loadMoreFavorites() {
  try {
    await fetchFavorites()
  } catch (err) {
    showToast(errorMessage(err, '收藏加载失败'))
  }
}

async function fetchAiLogs({ reset = false } = {}) {
  if (loadingAiLogs.value) return
  loadingAiLogs.value = true
  try {
    const offset = reset ? 0 : aiLogsOffset.value
    const response = await api.aiGenerations('', { q: aiLogQuery.value, limit: aiLogsPageSize, offset })
    const data = listItems(response)
    const page = listPageInfo(response, aiLogsPageSize, offset)
    aiLogs.value = reset ? data : [...aiLogs.value, ...data]
    aiLogsOffset.value = page.nextOffset
    aiLogsHasMore.value = page.hasMore
  } finally {
    loadingAiLogs.value = false
  }
}

async function searchAiLogs() {
  try {
    await fetchAiLogs({ reset: true })
  } catch (err) {
    showToast(errorMessage(err, 'AI 记录搜索失败'))
  }
}

async function loadMoreAiLogs() {
  try {
    await fetchAiLogs()
  } catch (err) {
    showToast(errorMessage(err, 'AI 记录加载失败'))
  }
}

async function mockWechatLogin() {
  try {
    const session = await api.wechatMockLogin({
      code: `dev-${Date.now()}`,
      nickName: profile.value.name || '林小微'
    })
    wechatSession.value = session
    profile.value = session.profile || profile.value
    showToast('微信登录成功')
  } catch (err) {
    showToast(errorMessage(err, '微信登录失败'))
  }
}

function openProfileForm() {
  profileForm.value = {
    ...profile.value,
    name: profile.value.name || '',
    school: profile.value.school || '',
    stage: profile.value.stage || '',
    subject: profile.value.subject || ''
  }
  profileFormError.value = ''
  profileFormOpen.value = true
}

function closeProfileForm() {
  profileFormOpen.value = false
  profileFormError.value = ''
}

async function saveProfile() {
  if (saving.value) return
  const validation = validateProfileForm(profileForm.value)
  if (validation) {
    profileFormError.value = validation
    return
  }
  saving.value = true
  profileFormError.value = ''
  try {
    profile.value = await api.updateMe(cleanProfilePayload(profileForm.value))
    closeProfileForm()
    showToast('已更新教师资料')
  } catch (err) {
    profileFormError.value = errorMessage(err, '资料保存失败')
    showToast(profileFormError.value)
  } finally {
    saving.value = false
  }
}

async function toggleReminderPolicy() {
  try {
    const nextPolicy = profile.value.reminderPolicy === 'low_interrupt' ? 'normal' : 'low_interrupt'
    notificationSettings.value = await api.updateNotificationSettings({ reminderPolicy: nextPolicy })
    profile.value = { ...profile.value, reminderPolicy: notificationSettings.value.reminderPolicy }
    openBoundaryDialog({
      title: '提醒与通知',
      message: notificationSettings.value.message || '提醒偏好已保存。系统推送 provider 与客户端通知权限接入后，会按这个策略触达。',
      items: notificationItems(),
      actionText: '好的'
    })
  } catch (err) {
    showToast(errorMessage(err, '提醒偏好更新失败'))
  }
}

async function showNotificationStatus() {
  if (!notificationSettings.value.providerStatus) await loadBoundaries()
  openBoundaryDialog({
    title: '提醒与通知',
    message: notificationSettings.value.message || '服务端已保存提醒偏好；系统推送 provider 与客户端通知权限尚未接入。',
    items: notificationItems(),
    actionText: '知道了'
  })
}

async function loadBoundaries() {
  try {
    const [nextEntitlements, nextNotifications, nextSync] = await Promise.all([
      api.entitlements(),
      api.notificationSettings(),
      api.syncStatus()
    ])
    entitlements.value = nextEntitlements
    notificationSettings.value = nextNotifications
    sync.value = nextSync
  } catch (err) {
    showToast(errorMessage(err, '权益状态加载失败'))
  }
}

function openFavoriteForm() {
  favoriteForm.value = defaultFavoriteForm()
  favoriteFormError.value = ''
  favoriteFormOpen.value = true
}

function closeFavoriteForm() {
  favoriteFormOpen.value = false
  favoriteFormError.value = ''
}

function setFavoriteType(event) {
  const option = favoriteTypeOptions[event.detail.value]
  if (option) favoriteForm.value.type = option.value
}

async function submitFavorite() {
  if (saving.value) return
  const validation = validateFavoriteForm(favoriteForm.value)
  if (validation) {
    favoriteFormError.value = validation
    return
  }
  saving.value = true
  favoriteFormError.value = ''
  try {
    const item = await api.createFavorite(cleanFavoritePayload(favoriteForm.value))
    favorites.value.unshift(item)
    favoritesOffset.value += 1
    closeFavoriteForm()
    showToast('已加入收藏')
  } catch (err) {
    favoriteFormError.value = errorMessage(err, '收藏失败')
    showToast(favoriteFormError.value)
  } finally {
    saving.value = false
  }
}

async function removeFavorite(id) {
  const confirmed = await confirmAction({
    title: '删除收藏',
    content: '删除后这条素材会从收藏库移除，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) return
  try {
    await api.deleteFavorite(id)
    favorites.value = favorites.value.filter((item) => item.id !== id)
    favoritesOffset.value = Math.max(0, favoritesOffset.value - 1)
    showToast('已删除收藏')
  } catch (err) {
    showToast(errorMessage(err, '删除收藏失败'))
  }
}

function openAiLog(log) {
  if (!log?.id) return
  uni.navigateTo({ url: `/pages/profile/ai-log-detail?id=${encodeURIComponent(log.id)}` })
}

async function removeAiLog(log) {
  if (!log?.id) return
  const confirmed = await confirmAction({
    title: '删除 AI 记录',
    content: '删除后这条 AI 输入、输出和使用审计会从当前账号移除，确认删除吗？',
    confirmText: '删除'
  })
  if (!confirmed) return
  try {
    await api.deleteAIGeneration(log.id)
    aiLogs.value = aiLogs.value.filter((item) => item.id !== log.id)
    aiLogsOffset.value = Math.max(0, aiLogsOffset.value - 1)
    showToast('已删除 AI 记录')
  } catch (err) {
    showToast(errorMessage(err, 'AI 记录删除失败'))
  }
}

async function showBenefits() {
  if (!entitlements.value.features?.length) await loadBoundaries()
  openBoundaryDialog({
    title: '微光 Pro 权益',
    message: entitlements.value.checkoutMessage || '当前仅展示账号权益状态；支付通道接入前不会创建真实订单或变更会员状态。',
    items: entitlementItems(),
    actionText: '知道了'
  })
}

async function showSyncStatus() {
  await loadBoundaries()
  openBoundaryDialog({
    title: '数据同步状态',
    message: sync.value.message || '当前数据按账号保存在服务端；对象存储和跨端云同步接入后，会在这里显示真实同步状态。',
    items: syncItems(),
    actionText: '好的'
  })
}

async function exportAccountData() {
  if (exportingAccount.value) return
  exportingAccount.value = true
  try {
    const data = await api.exportMe()
    const text = JSON.stringify(data, null, 2)
    const filename = `littlelight-account-export-${new Date().toISOString().slice(0, 10)}.json`
    // #ifdef H5
    const blob = new Blob([text], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    showToast('已导出账号数据')
    // #endif
    // #ifndef H5
    uni.setClipboardData({ data: text })
    showToast('已复制账号数据 JSON')
    // #endif
  } catch (err) {
    showToast(errorMessage(err, '账号数据导出失败'))
  } finally {
    exportingAccount.value = false
  }
}

async function deleteAccount() {
  if (deletingAccount.value) return
  const confirmed = await confirmAction({
    title: '删除账号与数据',
    content: '这会删除当前账号下的课程、提醒、家长档案、沟通记录、疗愈记录、AI 记录、收藏和服务端 session，确认继续吗？',
    confirmText: '删除'
  })
  if (!confirmed) return
  deletingAccount.value = true
  try {
    await api.deleteMe()
    api.logout()
    showToast('账号与数据已删除')
    uni.reLaunch({ url: '/pages/login/index' })
  } catch (err) {
    showToast(errorMessage(err, '账号删除失败'))
  } finally {
    deletingAccount.value = false
  }
}

function openBoundaryDialog({ title, message, items = [], actionText = '知道了' }) {
  boundaryDialogTitle.value = title
  boundaryDialogMessage.value = message
  boundaryDialogItems.value = items
  boundaryDialogActionText.value = actionText
  boundaryDialogOpen.value = true
}

function closeBoundaryDialog() {
  boundaryDialogOpen.value = false
}

function entitlementItems() {
  const features = entitlements.value.features?.length ? entitlements.value.features : ['AI 沟通与安全审计', '课表和名单导入预览', '家长档案与素材库']
  return [
    { label: '当前方案', status: entitlementText.value, detail: entitlements.value.status ? `账号权益状态：${proText.value}` : '当前账号按免费版能力运行。' },
    { label: '支付入口', status: checkoutStatusText(entitlements.value.checkoutStatus), detail: entitlements.value.checkoutMessage || '支付 provider 未配置前只展示状态，不会扣费。' },
    ...features.map((feature) => ({ label: feature, status: 'MVP', detail: '已纳入当前账号能力或当前阶段范围，不代表付费闭环已接入。' }))
  ]
}

function notificationItems() {
  return [
    { label: '提醒策略', status: notificationText.value, detail: notificationSettings.value.reminderPolicy === 'normal' ? '待办会按普通提醒策略保存。' : '待办会按低打扰策略保存，减少非必要打断。' },
    { label: '推送 provider', status: providerStatusText(notificationSettings.value.providerStatus), detail: notificationSettings.value.message || '服务端已保存偏好，推送通道尚未接入。' },
    { label: '客户端权限', status: notificationSettings.value.permissionRequired ? '需授权' : '已授权', detail: notificationSettings.value.permissionRequired ? '小程序或 App 端接入通知权限后，需要用户主动授权。' : '当前客户端已满足通知权限要求。' }
  ]
}

function syncItems() {
  return [
    { label: '账号数据', status: syncStatusText(sync.value.cloudSyncStatus), detail: sync.value.cloudSyncStatus === 'local_persistence' ? '课程、提醒、家长档案等已按登录账号保存在后端数据库。' : '等待云同步服务配置。' },
    { label: '对象存储', status: providerStatusText(sync.value.objectStorageStatus), detail: sync.value.objectStorageStatus === 'ready' ? '文件和素材可使用对象存储。' : '当前导入文件即时解析，不做长期对象存储。' },
    { label: '最近同步', status: sync.value.lastSyncedAt ? '已同步' : '待接入', detail: sync.value.lastSyncedAt ? formatTime(sync.value.lastSyncedAt) : '跨端云同步时间线尚未接入。' }
  ]
}

function checkoutStatusText(status) {
  return ({ not_configured: '未配置', unavailable: '暂不可用', ready: '可使用' })[status] || '未配置'
}

function providerStatusText(status) {
  return ({ not_configured: '未配置', ready: '已接入' })[status] || '未配置'
}

function syncStatusText(status) {
  return ({ local_persistence: '账号内保存', not_configured: '未配置', ready: '已同步' })[status] || '待检查'
}

async function logout() {
  if (loggingOut.value) return
  loggingOut.value = true
  try {
    await api.logoutRemote()
    showToast('已退出登录')
  } catch (err) {
    api.logout()
    showToast(errorMessage(err, '已清除本地登录态'))
  } finally {
    loggingOut.value = false
    uni.reLaunch({ url: '/pages/login/index' })
  }
}

function favoriteTypeText(type) {
  return ({ communication_template: '沟通模板', ai_praise: 'AI夸夸', class_feedback: '班级反馈' })[type] || '素材'
}

function defaultFavoriteForm() {
  return { type: 'communication_template', title: '', content: '' }
}

function validateProfileForm(form) {
  if (!hasText(form.name)) return '请填写姓名'
  if (!withinLength(form.name, 40)) return '姓名最多 40 个字'
  if (!withinLength(form.school, 80)) return '学校最多 80 个字'
  if (!withinLength(form.stage, 40)) return '学段最多 40 个字'
  if (!withinLength(form.subject, 40)) return '学科最多 40 个字'
  return ''
}

function validateFavoriteForm(form) {
  if (!hasText(form.title) || !hasText(form.content)) return '请填写标题和内容'
  if (!withinLength(form.title, 80)) return '素材标题最多 80 个字'
  if (!withinLength(form.content, 2000)) return '素材内容最多 2000 个字'
  return ''
}

function cleanProfilePayload(form) {
  return {
    ...form,
    name: trimmed(form.name),
    school: trimmed(form.school),
    stage: trimmed(form.stage),
    subject: trimmed(form.subject)
  }
}

function cleanFavoritePayload(form) {
  return {
    ...form,
    title: trimmed(form.title),
    content: trimmed(form.content)
  }
}

function scenarioText(scenario) {
  return ({ parent_drafts: '家校草稿', praise: 'AI夸夸' })[scenario] || 'AI 生成'
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

function outputPreview(log) {
  const output = log.output || {}
  if (output.content) return output.content
  if (output.draft?.content) return output.draft.content
  if (Array.isArray(output.drafts) && output.drafts[0]?.content) return output.drafts[0].content
  return '已生成内容，进入详情后可做质量复盘。'
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
.header, .section-head { padding: 0 4rpx 14rpx; }
.profile { display: flex; gap: 24rpx; align-items: center; }
.profile-main { flex: 1; min-width: 0; }
.login-card { background: rgba(255,255,255,.72); }
.avatar { width: 112rpx; height: 112rpx; border-radius: 32rpx; background: linear-gradient(145deg,#758ce7,#64bfd1); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 44rpx; font-weight: 950; box-shadow: 0 18rpx 34rpx rgba(89,102,209,.20); }
.block { display: block; margin: 8rpx 0 16rpx; }
.tag-row { flex-wrap: wrap; }
.material { margin: 20rpx 0; padding: 32rpx; border-radius: 32rpx; background: linear-gradient(145deg,rgba(255,210,229,.82),rgba(255,232,185,.72)); display: flex; flex-direction: column; gap: 22rpx; }
.material-copy { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 8rpx; }
	.boundary-card { display: flex; flex-direction: column; gap: 16rpx; }
	.danger-zone { background: rgba(255,241,242,.9); }
	.boundary-meta { display: flex; align-items: center; justify-content: space-between; gap: 16rpx; padding-top: 14rpx; border-top: 1px solid rgba(8,145,178,.12); }
.mutedTag { color: #6b778f; background: rgba(247,252,253,.96); }
.ai-log-card { display: flex; flex-direction: column; gap: 18rpx; }
.ai-log-foot { align-items: flex-end; }
.small { min-height: 62rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 54rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
.search-bar { display: flex; gap: 12rpx; align-items: center; }
.search-input { flex: 1; min-width: 0; }
.load-more { margin: 8rpx 0 28rpx; }
.form-grid { display: flex; flex-direction: column; gap: 18rpx; }
.boundary-dialog { gap: 26rpx; }
.boundary-list { display: flex; flex-direction: column; gap: 16rpx; }
.boundary-row { padding: 22rpx; border-radius: 24rpx; background: rgba(255,255,255,.58); border: 1rpx solid rgba(255,255,255,.78); display: flex; gap: 16rpx; align-items: flex-start; }
.boundary-copy { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 6rpx; }
.strong { color: #14313b; font-weight: 900; }
.textarea.short { min-height: 140rpx; }
.role-toggle { min-height: 78rpx; border-radius: 999rpx; background: linear-gradient(180deg, rgba(255,255,255,.94), rgba(250,253,255,.84)); color: #506075; font-weight: 900; border: 1px solid rgba(97,116,166,.16); box-shadow: 0 8rpx 18rpx rgba(73,91,146,.08), inset 0 1px 0 rgba(255,255,255,.96); }
.role-toggle.active { color: #fff; border-color: rgba(255,255,255,.42); background: linear-gradient(135deg,#6f86df,#52b8cf); box-shadow: 0 12rpx 22rpx rgba(74,111,190,.18), inset 0 1px 0 rgba(255,255,255,.30); }
</style>
