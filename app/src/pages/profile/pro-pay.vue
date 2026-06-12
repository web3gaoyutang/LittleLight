<template>
  <view class="page-wrap pay-page">
    <view class="pay-hero" :class="{ success: paid }">
      <view class="pay-rings" aria-hidden="true">
        <view class="ring one"></view>
        <view class="ring two"></view>
      </view>
      <text class="pay-icon"><AppIcon :name="paid ? 'check' : 'wechat'" /></text>
      <text class="pay-title">{{ paid ? 'Pro 已开通' : '微信支付确认' }}</text>
      <text class="pay-subtitle">{{ paid ? '模拟支付成功，会员权益已经写入当前账号。' : '当前为模拟微信支付，不会产生真实扣款。' }}</text>
    </view>

    <AppState v-if="loading" type="loading" message="正在准备支付信息..." />
    <AppState v-if="error" type="error" :message="error" action-text="重试" @action="load" />

    <view class="order-card">
      <view class="row between">
        <view>
          <text class="caption">会员套餐</text>
          <text class="section-title block">{{ currentPlan.name }}</text>
        </view>
        <text class="plan-pill">{{ currentPlan.periodText }}</text>
      </view>
      <view class="amount-row">
        <text class="currency">¥</text>
        <text class="amount">{{ currentPlan.price }}</text>
      </view>
      <view class="order-lines">
        <view class="order-line row between">
          <text>支付方式</text>
          <text class="line-strong"><AppIcon name="wechat" />微信支付</text>
        </view>
        <view class="order-line row between">
          <text>支付类型</text>
          <text class="line-strong">模拟支付</text>
        </view>
        <view class="order-line row between">
          <text>会员状态</text>
          <text class="line-strong">{{ proText }}</text>
        </view>
        <view v-if="checkoutResult.orderId" class="order-line row between">
          <text>订单编号</text>
          <text class="order-id">{{ checkoutResult.orderId }}</text>
        </view>
      </view>
    </view>

    <view class="safety-card">
      <text class="safety-icon"><AppIcon name="shieldCheck" /></text>
      <view class="safety-copy">
        <text class="safety-title">模拟微信支付说明</text>
        <text class="safety-desc">点击确认后会调用服务端模拟支付接口，并把当前账号更新为 Pro。正式微信商户号接入前不会拉起真实收银台。</text>
      </view>
    </view>

    <button
      class="primary-btn pay-action"
      :disabled="paying || paid"
      data-testid="mock-wechat-pay-button"
      @tap="confirmPay"
    >
      <text class="btn-icon"><AppIcon :name="paid ? 'check' : 'wechat'" /></text>
      {{ paid ? '已开通 Pro' : (paying ? '支付中...' : `确认支付 ¥${currentPlan.price}`) }}
    </button>
    <button v-if="paid" class="ghost-btn return-btn" data-testid="pay-return-profile-button" @tap="backProfile">
      返回我的
    </button>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { ensureLoggedIn, errorMessage, showToast } from '../../utils/ui'
import AppState from '../../components/AppState.vue'
import AppIcon from '../../components/AppIcon.vue'

const planId = ref('yearly')
const profile = ref({})
const loading = ref(false)
const paying = ref(false)
const paid = ref(false)
const error = ref('')
const checkoutResult = ref({})

const plans = {
  monthly: { id: 'monthly', name: '微光 Pro 月度会员', price: 20, periodText: '月费' },
  yearly: { id: 'yearly', name: '微光 Pro 年度会员', price: 200, periodText: '年费' }
}

const currentPlan = computed(() => plans[planId.value] || plans.yearly)
const proText = computed(() => ({ free: '免费版', trial: 'Pro 试用', pro: 'Pro 已开通', expired: 'Pro 已过期' })[profile.value.proStatus] || '免费版')

onLoad((query) => {
  if (query?.plan === 'monthly' || query?.plan === 'yearly') {
    planId.value = query.plan
  }
})

onShow(load)

async function load() {
  if (!ensureLoggedIn(api)) return
  loading.value = true
  error.value = ''
  try {
    profile.value = await api.me()
  } catch (err) {
    error.value = errorMessage(err, '支付信息加载失败')
  } finally {
    loading.value = false
  }
}

async function confirmPay() {
  if (paying.value || paid.value) return
  paying.value = true
  error.value = ''
  try {
    const result = await api.checkout({
      plan: currentPlan.value.id,
      provider: 'wechat',
      mock: true
    })
    checkoutResult.value = result || {}
    profile.value = result.profile || profile.value
    paid.value = result.status === 'paid'
    showToast(result.message || '模拟微信支付成功')
  } catch (err) {
    error.value = errorMessage(err, '模拟微信支付失败')
    showToast(error.value)
  } finally {
    paying.value = false
  }
}

function backProfile() {
  uni.switchTab({ url: '/pages/profile/index' })
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.pay-page { padding-bottom: calc(72rpx + env(safe-area-inset-bottom)); }
.pay-hero {
  margin: 8rpx 0 24rpx;
  min-height: 380rpx;
  padding: 40rpx 32rpx;
  border-radius: 36rpx;
  box-sizing: border-box;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18rpx;
  color: #fff;
  text-align: center;
  background:
    radial-gradient(78% 92% at 86% 6%, rgba(255, 215, 112, .42), transparent 56%),
    linear-gradient(135deg, #153b34 0%, #22b96f 52%, #31d987 100%);
  border: 1rpx solid rgba(255,255,255,.74);
  box-shadow: 0 24rpx 54rpx rgba(34, 182, 109, .2), inset 0 1rpx 0 rgba(255,255,255,.28);
}
.pay-hero.success {
  background:
    radial-gradient(78% 92% at 86% 6%, rgba(255, 225, 127, .48), transparent 56%),
    linear-gradient(135deg, #18244f 0%, #5364dc 52%, #22bd8a 100%);
}
.pay-rings { position: absolute; inset: 0; pointer-events: none; }
.ring { position: absolute; border-radius: 50%; border: 2rpx solid rgba(255,255,255,.16); animation: ringPulse 3.8s ease-in-out infinite; }
.ring.one { width: 260rpx; height: 260rpx; left: -80rpx; top: -70rpx; }
.ring.two { width: 180rpx; height: 180rpx; right: -40rpx; bottom: -40rpx; animation-delay: -1.4s; }
.pay-icon {
  position: relative;
  z-index: 1;
  width: 96rpx;
  height: 96rpx;
  border-radius: 32rpx;
  color: #17b86b;
  background: rgba(255,255,255,.96);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48rpx;
  box-shadow: 0 16rpx 34rpx rgba(18, 86, 57, .18), inset 0 1rpx 0 rgba(255,255,255,.92);
}
.pay-title { position: relative; z-index: 1; color: #fff; font-size: 46rpx; line-height: 1.12; font-weight: 950; letter-spacing: 0; }
.pay-subtitle { position: relative; z-index: 1; max-width: 560rpx; color: rgba(255,255,255,.86); font-size: 25rpx; line-height: 1.5; font-weight: 820; }
.order-card {
  margin: 20rpx 0;
  padding: 30rpx;
  border-radius: 30rpx;
  background: rgba(255,255,255,.72);
  border: 1rpx solid rgba(255,255,255,.82);
  box-shadow: 0 18rpx 44rpx rgba(73,91,146,.1), inset 0 1rpx 0 rgba(255,255,255,.9);
}
.plan-pill { padding: 8rpx 16rpx; border-radius: 999rpx; color: #573f13; background: linear-gradient(135deg,#fff0b8,#ffd066); font-size: 22rpx; font-weight: 950; line-height: 1.2; }
.amount-row { margin: 30rpx 0; display: flex; align-items: flex-end; justify-content: center; color: #172039; }
.currency { font-size: 36rpx; font-weight: 950; line-height: 1.2; padding-bottom: 12rpx; }
.amount { font-size: 96rpx; line-height: .95; font-weight: 950; letter-spacing: 0; }
.order-lines { display: flex; flex-direction: column; gap: 2rpx; border-top: 1rpx solid rgba(97,116,166,.12); padding-top: 16rpx; }
.order-line { min-height: 62rpx; color: #66738d; font-size: 24rpx; line-height: 1.2; font-weight: 820; }
.line-strong { display: inline-flex; align-items: center; gap: 8rpx; color: #172039; font-weight: 930; }
.line-strong .app-icon { width: 26rpx; height: 26rpx; color: #20b86d; }
.order-id { max-width: 360rpx; color: #172039; font-size: 21rpx; line-height: 1.25; font-weight: 900; text-align: right; overflow-wrap: anywhere; }
.safety-card {
  margin: 20rpx 0 28rpx;
  padding: 24rpx;
  border-radius: 26rpx;
  background: rgba(255,255,255,.62);
  border: 1rpx solid rgba(255,255,255,.78);
  display: flex;
  gap: 16rpx;
  align-items: flex-start;
  box-shadow: 0 10rpx 24rpx rgba(73,91,146,.07), inset 0 1rpx 0 rgba(255,255,255,.88);
}
.safety-icon { flex: 0 0 auto; width: 50rpx; height: 50rpx; border-radius: 18rpx; color: #22a970; background: rgba(232,255,242,.92); display: flex; align-items: center; justify-content: center; font-size: 28rpx; }
.safety-copy { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 8rpx; }
.safety-title { color: #172039; font-size: 25rpx; line-height: 1.2; font-weight: 950; }
.safety-desc { color: #5f6a84; font-size: 22rpx; line-height: 1.48; font-weight: 780; }
.pay-action { width: 100%; min-height: 96rpx; background: linear-gradient(135deg, #39e58f 0%, #28cb75 100%); box-shadow: 0 16rpx 30rpx rgba(35, 190, 112, .22), inset 0 1rpx 0 rgba(255,255,255,.38); }
.pay-action[disabled] { opacity: .72; }
.return-btn { width: 100%; margin-top: 18rpx; }
@keyframes ringPulse {
  0%, 100% { transform: scale(.96); opacity: .62; }
  50% { transform: scale(1.06); opacity: 1; }
}
@media (prefers-reduced-motion: reduce) {
  .ring { animation: none; }
}
</style>
