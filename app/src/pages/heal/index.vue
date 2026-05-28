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
      <view class="row between"><view><text class="section-title">AI夸夸</text><text class="caption block">写下今天发生了什么</text></view><text class="tag">抱抱</text></view>
      <textarea class="textarea" v-model="content" placeholder="比如：今天课很多，还处理了家长反馈..." />
      <button class="primary-btn praise-btn" @tap="makePraise">生成一句抱抱</button>
      <view class="reply"><text class="body">{{ reply }}</text></view>
    </view>

    <view class="section-head row between">
      <text class="section-title">疗愈记录</text>
      <text class="tag">{{ entries.length }} 条</text>
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
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'

const breathing = ref(false)
const left = ref(60)
let timer = null
const content = ref('')
const reply = ref('你今天已经完成了很多复杂判断。先把肩膀放下来，剩下的事情可以一件一件处理。')
const entries = ref([])
const leftText = computed(() => `00:${String(left.value).padStart(2, '0')}`)

onShow(loadEntries)

async function loadEntries() {
  entries.value = await api.healingEntries()
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
  const entry = await api.healingEntry({
    type: 'breath',
    mood: 'calm',
    content: '完成 1 分钟呼吸练习',
    aiReply: '已完成一次短恢复。'
  })
  entries.value.unshift(entry)
  showToast('一分钟呼吸完成')
}

async function makePraise() {
  const data = await api.praise({ persona: '温柔前辈', content: content.value })
  reply.value = data.content
  const entry = await api.healingEntry({ type: 'praise', mood: 'warm', content: content.value, aiReply: data.content })
  entries.value.unshift(entry)
  showToast('已生成新的 AI 夸夸')
}

async function removeEntry(id) {
  await api.deleteHealingEntry(id)
  entries.value = entries.value.filter((item) => item.id !== id)
  showToast('已删除疗愈记录')
}

function entryLabel(type) {
  const labels = { breath: '呼吸', praise: 'AI夸夸', treehole: '树洞', sound: '声音' }
  return labels[type] || '记录'
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
.inner { width: 150rpx; height: 150rpx; border-radius: 999rpx; background: linear-gradient(135deg,#6f86df,#52b8cf); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 42rpx; }
.breath-btn, .praise-btn { width: 100%; margin-top: 24rpx; }
.reply { margin-top: 24rpx; padding: 28rpx; border-radius: 30rpx; background: rgba(255,255,255,.68); }
.entry-card { display: flex; flex-direction: column; gap: 18rpx; }
.source-text { margin-top: 6rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
@keyframes breathe { 0%,100% { transform: scale(.86); opacity:.9 } 50% { transform: scale(1.08); opacity:.52 } }
</style>
