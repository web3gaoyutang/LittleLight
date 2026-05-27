<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">账号 · 素材库</text>
        <view class="title">我的微光</view>
      </view>
      <button class="ghost-btn small" @tap="editProfile">编辑资料</button>
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

    <view class="material">
      <view class="row between">
        <text class="tag">素材</text>
        <button class="primary-btn small" @tap="addFavorite">添加收藏</button>
      </view>
      <text class="section-title">常用素材库</text>
      <text class="body">常用沟通模板、AI夸夸用语、班级反馈句式。</text>
    </view>

    <view class="section-head row between">
      <text class="section-title">我的收藏</text>
      <text class="caption">{{ favorites.length }} 条</text>
    </view>
    <view v-for="item in favorites" :key="item.id" class="card">
      <view class="row between">
        <text class="tag">{{ favoriteTypeText(item.type) }}</text>
        <button class="ghost-btn mini danger" @tap="removeFavorite(item.id)">删除</button>
      </view>
      <text class="section-title">{{ item.title }}</text>
      <text class="body">{{ item.content }}</text>
    </view>

    <view class="section-head row between">
      <text class="section-title">数据与权益</text>
      <button class="ghost-btn small" @tap="showBenefits">查看权益</button>
    </view>
    <view class="card" @tap="renewPro">
      <view class="row between">
        <text class="section-title">微光 Pro</text>
        <text class="tag">{{ proText }}</text>
      </view>
      <text class="body">AI 沟通、云同步、导入课表、素材库与高级提醒。</text>
    </view>
    <view class="card" @tap="toggleReminderPolicy">
      <view class="row between">
        <text class="section-title">今日提醒允许通知</text>
        <text class="tag">{{ reminderText }}</text>
      </view>
      <text class="body">点击切换普通提醒与低打扰提醒策略。</text>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'

const profile = ref({})
const favorites = ref([])

const avatarText = computed(() => (profile.value.name || '微').slice(0, 1))
const roleText = computed(() => profile.value.isHeadTeacher ? `${profile.value.stage || '学段'}班主任` : (profile.value.stage || '任课老师'))
const proText = computed(() => ({ free: '免费版', trial: 'Pro 试用', pro: 'Pro 已开通', expired: 'Pro 已过期' })[profile.value.proStatus] || '免费版')
const reminderText = computed(() => profile.value.reminderPolicy === 'low_interrupt' ? '低打扰提醒' : '普通提醒')

onShow(load)

async function load() {
  profile.value = await api.me()
  favorites.value = await api.favorites()
}

async function editProfile() {
  const next = {
    ...profile.value,
    name: profile.value.name || '林小微',
    school: profile.value.school || '微光实验小学',
    stage: profile.value.stage || '小学',
    subject: profile.value.subject || '语文',
    isHeadTeacher: !profile.value.isHeadTeacher
  }
  profile.value = await api.updateMe(next)
  showToast('已更新教师资料')
}

async function toggleReminderPolicy() {
  const nextPolicy = profile.value.reminderPolicy === 'low_interrupt' ? 'normal' : 'low_interrupt'
  profile.value = await api.updateMe({ ...profile.value, reminderPolicy: nextPolicy })
  showToast('已更新提醒偏好')
}

async function addFavorite() {
  const type = favorites.value.length % 2 === 0 ? 'communication_template' : 'ai_praise'
  const item = await api.createFavorite({
    type,
    title: type === 'communication_template' ? '先共情再同步' : '疲惫时的 AI 夸夸',
    content: type === 'communication_template'
      ? '我理解您对孩子状态的担心，我先同步今天观察到的具体表现，再一起看下一步。'
      : '你已经在很复杂的一天里稳住了很多细节，先给自己一点恢复空间。'
  })
  favorites.value.unshift(item)
  showToast('已加入收藏')
}

async function removeFavorite(id) {
  await api.deleteFavorite(id)
  favorites.value = favorites.value.filter((item) => item.id !== id)
  showToast('已删除收藏')
}

function showBenefits() {
  showToast('Pro 包含 AI 沟通、云同步、导入和高级提醒')
}

function renewPro() {
  showToast('续费页待接支付 SDK')
}

function favoriteTypeText(type) {
  return ({ communication_template: '沟通模板', ai_praise: 'AI夸夸', class_feedback: '班级反馈' })[type] || '素材'
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header, .section-head { padding: 0 32rpx 12rpx; }
.profile { display: flex; gap: 24rpx; align-items: center; }
.profile-main { flex: 1; min-width: 0; }
.avatar { width: 112rpx; height: 112rpx; border-radius: 36rpx; background: linear-gradient(145deg,#6f86df,#52b8cf); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 44rpx; font-weight: 950; }
.block { display: block; margin: 8rpx 0 16rpx; }
.tag-row { flex-wrap: wrap; }
.material { margin: 24rpx 32rpx; padding: 32rpx; border-radius: 44rpx; background: linear-gradient(145deg,rgba(255,214,231,.82),rgba(255,235,193,.74)); box-shadow: 0 24rpx 60rpx rgba(73,91,146,.10); display: flex; flex-direction: column; gap: 22rpx; }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
</style>
