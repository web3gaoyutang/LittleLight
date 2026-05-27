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
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'

const breathing = ref(false)
const left = ref(60)
let timer = null
const content = ref('')
const reply = ref('你今天已经完成了很多复杂判断。先把肩膀放下来，剩下的事情可以一件一件处理。')
const leftText = computed(() => `00:${String(left.value).padStart(2, '0')}`)

function toggleBreath() {
  if (breathing.value) {
    clearInterval(timer); breathing.value = false; left.value = 60; return
  }
  breathing.value = true; left.value = 60
  timer = setInterval(() => {
    left.value -= 1
    if (left.value <= 0) {
      clearInterval(timer); breathing.value = false; left.value = 60; showToast('一分钟呼吸完成')
    }
  }, 1000)
}
async function makePraise() {
  const data = await api.praise({ persona: '温柔前辈', content: content.value })
  reply.value = data.content
  await api.healingEntry({ type: 'praise', mood: 'warm', content: content.value, aiReply: data.content })
  showToast('已生成新的 AI 夸夸')
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header { padding: 0 32rpx 12rpx; }
.block { display: block; }
.breath { min-height: 720rpx; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 28rpx; }
.breath-core { width: 300rpx; height: 300rpx; border-radius: 999rpx; display: flex; align-items: center; justify-content: center; background: rgba(121,144,255,.14); }
.breath-core.active { animation: breathe 7.2s ease-in-out infinite; }
.inner { width: 150rpx; height: 150rpx; border-radius: 999rpx; background: linear-gradient(135deg,#6f86df,#52b8cf); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 42rpx; }
.breath-btn, .praise-btn { width: 100%; margin-top: 24rpx; }
.reply { margin-top: 24rpx; padding: 28rpx; border-radius: 30rpx; background: rgba(255,255,255,.68); }
@keyframes breathe { 0%,100% { transform: scale(.86); opacity:.9 } 50% { transform: scale(1.08); opacity:.52 } }
</style>
