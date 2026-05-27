<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">周三 · 第 12 周</text>
        <view class="title">早上好，林老师</view>
      </view>
      <button class="bell" @tap="goSchedule">⌁</button>
    </view>

    <view class="hero card">
      <text class="tag">今日教学心流</text>
      <view class="hero-title">{{ summary.rhythm?.title || '温柔但高效' }}</view>
      <text class="body">{{ summary.rhythm?.description || '先把节奏拆小，保持一点余裕。' }}</text>
      <view class="row actions">
        <button class="primary-btn action" @tap="goCommunication">AI 助手</button>
        <button class="ghost-btn action" @tap="goSchedule">今日安排</button>
      </view>
    </view>

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
      <view class="tile blue" @tap="goSchedule"><text class="tag">课程</text><text class="num">{{ summary.coursesCount || 0 }}</text><text class="caption">今日课程</text></view>
      <view class="tile mint" @tap="goSchedule"><text class="tag">待办</text><text class="num">{{ summary.remindersCount || 0 }}</text><text class="caption">待处理提醒</text></view>
      <view class="tile pink" @tap="goCommunication"><text class="tag">AI</text><text class="tile-title">生成家校回复</text><text class="caption">多语气版本</text></view>
      <view class="tile peach" @tap="goHeal"><text class="tag">恢复</text><text class="tile-title">一分钟呼吸</text><text class="caption">先收回注意力</text></view>
    </view>

    <view class="card" v-if="summary.nextCourse">
      <view class="row between"><text class="section-title">下一项</text><text class="tag">{{ summary.nextCourse.startTime }}</text></view>
      <view class="course-title">{{ summary.nextCourse.className }} · {{ summary.nextCourse.title }}</view>
      <text class="body">{{ summary.nextCourse.note }} · {{ summary.nextCourse.location }}</text>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'

const summary = ref({ rhythm: {} })
const mood = ref('steady')
const moods = [
  { code: 'steady', icon: '🌤', label: '平稳' },
  { code: 'busy', icon: '⚡', label: '偏满' },
  { code: 'soft', icon: '🌙', label: '缓冲' },
  { code: 'focus', icon: '🎯', label: '专注' },
  { code: 'warm', icon: '💬', label: '沟通' }
]

onShow(async () => {
  try { summary.value = await api.dashboard() } catch (error) { console.warn(error) }
})
function goSchedule() { uni.switchTab({ url: '/pages/schedule/index' }) }
function goHeal() { uni.switchTab({ url: '/pages/heal/index' }) }
function goCommunication() { uni.switchTab({ url: '/pages/communication/index' }) }
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header { padding: 0 32rpx 12rpx; }
.bell { width: 88rpx; height: 88rpx; border-radius: 34rpx; background: rgba(255,255,255,.86); color: #435376; }
.hero { background: linear-gradient(134deg,rgba(235,246,255,.96),rgba(241,235,255,.93) 36%,rgba(255,241,236,.90) 70%,rgba(242,252,246,.94)); }
.hero-title { margin: 20rpx 0 12rpx; font-size: 56rpx; font-weight: 950; color: #172039; }
.actions { margin-top: 28rpx; }
.action { flex: 1; }
.section-head { padding: 16rpx 32rpx 0; }
.mood-row { display: flex; gap: 16rpx; padding: 20rpx 32rpx; overflow-x: auto; }
.mood-chip { min-width: 112rpx; height: 112rpx; border-radius: 34rpx; background: rgba(255,255,255,.74); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8rpx; font-size: 22rpx; font-weight: 800; color: #3e415a; }
.mood-chip.active { outline: 4rpx solid rgba(98,166,233,.32); }
.mood-icon { font-size: 32rpx; }
.bento { display: grid; grid-template-columns: 1fr 1fr; gap: 24rpx; padding: 0 32rpx; }
.tile { min-height: 190rpx; border-radius: 42rpx; padding: 28rpx; display: flex; flex-direction: column; justify-content: space-between; box-shadow: 0 20rpx 44rpx rgba(73,91,146,.10); }
.tile.blue { background: linear-gradient(145deg,rgba(209,231,255,.90),rgba(222,216,255,.74)); }
.tile.mint { background: linear-gradient(145deg,rgba(204,241,232,.86),rgba(219,239,255,.72)); }
.tile.pink { background: linear-gradient(145deg,rgba(255,214,231,.82),rgba(255,235,193,.74)); }
.tile.peach { background: linear-gradient(145deg,rgba(255,241,191,.82),rgba(255,219,207,.72)); }
.num { font-size: 56rpx; font-weight: 950; color: #182033; }
.tile-title, .course-title { font-size: 32rpx; font-weight: 900; color: #182033; }
</style>
