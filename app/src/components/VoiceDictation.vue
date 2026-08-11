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
          <view class="voice-head-main">
            <view class="voice-panel-icon" :class="{ active: recording, failed: status === 'error' }">
              <AppIcon :name="recording ? 'square' : 'mic'" />
            </view>
            <view>
              <text class="caption">{{ statusText }}</text>
              <view class="voice-title">{{ timerText }}</view>
            </view>
          </view>
          <button class="voice-close-btn" aria-label="关闭听写" @tap="closePanel"><AppIcon name="x" /></button>
        </view>

        <view class="language-tabs">
          <view
            v-for="item in languages"
            :key="item.value"
            class="language-tab"
            :class="{ active: language === item.value, disabled: recording || status === 'connecting' }"
            role="button"
            :aria-label="item.label"
            @tap="selectLanguage(item.value)"
          >
            <text class="language-tab-label">{{ item.label }}</text>
          </view>
        </view>

        <view class="transcript-box" :class="{ empty: !displayText }">
          <textarea
            class="transcript-textarea"
            v-model="editableText"
            maxlength="20000"
            placeholder="点击下方按钮，说出来。识别结果会实时出现在这里。"
            aria-label="语音听写识别文本"
            data-testid="voice-dictation-textarea"
          />
          <text v-if="unstableText" class="unstable-text">{{ unstableText }}</text>
        </view>

        <text v-if="error" class="form-error">{{ error }}</text>

        <view class="voice-actions">
          <button class="primary-btn voice-main-action" :disabled="status === 'ending'" @tap="toggleRecording">
            <view class="btn-icon"><AppIcon :name="recording ? 'square' : 'mic'" /></view>
            {{ recording ? '结束听写' : '开始听写' }}
          </button>
          <button class="voice-tool-btn" :disabled="!editableText" @tap="copyText">
            <view class="tool-icon"><AppIcon name="copy" /></view>
            <text>复制</text>
          </button>
          <button class="voice-tool-btn danger" :disabled="!editableText" @tap="clearText">
            <view class="tool-icon"><AppIcon name="trash" /></view>
            <text>清空</text>
          </button>
          <button class="voice-tool-btn success" :disabled="!editableText" @tap="insertText">
            <view class="tool-icon"><AppIcon name="check" /></view>
            <text>插入</text>
          </button>
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
const maxDictationSeconds = 5 * 60
const segmentSeconds = 50
const segmentStopWaitMs = 1400
const languages = [
  { value: 'zh_cn', label: '中文普通话' },
  { value: 'en_us', label: 'English' }
]

const panelOpen = ref(false)
const status = ref('idle')
const language = ref(uni.getStorageSync(languageKey) || 'zh_cn')
const committedText = ref('')
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
let nextSegmentAt = segmentSeconds
let currentSegmentText = ''
let segmentFinalCommitted = false
let rotatingSegment = false
let segmentDoneResolver = null

const shouldShow = computed(() => !routePath.value.includes('/login/'))
const recording = computed(() => status.value === 'recording')
const displayText = computed(() => editableText.value || stableText.value || unstableText.value)
const statusText = computed(() => ({
  idle: '点一下，说出来',
  connecting: '正在连接听写服务',
  recording: '正在听，可以连续说到 5 分钟',
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

function selectLanguage(value) {
  if (recording.value || status.value === 'connecting') return
  language.value = value
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
  committedText.value = editableText.value.trim()
  stableText.value = committedText.value
  unstableText.value = ''
  seq = 0
  try {
    await ensureMicrophoneAccess()
    await openDictationSegment()
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
    fail(dictationErrorMessage(err, '无法启动语音听写'))
    stopEverything()
  }
}

async function openDictationSegment() {
  currentSegmentText = ''
  segmentFinalCommitted = false
  const socket = createDictationSocket(api.apiWSURL('/dictation/stream', {
    token: api.currentWechatSession()?.sessionToken || '',
    language: language.value,
    sampleRate: 16000
  }))
  ws = socket
  socket.onMessage((event) => handleSocketMessage(event, socket))
  socket.onError(() => {
    if (ws === socket) fail('听写连接异常，请稍后重试')
  })
  socket.onClose(() => {
    if (ws !== socket) return
    if (status.value === 'recording' || status.value === 'connecting') {
      fail('听写连接已断开，已保留当前文字')
    }
  })
  await waitForOpen(socket)
  if (ws !== socket) return
  sendSocket({ type: 'start', language: language.value, sampleRate: 16000 }, socket)
}

async function ensureMicrophoneAccess() {
  // #ifdef H5
  if (typeof navigator === 'undefined' || !navigator.mediaDevices?.getUserMedia) {
    throw new Error('当前浏览器不支持麦克风录音')
  }
  let stream = null
  try {
    stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true
      }
    })
  } catch (err) {
    throw new Error(dictationErrorMessage(err, '无法获取麦克风权限'))
  } finally {
    stream?.getTracks?.().forEach((track) => track.stop())
  }
  // #endif
}

async function stopRecording() {
  if (status.value !== 'recording' && status.value !== 'connecting') return
  status.value = 'ending'
  rotatingSegment = false
  await stopRecorder()
  sendSocket({ type: 'stop' })
  stopTimer()
  setTimeout(() => {
    if (status.value === 'ending') status.value = 'idle'
  }, 1800)
}

function handleSocketMessage(event, socket = ws) {
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
    currentSegmentText = data.stableText || data.text || currentSegmentText
    stableText.value = joinDictationParts(committedText.value, currentSegmentText)
    unstableText.value = data.unstableText || ''
    editableText.value = stableText.value
    return
  }
  if (data.type === 'final' || data.type === 'done') {
    const finalText = data.text || data.stableText || currentSegmentText
    commitSegmentText(finalText)
    unstableText.value = ''
    segmentDoneResolver?.()
    if (rotatingSegment) return
    if (socket === ws || status.value === 'ending') {
      status.value = 'idle'
      stopTimer()
    }
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

function sendSocket(payload, socket = ws) {
  if (!socket) return false
  return socket.send(JSON.stringify(payload))
}

function copyText() {
  const text = editableText.value.trim()
  if (!text) return
  uni.setClipboardData({ data: text })
  showToast('已复制听写文字')
}

function clearText() {
  committedText.value = ''
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
  error.value = dictationErrorMessage(message, '语音听写暂时不可用')
  status.value = 'error'
  stopRecorder()
  stopTimer()
}

function dictationErrorMessage(errorValue, fallback) {
  const name = String(errorValue?.name || '')
  const message = String(errorValue?.message || errorValue || '').trim()
  const raw = `${name} ${message}`.toLowerCase()
  if (/notallowed|permission denied|permissiondismissed|denied|权限/.test(raw)) {
    return '麦克风权限被拒绝。请在浏览器地址栏或系统设置里允许此页面使用麦克风，然后重新开始听写。'
  }
  if (/notfound|devicesnotfound|no audio input|requested device not found/.test(raw)) {
    return '没有找到可用麦克风。请连接或启用麦克风后再试。'
  }
  if (/notreadable|trackstart|could not start|device in use/.test(raw)) {
    return '麦克风暂时无法使用，可能被其他应用占用。请关闭占用录音的软件后再试。'
  }
  if (/secure context|only secure origins|https/.test(raw)) {
    return '浏览器要求在 localhost 或 HTTPS 页面中使用麦克风。'
  }
  return message || fallback
}

function startTimer() {
  stopTimer()
  seconds.value = 0
  nextSegmentAt = segmentSeconds
  timer = setInterval(() => {
    seconds.value += 1
    if (seconds.value >= maxDictationSeconds) {
      showToast('已达到单次 5 分钟听写上限')
      stopRecording()
      return
    }
    if (seconds.value >= nextSegmentAt) {
      nextSegmentAt += segmentSeconds
      rotateDictationSegment()
    }
  }, 1000)
}

async function rotateDictationSegment() {
  if (rotatingSegment || status.value !== 'recording' || !ws) return
  rotatingSegment = true
  const oldSocket = ws
  ws = null
  try {
    await finishCurrentSegment(oldSocket)
    oldSocket?.close?.()
    if (status.value !== 'recording') return
    seq = 0
    await openDictationSegment()
  } catch (err) {
    if (status.value === 'recording') {
      fail(dictationErrorMessage(err, '长听写续接失败，已保留当前文字'))
    }
  } finally {
    rotatingSegment = false
  }
}

function finishCurrentSegment(socket) {
  return new Promise((resolve) => {
    let done = false
    const finish = () => {
      if (done) return
      done = true
      if (segmentDoneResolver === finish) segmentDoneResolver = null
      if (!segmentFinalCommitted) commitSegmentText(currentSegmentText)
      resolve()
    }
    segmentDoneResolver = finish
    sendSocket({ type: 'stop' }, socket)
    setTimeout(finish, segmentStopWaitMs)
  })
}

function commitSegmentText(value) {
  const text = String(value || '').trim()
  if (!text || segmentFinalCommitted) return
  committedText.value = joinDictationParts(committedText.value, text)
  stableText.value = committedText.value
  editableText.value = stableText.value
  currentSegmentText = ''
  segmentFinalCommitted = true
}

function joinDictationParts(base, next) {
  const previous = String(base || '').trim()
  const incoming = String(next || '').trim()
  if (!previous) return incoming
  if (!incoming) return previous
  if (previous.endsWith(incoming)) return previous
  const needsSpace = /[A-Za-z0-9,.;:!?]$/.test(previous) && /^[A-Za-z0-9]/.test(incoming)
  return `${previous}${needsSpace ? ' ' : ''}${incoming}`
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
  rotatingSegment = false
  segmentDoneResolver?.()
  segmentDoneResolver = null
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
  bottom: calc(23px + env(safe-area-inset-bottom, 0px));
  transform: translateX(-50%);
  width: 90rpx;
  height: 90rpx;
  padding: 0;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  border: 1rpx solid rgba(255,255,255,.72);
  background: linear-gradient(145deg, #6f86df 0%, #52b8cf 100%);
  box-shadow: 0 14rpx 30rpx rgba(73,91,146,.18), 0 0 0 8rpx rgba(255,255,255,.78), inset 0 1rpx 0 rgba(255,255,255,.42);
  backdrop-filter: blur(16rpx) saturate(1.12);
  pointer-events: auto;
}

.voice-fab.active {
  background: linear-gradient(145deg, #ef8f98 0%, #6f86df 100%);
}

.voice-fab.failed {
  background: linear-gradient(145deg, #ef8f98 0%, #d85d73 100%);
}

.voice-fab:active {
  transform: translate(-50%, 1px) scale(.98);
}

.voice-icon {
  position: relative;
  z-index: 2;
  font-size: 34rpx;
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

.voice-head-main {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 18rpx;
}

.voice-panel-icon {
  flex: 0 0 auto;
  width: 78rpx;
  height: 78rpx;
  border-radius: 26rpx;
  color: #fff;
  background: linear-gradient(145deg, #6f86df, #52b8cf);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 14rpx 26rpx rgba(82,111,190,.18), inset 0 1rpx 0 rgba(255,255,255,.32);
}

.voice-panel-icon.active {
  background: linear-gradient(145deg, #ef8f98, #6f86df);
}

.voice-panel-icon.failed {
  background: linear-gradient(145deg, #ef8f98, #d85d73);
}

.voice-panel-icon .app-icon {
  width: 38rpx;
  height: 38rpx;
}

.voice-close-btn {
  flex: 0 0 auto;
  width: 62rpx;
  height: 62rpx;
  padding: 0;
  border-radius: 999rpx;
  color: #52617f;
  background: rgba(255,255,255,.72);
  border: 1rpx solid rgba(97,116,166,.10);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8rpx 18rpx rgba(73,91,146,.07), inset 0 1rpx 0 rgba(255,255,255,.92);
}

.voice-close-btn .app-icon {
  width: 30rpx;
  height: 30rpx;
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
  position: relative;
  height: 70rpx;
  min-height: 70rpx;
  padding: 0 12rpx;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  box-sizing: border-box;
  font-size: 24rpx;
  line-height: 1;
  font-weight: 900;
  color: #687391;
  -webkit-tap-highlight-color: transparent;
}

.language-tab.active {
  color: #fff;
  background: linear-gradient(135deg, #6f86df, #56b9cf);
  box-shadow: 0 10rpx 18rpx rgba(74,111,190,.16);
}

.language-tab.disabled {
  opacity: .62;
}

.language-tab-label {
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: auto;
  height: 1em;
  text-align: center;
  line-height: 1;
  font-size: inherit;
  font-weight: inherit;
  color: inherit;
  pointer-events: none;
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12rpx;
  margin-top: 22rpx;
  align-items: center;
}

.voice-main-action {
  position: relative;
  grid-column: 1 / -1;
  min-width: 0;
  min-height: 76rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  font-size: 26rpx;
  color: #fff;
  border-radius: 24rpx;
  border: 1rpx solid rgba(255,255,255,.36);
  background: linear-gradient(135deg, #6f86df 0%, #52b8cf 100%);
  box-shadow: 0 8rpx 16rpx rgba(74,111,190,.12);
  overflow: hidden;
}

.voice-main-action::after {
  display: none !important;
  border: 0 !important;
}

.voice-main-action .btn-icon {
  width: 28rpx;
  height: 28rpx;
  background: transparent;
  position: static;
  box-shadow: none;
}

.voice-main-action .btn-icon .app-icon {
  width: 26rpx;
  height: 26rpx;
}

.voice-tool-btn {
  min-width: 0;
  min-height: 78rpx;
  padding: 0;
  border-radius: 24rpx;
  color: #586381;
  background: linear-gradient(180deg, rgba(255,255,255,.86), rgba(247,250,255,.72));
  border: 1rpx solid rgba(97,116,166,.10);
  box-shadow: 0 8rpx 16rpx rgba(73,91,146,.07), inset 0 1rpx 0 rgba(255,255,255,.95);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 5rpx;
  font-size: 20rpx;
  line-height: 1.1;
  font-weight: 900;
  transition: transform .16s ease, opacity .16s ease, background .16s ease;
}

.voice-tool-btn:active {
  transform: translateY(1px);
}

.voice-tool-btn[disabled] {
  opacity: .42;
}

.voice-tool-btn.success {
  color: #2f7f6e;
  background: linear-gradient(180deg, rgba(245,255,250,.92), rgba(231,248,241,.74));
}

.voice-tool-btn.danger {
  color: #a85c65;
  background: linear-gradient(180deg, rgba(255,248,249,.92), rgba(255,241,243,.72));
}

.tool-icon {
  width: 30rpx;
  height: 30rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tool-icon .app-icon {
  width: 28rpx;
  height: 28rpx;
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

@media (max-width: 360px) {
  .voice-panel {
    padding: 28rpx;
  }

  .voice-tool-btn {
    min-height: 74rpx;
  }
}
</style>
