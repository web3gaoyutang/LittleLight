<template>
  <view class="page-wrap pro-page">
    <view class="hero-card" :class="{ active: isProMember }">
      <view class="hero-glow" aria-hidden="true"></view>
      <view class="hero-top row between">
        <view class="hero-title-block">
          <text class="hero-kicker">{{ isProMember ? '专属会员' : '微光 Pro' }}</text>
          <text class="hero-title">{{ isProMember ? 'Pro 权益已生效' : '让教研工作更省心' }}</text>
        </view>
        <text class="hero-icon"><AppIcon name="crown" /></text>
      </view>
      <text class="hero-copy">{{ isProMember ? '你的账号已经开通微光 Pro，AI 助手、课表同步和资料能力已解锁。' : '开通后解锁 AI 助手、课表同步、家校沟通素材库和账号数据能力。' }}</text>
      <view class="hero-status row between">
        <text class="status-pill">{{ proText }}</text>
        <text class="mock-pill">模拟微信支付</text>
      </view>
    </view>

    <AppState v-if="loading" type="loading" message="正在读取会员权益..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="plan-section">
      <view
        v-for="plan in plans"
        :key="plan.id"
        class="plan-card"
        :class="{ selected: selectedPlan === plan.id, popular: plan.id === 'yearly' }"
        :data-testid="`pro-plan-${plan.id}`"
        @tap="selectedPlan = plan.id"
      >
        <view class="plan-top row between">
          <view class="plan-name-wrap">
            <text class="plan-name">{{ plan.name }}</text>
            <text class="plan-note">{{ plan.note }}</text>
          </view>
          <text v-if="plan.id === 'yearly'" class="save-tag">省 40 元</text>
        </view>
        <view class="price-row">
          <text class="currency">¥</text>
          <text class="price">{{ plan.price }}</text>
          <text class="period">/{{ plan.period }}</text>
        </view>
        <view class="select-row">
          <text class="select-dot"><AppIcon v-if="selectedPlan === plan.id" name="check" /></text>
          <text>{{ selectedPlan === plan.id ? '已选择' : '点击选择' }}</text>
        </view>
      </view>
    </view>

    <view class="benefits-grid">
      <view v-for="item in benefits" :key="item.title" class="benefit-item">
        <text class="benefit-icon"><AppIcon :name="item.icon" /></text>
        <view class="benefit-copy">
          <text class="benefit-title">{{ item.title }}</text>
          <text class="benefit-desc">{{ item.desc }}</text>
        </view>
      </view>
    </view>

    <view class="notice-card">
      <text class="notice-icon"><AppIcon name="info" /></text>
      <text class="notice-copy">当前为模拟微信支付流程，用于验证会员购买页面和状态流转，不会产生真实扣款。</text>
    </view>

    <view class="bottom-bar">
      <view class="pay-summary">
        <text class="summary-label">{{ currentPlan.name }}</text>
        <text class="summary-price">¥{{ currentPlan.price }}/{{ currentPlan.period }}</text>
      </view>
      <button class="primary-btn pay-btn" data-testid="pro-open-pay-button" @tap="goPay">
        <text class="btn-icon"><AppIcon name="wechat" /></text>
        {{ isProMember ? '继续续费' : '微信支付开通' }}
      </button>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { ensureLoggedIn, errorMessage, showToast } from '../../utils/ui'
import AppState from '../../components/AppState.vue'
import AppIcon from '../../components/AppIcon.vue'

const profile = ref({})
const entitlements = ref({})
const loading = ref(false)
const error = ref('')
const selectedPlan = ref('yearly')

const plans = [
  { id: 'monthly', name: '月度会员', price: 20, period: '月', note: '轻量体验，随时续费' },
  { id: 'yearly', name: '年度会员', price: 200, period: '年', note: '长期使用更划算' }
]

const benefits = [
  { icon: 'sparkles', title: 'AI 助手', desc: '高频生成沟通草稿、夸夸语和复盘记录。' },
  { icon: 'calendarCheck', title: '课表同步', desc: '课程、待办和提醒按账号持续同步。' },
  { icon: 'message', title: '家校沟通', desc: '沉淀家长档案、跟进记录和常用话术。' },
  { icon: 'database', title: '数据与导出', desc: '账号数据导出、素材库和 AI 审计记录。' },
  { icon: 'shieldCheck', title: '账号安全', desc: '微信登录隔离账号数据，敏感操作可追踪。' },
  { icon: 'bell', title: '高级提醒', desc: '低打扰策略和待办提醒为教师节奏服务。' }
]

const currentPlan = computed(() => plans.find((item) => item.id === selectedPlan.value) || plans[1])
const isProMember = computed(() => profile.value.proStatus === 'pro' || profile.value.proStatus === 'trial' || entitlements.value.plan === 'pro')
const proText = computed(() => ({ free: '免费版', trial: 'Pro 试用', pro: 'Pro 已开通', expired: 'Pro 已过期' })[profile.value.proStatus] || '免费版')

onShow(load)

async function load() {
  if (!ensureLoggedIn(api)) return
  loading.value = true
  error.value = ''
  try {
    const [nextProfile, nextEntitlements] = await Promise.all([api.me(), api.entitlements()])
    profile.value = nextProfile
    entitlements.value = nextEntitlements
  } catch (err) {
    error.value = errorMessage(err, '会员权益加载失败')
  } finally {
    loading.value = false
  }
}

function goPay() {
  const plan = currentPlan.value
  if (!plan?.id) {
    showToast('请选择会员套餐')
    return
  }
  uni.navigateTo({ url: `/pages/profile/pro-pay?plan=${encodeURIComponent(plan.id)}` })
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.pro-page { padding-bottom: calc(174rpx + env(safe-area-inset-bottom)); }
.hero-card {
  margin: 8rpx 0 24rpx;
  padding: 34rpx;
  border-radius: 34rpx;
  position: relative;
  overflow: hidden;
  color: #fff;
  background:
    radial-gradient(82% 94% at 92% 0%, rgba(255, 215, 112, .46), transparent 56%),
    linear-gradient(135deg, #202a55 0%, #5364df 52%, #22bd8b 100%);
  border: 1rpx solid rgba(255,255,255,.74);
  box-shadow: 0 22rpx 54rpx rgba(55, 72, 150, .2), inset 0 1rpx 0 rgba(255,255,255,.24);
}
.hero-card.active {
  background:
    radial-gradient(82% 94% at 92% 0%, rgba(255, 225, 127, .5), transparent 56%),
    linear-gradient(135deg, #172040 0%, #4d5fd8 50%, #1fbf86 100%);
}
.hero-glow {
  position: absolute;
  right: -64rpx;
  bottom: -68rpx;
  width: 230rpx;
  height: 230rpx;
  border-radius: 50%;
  background: rgba(255,255,255,.13);
}
.hero-top, .hero-copy, .hero-status { position: relative; z-index: 1; }
.hero-title-block { min-width: 0; display: flex; flex-direction: column; gap: 10rpx; }
.hero-kicker { color: rgba(255,255,255,.72); font-size: 23rpx; font-weight: 900; line-height: 1.2; }
.hero-title { color: #fff; font-size: 46rpx; font-weight: 950; line-height: 1.12; letter-spacing: 0; }
.hero-copy { display: block; margin-top: 22rpx; color: rgba(255,255,255,.86); font-size: 26rpx; line-height: 1.55; font-weight: 780; }
.hero-icon { width: 78rpx; height: 78rpx; border-radius: 26rpx; color: #543b0b; background: linear-gradient(135deg,#fff0b8,#ffc552); display: flex; align-items: center; justify-content: center; font-size: 38rpx; box-shadow: 0 14rpx 24rpx rgba(23, 31, 67, .18), inset 0 1rpx 0 rgba(255,255,255,.78); }
.hero-status { margin-top: 26rpx; gap: 14rpx; flex-wrap: wrap; }
.status-pill, .mock-pill {
  min-height: 46rpx;
  padding: 8rpx 16rpx;
  border-radius: 999rpx;
  font-size: 22rpx;
  line-height: 1.2;
  font-weight: 930;
  border: 1rpx solid rgba(255,255,255,.28);
}
.status-pill { color: #4d3509; background: linear-gradient(135deg,#fff0b6,#ffd066); }
.mock-pill { color: #eef7ff; background: rgba(255,255,255,.16); }
.plan-section { display: grid; grid-template-columns: 1fr 1fr; gap: 18rpx; margin: 20rpx 0; }
.plan-card {
  min-height: 246rpx;
  padding: 24rpx;
  border-radius: 28rpx;
  background: rgba(255,255,255,.66);
  border: 2rpx solid rgba(255,255,255,.78);
  box-shadow: 0 12rpx 26rpx rgba(73,91,146,.08), inset 0 1rpx 0 rgba(255,255,255,.9);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 18rpx;
}
.plan-card.selected {
  border-color: rgba(255, 197, 82, .9);
  background:
    radial-gradient(90% 80% at 100% 0%, rgba(255, 218, 126, .28), transparent 52%),
    rgba(255,255,255,.82);
  box-shadow: 0 16rpx 34rpx rgba(124, 95, 38, .12), inset 0 1rpx 0 rgba(255,255,255,.95);
}
.plan-card.popular { position: relative; }
.plan-top { align-items: flex-start; gap: 10rpx; }
.plan-name-wrap { min-width: 0; display: flex; flex-direction: column; gap: 8rpx; }
.plan-name { color: #172039; font-size: 29rpx; line-height: 1.2; font-weight: 950; }
.plan-note { color: #66738d; font-size: 21rpx; line-height: 1.35; font-weight: 820; }
.save-tag { flex: 0 0 auto; padding: 6rpx 12rpx; border-radius: 999rpx; color: #6d4503; background: #ffe49d; font-size: 19rpx; line-height: 1.2; font-weight: 950; }
.price-row { display: flex; align-items: flex-end; gap: 4rpx; color: #172039; }
.currency { font-size: 25rpx; line-height: 1.2; font-weight: 950; padding-bottom: 6rpx; }
.price { font-size: 56rpx; line-height: .98; font-weight: 950; letter-spacing: 0; }
.period { color: #7a849c; font-size: 22rpx; line-height: 1.25; font-weight: 900; padding-bottom: 6rpx; }
.select-row { margin-top: auto; display: flex; align-items: center; gap: 10rpx; color: #536079; font-size: 22rpx; font-weight: 900; }
.select-dot { width: 32rpx; height: 32rpx; border-radius: 50%; display: flex; align-items: center; justify-content: center; color: #fff; background: rgba(255,255,255,.9); border: 2rpx solid rgba(91,112,138,.32); font-size: 19rpx; }
.plan-card.selected .select-dot { border-color: #22c47b; background: #22c47b; }
.benefits-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16rpx; margin: 24rpx 0; }
.benefit-item {
  min-height: 184rpx;
  padding: 22rpx;
  border-radius: 26rpx;
  background: rgba(255,255,255,.62);
  border: 1rpx solid rgba(255,255,255,.78);
  box-shadow: 0 10rpx 22rpx rgba(73,91,146,.07), inset 0 1rpx 0 rgba(255,255,255,.9);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 14rpx;
}
.benefit-icon { width: 48rpx; height: 48rpx; border-radius: 17rpx; color: #5263d8; background: rgba(241,243,255,.94); display: flex; align-items: center; justify-content: center; font-size: 27rpx; }
.benefit-copy { min-width: 0; display: flex; flex-direction: column; gap: 8rpx; }
.benefit-title { color: #172039; font-size: 25rpx; font-weight: 950; line-height: 1.2; }
.benefit-desc { color: #5c6680; font-size: 22rpx; line-height: 1.42; font-weight: 760; }
.notice-card {
  margin: 20rpx 0;
  padding: 22rpx;
  border-radius: 24rpx;
  color: #526079;
  background: rgba(255,255,255,.6);
  border: 1rpx solid rgba(255,255,255,.78);
  display: flex;
  gap: 14rpx;
  align-items: flex-start;
}
.notice-icon { flex: 0 0 auto; width: 42rpx; height: 42rpx; border-radius: 15rpx; color: #4f61d8; background: rgba(241,243,255,.9); display: flex; align-items: center; justify-content: center; font-size: 24rpx; }
.notice-copy { flex: 1; min-width: 0; font-size: 22rpx; line-height: 1.5; font-weight: 820; }
.bottom-bar {
  position: fixed;
  left: 50%;
  bottom: 0;
  z-index: 10;
  width: 100%;
  max-width: 780rpx;
  transform: translateX(-50%);
  padding: 18rpx 32rpx calc(18rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
  display: flex;
  align-items: center;
  gap: 18rpx;
  background: rgba(248,250,255,.82);
  backdrop-filter: blur(22rpx) saturate(1.1);
  border-top: 1rpx solid rgba(255,255,255,.76);
}
.pay-summary { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4rpx; }
.summary-label { color: #66738d; font-size: 21rpx; line-height: 1.2; font-weight: 850; }
.summary-price { color: #172039; font-size: 30rpx; line-height: 1.2; font-weight: 950; }
.pay-btn { flex: 0 0 auto; min-width: 284rpx; background: linear-gradient(135deg, #39e58f 0%, #28cb75 100%); box-shadow: 0 14rpx 26rpx rgba(35, 190, 112, .22), inset 0 1rpx 0 rgba(255,255,255,.38); }
@media (max-width: 380px) {
  .plan-section, .benefits-grid { grid-template-columns: 1fr; }
  .bottom-bar { align-items: stretch; flex-direction: column; }
  .pay-btn { width: 100%; min-width: 0; }
}
</style>
