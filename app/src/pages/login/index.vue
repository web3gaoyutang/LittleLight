<template>
  <view class="login-page">
    <view class="login-hero">
      <view class="app-mark" aria-hidden="true">
        <view class="app-mark-core"><AppIcon name="sparkles" /></view>
      </view>
      <view class="hero-copy">
        <text class="login-title">欢迎使用 林小微</text>
        <text class="login-subtitle">智能高效的教研助手</text>
      </view>

      <view class="illustration-wrap">
        <view class="illustration-glow"></view>
        <view class="wave-lines" aria-hidden="true">
          <view class="wave-line line-one"></view>
          <view class="wave-line line-two"></view>
          <view class="wave-dot"></view>
        </view>
        <image
          class="mascot-image"
          src="/static/login-mascot.png"
          mode="aspectFit"
          aria-label="林小微智能教研助手插画"
        />
        <view class="feature-chip chip-schedule">
          <text class="chip-icon"><AppIcon name="calendarCheck" /></text>
          <text>课表同步</text>
        </view>
        <view class="feature-chip chip-ai">
          <text class="chip-icon"><AppIcon name="sparkles" /></text>
          <text>AI 助手</text>
        </view>
        <view class="feature-chip chip-chat">
          <text class="chip-icon"><AppIcon name="message" /></text>
          <text>家校沟通</text>
        </view>
        <view class="feature-chip chip-reminder">
          <text class="chip-icon"><AppIcon name="bell" /></text>
          <text>待办提醒</text>
        </view>
        <view class="feature-chip chip-heal">
          <text class="chip-icon"><AppIcon name="heart" /></text>
          <text>情绪疗愈</text>
        </view>
        <view class="feature-chip chip-safe">
          <text class="chip-icon"><AppIcon name="shieldCheck" /></text>
          <text>账号安全</text>
        </view>
      </view>
    </view>

    <view class="login-actions">
      <view class="sr-only-field" aria-hidden="true">
        <input class="input" v-model="nickName" aria-label="登录昵称" data-testid="login-nickname-input" />
      </view>
      <button
        class="wechat-btn"
        :class="{ disabled: !canUsePrimaryLogin }"
        :disabled="loading || !canUsePrimaryLogin"
        :aria-busy="loading ? 'true' : 'false'"
        aria-label="微信快捷登录"
        data-testid="wechat-login-button"
        @tap="login"
      >
        <text class="wechat-mark"><AppIcon name="wechat" /></text>
        <text>{{ primaryButtonText }}</text>
      </button>
      <view class="agreement-row">
        <view
          class="agreement-check"
          :class="{ checked: agreementAccepted }"
          role="checkbox"
          :aria-checked="agreementAccepted ? 'true' : 'false'"
          aria-label="同意用户协议和隐私政策"
          @tap="toggleAgreement"
        >
          <text v-if="agreementAccepted" class="agreement-tick"><AppIcon name="check" /></text>
        </view>
        <view class="agreement-copy">
          <text>同意</text>
          <text class="agreement-link" @tap.stop="openAgreement('user')">《用户协议》</text>
          <text>及</text>
          <text class="agreement-link" @tap.stop="openAgreement('privacy')">《隐私政策》</text>
        </view>
      </view>
      <text v-if="errorText" class="error-text">{{ errorText }}</text>
    </view>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { errorMessage, showToast } from '../../utils/ui'
import AppIcon from '../../components/AppIcon.vue'

const nickName = ref('林小微')
const loading = ref(false)
const errorText = ref('')
const agreementAccepted = ref(true)
const showDevLogin = api.isDevAuthAvailable()
const canUseWechatLogin = canRequestWechatLogin()
const canUsePrimaryLogin = computed(() => canUseWechatLogin || showDevLogin)
const primaryButtonText = computed(() => primaryLoginText())

onShow(() => {
  api.resetAuthRedirect()
  if (api.isLoggedIn()) {
    uni.switchTab({ url: '/pages/home/index' })
  }
})

async function login() {
  if (loading.value) return
  if (!canUseWechatLogin && showDevLogin) {
    await devLogin()
    return
  }
  if (!canUseWechatLogin) {
    errorText.value = '请在 App 中使用微信快捷登录。'
    return
  }
  loading.value = true
  errorText.value = ''
  try {
    await loginWithWechat()
    showToast('登录成功')
    uni.switchTab({ url: '/pages/home/index' })
  } catch (error) {
    const message = errorMessage(error, '微信登录失败')
    errorText.value = message
    showToast(message)
  } finally {
    loading.value = false
  }
}

async function loginWithWechat() {
  const code = await wxLoginCode()
  const profile = await userProfile()
  return api.wechatLogin({
    code,
    nickName: profile.nickName || nickName.value || '微光老师',
    avatarUrl: profile.avatarUrl || ''
  })
}

async function devLogin() {
  if (loading.value || !showDevLogin) return
  loading.value = true
  errorText.value = ''
  try {
    await loginWithMock()
    showToast('登录成功')
    uni.switchTab({ url: '/pages/home/index' })
  } catch (err) {
    const message = errorMessage(err, '登录失败')
    errorText.value = message
    showToast(message)
  } finally {
    loading.value = false
  }
}

function loginWithMock() {
  return api.wechatMockLogin({
    code: `app-${Date.now()}`,
    nickName: nickName.value || '微光老师'
  })
}

function wxLoginCode() {
  return new Promise((resolve, reject) => {
    if (!uni?.login) {
      reject(new Error('微信登录能力不可用'))
      return
    }
    const loginOptions = { provider: 'weixin' }
    // #ifdef APP-PLUS
    loginOptions.onlyAuthorize = true
    // #endif
    uni.login({
      ...loginOptions,
      success: (res) => {
        if (res?.code) resolve(res.code)
        else reject(new Error('微信登录未返回 code'))
      },
      fail: reject
    })
  })
}

function userProfile() {
  return new Promise((resolve) => {
    if (!canUseWechatLogin || typeof uni.getUserProfile !== 'function') {
      resolve({})
      return
    }
    uni.getUserProfile({
      desc: '用于展示教师昵称并完成账号资料初始化',
      success: (res) => {
        const userInfo = res?.userInfo || {}
        resolve({
          nickName: userInfo.nickName || '',
          avatarUrl: userInfo.avatarUrl || ''
        })
      },
      fail: () => resolve({})
    })
  })
}

function canRequestWechatLogin() {
  // #ifdef APP-PLUS
  return true
  // #endif
  return false
}

function primaryLoginText() {
  if (loading.value) return '登录中...'
  return '微信快捷登录'
}

function toggleAgreement() {
  agreementAccepted.value = !agreementAccepted.value
}

function openAgreement(type) {
  showToast(type === 'privacy' ? '隐私政策待配置' : '用户协议待配置')
}

</script>

<style src="../../static/common.css"></style>
<style scoped>
.login-page {
  min-height: 100vh;
  padding: calc(72rpx + env(safe-area-inset-top)) 44rpx calc(24rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  gap: 16rpx;
  background:
    radial-gradient(120% 72% at 10% 0%, rgba(157, 224, 242, .48), transparent 54%),
    radial-gradient(88% 66% at 100% 8%, rgba(205, 190, 255, .44), transparent 52%),
    linear-gradient(150deg, #dff5ff 0%, #eef4ff 32%, #f5edff 66%, #fff6e8 100%);
}

.login-hero {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20rpx;
  transform: translateY(14rpx);
}

.app-mark {
  width: 144rpx;
  height: 144rpx;
  border-radius: 44rpx;
  padding: 18rpx;
  box-sizing: border-box;
  background: linear-gradient(145deg, rgba(255,255,255,.92), rgba(245,250,255,.64));
  border: 1rpx solid rgba(255,255,255,.9);
  box-shadow: 0 22rpx 46rpx rgba(69, 87, 150, .14), inset 0 1rpx 0 rgba(255,255,255,.95);
}

.app-mark-core {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 54rpx;
  background: linear-gradient(145deg, #425fb3 0%, #5f7ada 100%);
  box-shadow: inset 0 -8rpx 18rpx rgba(23, 32, 57, .14);
}

.hero-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10rpx;
  text-align: center;
}

.login-title {
  display: block;
  max-width: 100%;
  color: #172039;
  font-size: 54rpx;
  line-height: 1.14;
  font-weight: 950;
  letter-spacing: 0;
  overflow-wrap: anywhere;
}

.login-subtitle {
  display: block;
  color: #66738d;
  font-size: 28rpx;
  line-height: 1.35;
  font-weight: 850;
  letter-spacing: 0;
}

.illustration-wrap {
  width: 100%;
  height: 572rpx;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 8rpx;
}

.illustration-wrap::before {
  content: "";
  position: absolute;
  left: 28rpx;
  right: 28rpx;
  top: 54rpx;
  bottom: 38rpx;
  border-radius: 38rpx;
  background:
    linear-gradient(180deg, rgba(255,255,255,.86), rgba(255,255,255,.34)),
    linear-gradient(135deg, rgba(95, 122, 218, .12), rgba(99, 200, 141, .12));
  border: 1rpx solid rgba(255,255,255,.8);
  box-shadow: 0 30rpx 70rpx rgba(72, 88, 145, .12), inset 0 1rpx 0 rgba(255,255,255,.88);
}

.illustration-glow {
  position: absolute;
  width: 420rpx;
  height: 48rpx;
  bottom: 42rpx;
  border-radius: 50%;
  background: rgba(75, 86, 142, .12);
  filter: blur(14rpx);
  animation: glowPulse 4.6s ease-in-out infinite;
}

.wave-lines {
  position: absolute;
  z-index: 3;
  left: 108rpx;
  top: 240rpx;
  width: 156rpx;
  height: 138rpx;
  pointer-events: none;
  animation: waveFloat 2.9s ease-in-out infinite;
}

.wave-line {
  position: absolute;
  border: 4rpx solid rgba(95, 122, 218, .5);
  border-left-color: transparent;
  border-bottom-color: transparent;
  border-radius: 999rpx;
  transform: rotate(-24deg);
}

.line-one {
  width: 74rpx;
  height: 74rpx;
  left: 38rpx;
  top: 18rpx;
}

.line-two {
  width: 106rpx;
  height: 106rpx;
  left: 12rpx;
  top: 4rpx;
  opacity: .58;
  animation: waveFade 2.9s ease-in-out infinite;
}

.wave-dot {
  position: absolute;
  right: 18rpx;
  top: 12rpx;
  width: 15rpx;
  height: 15rpx;
  border-radius: 50%;
  background: #f8c971;
  box-shadow: 0 0 0 8rpx rgba(248, 201, 113, .18);
  animation: dotPop 2.9s ease-in-out infinite;
}

.mascot-image {
  position: relative;
  z-index: 1;
  width: 626rpx;
  height: 548rpx;
  margin-top: 28rpx;
  filter: drop-shadow(0 20rpx 28rpx rgba(51, 66, 118, .16));
  transform-origin: 42% 74%;
  animation: mascotWave 3.2s ease-in-out infinite;
}

.feature-chip {
  position: absolute;
  z-index: 2;
  min-height: 52rpx;
  max-width: 216rpx;
  padding: 10rpx 16rpx;
  box-sizing: border-box;
  border-radius: 999rpx;
  display: flex;
  align-items: center;
  gap: 8rpx;
  color: #26304f;
  font-size: 21rpx;
  font-weight: 900;
  line-height: 1.15;
  background: rgba(255,255,255,.82);
  border: 1rpx solid rgba(255,255,255,.92);
  box-shadow: 0 12rpx 24rpx rgba(70, 88, 148, .12), inset 0 1rpx 0 rgba(255,255,255,.95);
  backdrop-filter: blur(18rpx) saturate(1.1);
  animation: chipFloat 4.8s ease-in-out infinite;
  overflow: hidden;
}

.chip-schedule {
  left: 24rpx;
  top: 104rpx;
  color: #26304f;
  animation-delay: -.2s;
}

.chip-ai {
  right: 28rpx;
  top: 146rpx;
  color: #2f315e;
  animation-delay: -1.3s;
}

.chip-chat {
  left: 8rpx;
  top: 292rpx;
  color: #25445b;
  animation-delay: -2.1s;
}

.chip-reminder {
  right: 4rpx;
  top: 310rpx;
  color: #4e3656;
  animation-delay: -.8s;
}

.chip-heal {
  left: 54rpx;
  bottom: 58rpx;
  color: #4a3850;
  animation-delay: -1.8s;
}

.chip-safe {
  right: 24rpx;
  bottom: 68rpx;
  color: #26304f;
  animation-delay: -2.8s;
}

.chip-icon {
  width: 30rpx;
  height: 30rpx;
  flex: 0 0 30rpx;
  color: #5f7ada;
  font-size: 26rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chip-ai .chip-icon,
.chip-safe .chip-icon {
  color: #5f7ada;
}

.chip-chat .chip-icon {
  color: #43a6b9;
}

.chip-reminder .chip-icon {
  color: #de8c57;
}

.chip-heal .chip-icon {
  color: #d67592;
}

.login-actions {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 18rpx;
  margin-top: 184rpx;
  transform: translateY(4rpx);
}

.sr-only-field {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  opacity: 0;
  pointer-events: none;
}

.wechat-btn {
  min-height: 104rpx;
  border-radius: 999rpx;
  color: #fff;
  font-size: 30rpx;
  font-weight: 930;
  line-height: 1.2;
  background: linear-gradient(135deg, #39e58f 0%, #32d97d 44%, #22c56e 100%);
  border: 1rpx solid rgba(255,255,255,.66);
  box-shadow: 0 18rpx 34rpx rgba(32, 197, 110, .26), inset 0 1rpx 0 rgba(255,255,255,.42);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18rpx;
  transition: transform .18s ease, box-shadow .18s ease, opacity .18s ease;
}

.wechat-btn:active {
  transform: translateY(2rpx);
  box-shadow: 0 10rpx 20rpx rgba(32, 197, 110, .22), inset 0 1rpx 0 rgba(255,255,255,.38);
}

.wechat-btn[disabled],
.wechat-btn.disabled {
  opacity: .66;
}

.wechat-mark {
  width: 52rpx;
  height: 52rpx;
  border-radius: 16rpx;
  color: #18b86a;
  background: rgba(255,255,255,.95);
  font-size: 32rpx;
  font-weight: 950;
  display: flex;
  align-items: center;
  justify-content: center;
}

.agreement-row {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10rpx;
  color: #58637d;
  font-size: 22rpx;
  line-height: 1.25;
  white-space: nowrap;
}

.agreement-check {
  width: 32rpx;
  height: 32rpx;
  min-width: 32rpx;
  min-height: 32rpx;
  flex: 0 0 32rpx;
  aspect-ratio: 1 / 1;
  box-sizing: border-box;
  border-radius: 6rpx;
  color: #18c975;
  font-size: 20rpx;
  line-height: 1;
  background: rgba(255,255,255,.94);
  border: 2rpx solid rgba(91, 112, 138, .44);
  box-shadow: inset 0 1rpx 0 rgba(255,255,255,.95), 0 4rpx 10rpx rgba(73,91,146,.08);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color .18s ease, background .18s ease, transform .18s ease;
}

.agreement-check.checked {
  border-color: rgba(34, 197, 110, .9);
  background: rgba(240, 255, 247, .96);
}

.agreement-check:active {
  transform: scale(.92);
}

.agreement-tick {
  width: 20rpx;
  height: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #16c970;
  font-size: 20rpx;
  line-height: 1;
  font-weight: 950;
}

.agreement-copy {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 2rpx;
  flex: 0 1 auto;
  max-width: 610rpx;
  white-space: nowrap;
}

.agreement-link {
  color: #3f61b8;
  font-weight: 900;
}

.error-text {
  padding: 16rpx 18rpx;
  border-radius: 20rpx;
  background: rgba(255,241,242,.72);
  color: #a84b55;
  font-size: 22rpx;
  line-height: 1.5;
  font-weight: 850;
}

@media (max-width: 380px) {
  .login-page {
    padding-top: calc(58rpx + env(safe-area-inset-top));
    padding-bottom: calc(24rpx + env(safe-area-inset-bottom));
    padding-left: 34rpx;
    padding-right: 34rpx;
    gap: 12rpx;
  }

  .login-actions {
    margin-top: 150rpx;
    transform: translateY(2rpx);
  }

  .login-hero {
    transform: translateY(10rpx);
  }

  .app-mark {
    width: 128rpx;
    height: 128rpx;
    border-radius: 38rpx;
  }

  .login-title {
    font-size: 48rpx;
  }

  .illustration-wrap {
    height: 510rpx;
  }

  .mascot-image {
    width: 562rpx;
    height: 492rpx;
    margin-top: 24rpx;
  }

  .feature-chip {
    min-height: 48rpx;
    padding: 9rpx 14rpx;
    font-size: 19rpx;
    max-width: 178rpx;
  }

  .chip-schedule {
    left: 10rpx;
    top: 90rpx;
  }

  .chip-ai {
    right: 14rpx;
    top: 126rpx;
  }

  .chip-chat {
    left: -2rpx;
    top: 258rpx;
  }

  .chip-reminder {
    right: -6rpx;
    top: 274rpx;
  }

  .chip-heal {
    left: 26rpx;
    bottom: 58rpx;
  }

  .chip-safe {
    right: 10rpx;
    bottom: 60rpx;
  }

  .wave-lines {
    left: 86rpx;
    top: 226rpx;
    transform: scale(.88);
  }
}

@keyframes mascotWave {
  0%, 100% {
    transform: translateY(0) rotate(0deg);
  }

  18% {
    transform: translateY(-7rpx) rotate(-1.4deg);
  }

  38% {
    transform: translateY(3rpx) rotate(1.1deg);
  }

  58% {
    transform: translateY(-4rpx) rotate(-.7deg);
  }
}

@keyframes chipFloat {
  0%, 100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(-12rpx);
  }
}

@keyframes waveFloat {
  0%, 100% {
    opacity: .64;
    transform: rotate(-1deg) translateY(0);
  }

  38% {
    opacity: 1;
    transform: rotate(-6deg) translateY(-8rpx);
  }
}

@keyframes waveFade {
  0%, 100% {
    opacity: .28;
  }

  42% {
    opacity: .72;
  }
}

@keyframes dotPop {
  0%, 100% {
    transform: scale(.76);
    opacity: .52;
  }

  42% {
    transform: scale(1);
    opacity: 1;
  }
}

@keyframes glowPulse {
  0%, 100% {
    opacity: .74;
    transform: scaleX(.94);
  }

  50% {
    opacity: 1;
    transform: scaleX(1.04);
  }
}

@media (prefers-reduced-motion: reduce) {
  .illustration-glow,
  .wave-lines,
  .line-two,
  .wave-dot,
  .mascot-image,
  .feature-chip {
    animation: none;
  }
}
</style>
