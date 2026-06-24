<template>
  <view v-if="shouldShow" class="voice-dictation">
    <button
      class="voice-fab"
      :class="{ active: recording, connecting: status === 'connecting', failed: status === 'error' }"
      aria-label="开始语音听写"
      data-testid="voice-dictation-button"
      @tap="togglePanel"
    >
      <view v-if="recording" class="pulse-ring"></view>
      <text class="voice-icon"><AppIcon :name="recording ? 'square' : 'mic'" /></text>
    </button>

    <view v-if="panelOpen" class="voice-mask" @tap="closePanel">
      <view class="voice-panel" @tap.stop>
        <view class="voice-panel-head">
          <view>
            <text class="caption">{{ statusText }}</text>
            <view class="voice-title">{{ timerText }}</view>
          </view>
          <button class="ghost-btn mini" @tap="closePanel"><AppIcon name="x" /></button>
        </view>

        <view class="language-tabs">
          <button
            v-for="item in languages"
            :key="item.value"
            class="language-tab"
            :class="{ active: language === item.value }"
            :disabled="recording || status === 'connecting'"
            @tap="language = item.value"
          >
            {{ item.label }}
          </button>
        </view>

        <view class="transcript-box" :class="{ empty: !displayText }">
          <textarea
            class="transcript-textarea"
            v-model="editableText"
            maxlength="4000"
            placeholder="点击下方按钮，说出来。识别结果会实时出现在这里。"
            aria-label="语音听写识别文本"
            data-testid="voice-dictation-textarea"
          />
          <text v-if="unstableText" class="unstable-text">{{ unstableText }}</text>
        </view>

        <text v-if="error" class="form-error">{{ error }}</text>

        <view class="voice-actions">
          <button class="primary-btn voice-main-action" :disabled="status === 'ending'" @tap="toggleRecording">
            <text class="btn-icon"><AppIcon :name="recording ? 'square' : 'mic'" /></text>
            {{ recording ? '结束听写' : '开始听写' }}
          </button>
          <button class="ghost-btn voice-action" :disabled="!editableText" @tap="copyText"><AppIcon name="copy" /></button>
          <button class="ghost-btn voice-action" :disabled="!editableText" @tap="clearText"><AppIcon name="trash" /></button>
          <button class="ghost-btn voice-action" :disabled="!editableText" @tap="insertText"><AppIcon name="check" /></button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from '../api/client'
import { createDictationSocket, createPCMRecorder, currentRoutePath, dictationTargetForRoute, emitDictationText } from '../utils/dictation'
import { showToast } from '../utils/ui'
import AppIcon from './AppIcon.vue'

const languageKey = 'littlelight_dictation_language'
const draftKey = 'littlelight_dictation_drafts'
const languages = [
  { value: 'zh_cn', label: '中文普通话' },
  { value: 'en_us', label: 'English' }
]

const panelOpen = ref(false)
const status = ref('idle')
const language = ref(uni.getStorageSync(languageKey) || 'zh_cn')
const stableText = ref('')
const unstableText = ref('')
const editableText = ref('')
const error = ref('')
const seconds = ref(0)
const routePath = ref(currentRoutePath())
const loggedIn = ref(api.isLoggedIn())
let ws = null
let recorder = null
let timer = null
let routeTimer = null
let seq = 0

const shouldShow = computed(() => loggedIn.value && !routePath.value.includes('/login/'))
const recording = computed(() => status.value === 'recording')
const displayText = computed(() => editableText.value || stableText.value || unstableText.value)
const statusText = computed(() => ({
  idle: '点一下，说出来',
  connecting: '正在连接听写服务',
  recording: '正在听，你可以继续说',
  ending: '正在整理最后一句',
  error: '听写暂时不可用'
})[status.value] || '语音听写')
const timerText = computed(() => {
  const minutes = Math.floor(seconds.value / 60)
  const rest = seconds.value % 60
  return `${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`
})

watch(language, (value) => {
  uni.setStorageSync(languageKey, value)
})

onBeforeUnmount(stopEverything)
onMounted(() => {
  refreshVisibility()
  routeTimer = setInterval(refreshVisibility, 800)
})
onBeforeUnmount(() => {
  if (routeTimer) clearInterval(routeTimer)
})

function refreshVisibility() {
  routePath.value = currentRoutePath()
  loggedIn.value = api.isLoggedIn()
}

function togglePanel() {
  refreshVisibility()
  panelOpen.value = true
}

async function closePanel() {
  if (recording.value || status.value === 'connecting') {
    await stopRecording()
  }
  if (editableText.value.trim()) {
    saveDraft(editableText.value)
  }
  panelOpen.value = false
  status.value = 'idle'
  error.value = ''
}

async function toggleRecording() {
  if (recording.value || status.value === 'connecting') {
    await stopRecording()
    return
  }
  await startRecording()
}

async function startRecording() {
  if (!api.isLoggedIn()) {
    showToast('请先登录后使用语音听写')
    return
  }
  stopEverything()
  status.value = 'connecting'
  error.value = ''
  stableText.value = editableText.value.trim()
  unstableText.value = ''
  seq = 0
  try {
    ws = createDictationSocket(api.apiWSURL('/dictation/stream', {
      token: api.currentWechatSession()?.sessionToken || '',
      language: language.value,
      sampleRate: 16000
    }))
    ws.onMessage(handleSocketMessage)
    ws.onError(() => {
      fail('听写连接异常，请稍后重试')
    })
    ws.onClose(() => {
      if (status.value === 'recording' || status.value === 'connecting') {
        fail('听写连接已断开，已保留当前文字')
      }
    })
    await waitForOpen(ws)
    sendSocket({ type: 'start', language: language.value, sampleRate: 16000 })
    recorder = createPCMRecorder({
      sampleRate: 16000,
      frameSamples: 640,
      duration: 280,
      onFrame: (audio) => sendSocket({ type: 'audio', seq: ++seq, audio }),
      onError: (message) => fail(message || '录音异常，请检查麦克风权限')
    })
    await recorder.start()
    status.value = 'recording'
    startTimer()
  } catch (err) {
    fail(err?.message || '无法启动语音听写')
    stopEverything()
  }
}

async function stopRecording() {
  if (status.value !== 'recording' && status.value !== 'connecting') return
  status.value = 'ending'
  await stopRecorder()
  sendSocket({ type: 'stop' })
  stopTimer()
  setTimeout(() => {
    if (status.value === 'ending') status.value = 'idle'
  }, 1800)
}

function handleSocketMessage(event) {
  let data = null
  try {
    data = JSON.parse(event.data)
  } catch (err) {
    return
  }
  if (data.type === 'ready') {
    return
  }
  if (data.type === 'partial') {
    stableText.value = data.stableText || data.text || stableText.value
    unstableText.value = data.unstableText || ''
    editableText.value = stableText.value
    return
  }
  if (data.type === 'final' || data.type === 'done') {
    stableText.value = data.text || data.stableText || stableText.value
    unstableText.value = ''
    editableText.value = stableText.value
    status.value = 'idle'
    stopTimer()
    return
  }
  if (data.type === 'error') {
    fail(data.message || '语音听写失败')
  }
}

function waitForOpen(socket) {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error('听写服务连接超时')), 8000)
    socket.onOpen(() => {
      clearTimeout(timeout)
      resolve()
    })
    socket.onError(() => {
      clearTimeout(timeout)
      reject(new Error('听写服务连接失败'))
    })
  })
}

function sendSocket(payload) {
  if (!ws) return
  ws.send(JSON.stringify(payload))
}

function copyText() {
  const text = editableText.value.trim()
  if (!text) return
  uni.setClipboardData({ data: text })
  showToast('已复制听写文字')
}

function clearText() {
  stableText.value = ''
  unstableText.value = ''
  editableText.value = ''
}

function insertText() {
  const text = editableText.value.trim()
  if (!text) return
  const target = dictationTargetForRoute(currentRoutePath())
  if (emitDictationText(text, target)) {
    showToast(target === 'draft' ? '已保存为语音草稿' : '已插入当前页面')
    if (target === 'draft') saveDraft(text)
    panelOpen.value = false
  }
}

function saveDraft(text) {
  const value = String(text || '').trim()
  if (!value) return
  const drafts = uni.getStorageSync(draftKey) || []
  drafts.unshift({
    id: `dict_${Date.now()}`,
    text: value,
    route: currentRoutePath(),
    createdAt: new Date().toISOString()
  })
  uni.setStorageSync(draftKey, drafts.slice(0, 20))
}

function fail(message) {
  error.value = message
  status.value = 'error'
  stopRecorder()
  stopTimer()
}

function startTimer() {
  stopTimer()
  seconds.value = 0
  timer = setInterval(() => {
    seconds.value += 1
    if (seconds.value >= 55) stopRecording()
  }, 1000)
}

function stopTimer() {
  if (!timer) return
  clearInterval(timer)
  timer = null
}

async function stopRecorder() {
  if (!recorder) return
  await recorder.stop?.()
  recorder = null
}

function stopEverything() {
  stopRecorder()
  stopTimer()
  ws?.close()
  ws = null
}
</script>

<style scoped>
.voice-dictation {
  position: fixed;
  inset: 0;
  z-index: 5000;
  pointer-events: none;
}

.voice-fab {
  position: absolute;
  left: 50%;
  bottom: calc(96rpx + env(safe-area-inset-bottom));
  transform: translateX(-50%);
  width: 116rpx;
  height: 116rpx;
  padding: 0;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: linear-gradient(135deg, #ffbd6b 0%, #65c8d9 100%);
  box-shadow: 0 22rpx 46rpx rgba(75, 126, 169, .28), inset 0 1px 0 rgba(255,255,255,.45);
  pointer-events: auto;
}

.voice-fab.active {
  background: linear-gradient(135deg, #ff9f75 0%, #6f86df 100%);
}

.voice-fab.failed {
  background: linear-gradient(135deg, #ef8f98 0%, #d85d73 100%);
}

.voice-fab:active {
  transform: translate(-50%, 1px) scale(.98);
}

.voice-icon {
  position: relative;
  z-index: 2;
  font-size: 48rpx;
}

.pulse-ring {
  position: absolute;
  inset: -10rpx;
  border-radius: 999rpx;
  border: 4rpx solid rgba(255,255,255,.58);
  animation: voicePulse 1.2s ease-out infinite;
}

.voice-mask {
  position: fixed;
  inset: 0;
  z-index: 4999;
  display: flex;
  align-items: flex-end;
  padding: 32rpx;
  box-sizing: border-box;
  background: rgba(35, 37, 62, .22);
  backdrop-filter: blur(10rpx);
  pointer-events: auto;
  transform: translateX(0);
}

.voice-panel {
  width: min(100%, 720rpx);
  max-height: 86vh;
  margin: 0 auto;
  padding: 32rpx;
  padding-bottom: calc(32rpx + env(safe-area-inset-bottom));
  border-radius: 34rpx 34rpx 26rpx 26rpx;
  border: 1rpx solid rgba(255,255,255,.82);
  background: linear-gradient(145deg, rgba(255,255,255,.92), rgba(246,250,255,.78));
  box-shadow: 0 24rpx 64rpx rgba(73,91,146,.20), inset 0 1rpx 0 rgba(255,255,255,.92);
  backdrop-filter: blur(28rpx) saturate(1.18);
  box-sizing: border-box;
}

.voice-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18rpx;
}

.voice-title {
  margin-top: 4rpx;
  font-size: 42rpx;
  line-height: 1.1;
  font-weight: 950;
  color: #172039;
}

.language-tabs {
  margin-top: 24rpx;
  padding: 6rpx;
  border-radius: 999rpx;
  background: rgba(236,242,255,.86);
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6rpx;
}

.language-tab {
  min-height: 70rpx;
  border-radius: 999rpx;
  font-size: 24rpx;
  font-weight: 900;
  color: #687391;
}

.language-tab.active {
  color: #fff;
  background: linear-gradient(135deg, #6f86df, #56b9cf);
  box-shadow: 0 10rpx 18rpx rgba(74,111,190,.16);
}

.transcript-box {
  position: relative;
  margin-top: 22rpx;
  min-height: 300rpx;
  border-radius: 26rpx;
  border: 1rpx solid rgba(255,255,255,.88);
  background: rgba(255,255,255,.72);
  overflow: hidden;
}

.transcript-textarea {
  width: 100%;
  min-height: 300rpx;
  padding: 24rpx;
  box-sizing: border-box;
  color: #182033;
  font-size: 29rpx;
  line-height: 1.55;
  background: transparent;
}

.unstable-text {
  position: absolute;
  left: 24rpx;
  right: 24rpx;
  bottom: 18rpx;
  color: #7f89a4;
  font-size: 25rpx;
}

.voice-actions {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 78rpx 78rpx 78rpx;
  gap: 12rpx;
  margin-top: 22rpx;
  align-items: center;
}

.voice-main-action {
  min-width: 0;
}

.voice-action {
  width: 78rpx;
  min-height: 78rpx;
  padding: 0;
}

.mini {
  min-height: 58rpx;
  min-width: 58rpx;
  padding: 0 18rpx;
}

@keyframes voicePulse {
  0% { transform: scale(.85); opacity: .72; }
  100% { transform: scale(1.28); opacity: 0; }
}
</style>
