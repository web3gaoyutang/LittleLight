<template>
  <view class="login-page">
    <view class="brand">
      <view class="brand-row">
        <view class="brand-mark">微</view>
        <view>
          <text class="caption">LittleLight Teacher</text>
          <view class="brand-name">微光老师</view>
        </view>
      </view>
      <view class="login-title">微信登录后进入教师工作台</view>
      <text class="body">课表、待办、家长档案和 AI 生成记录都会跟随当前微信账号隔离保存。</text>
      <view class="hero-preview">
        <view class="preview-row">
          <text class="preview-label">账号</text>
          <text class="preview-main">服务端 session</text>
        </view>
        <view class="preview-row">
          <text class="preview-label">数据</text>
          <text class="preview-main">按用户隔离保存</text>
        </view>
        <view class="preview-row">
          <text class="preview-label">AI</text>
          <text class="preview-main">复核与审计留痕</text>
        </view>
      </view>
    </view>

    <view class="login-panel">
      <view class="panel-head">
        <text class="section-title">{{ loginModeTitle }}</text>
        <text class="hint">{{ hintText }}</text>
      </view>
      <view v-if="showNickInput" class="field">
        <text class="caption">{{ canUseWechatLogin ? '备用昵称' : '调试昵称' }}</text>
        <input class="input" v-model="nickName" placeholder="例如：林老师" aria-label="登录昵称" data-testid="login-nickname-input" />
      </view>
      <button class="wechat-btn" :class="{ disabled: !canUseWechatLogin }" :disabled="loading || !canUseWechatLogin" data-testid="wechat-login-button" @tap="login">
        <text class="wechat-mark">微信</text>
        <text>{{ primaryButtonText }}</text>
      </button>
      <view v-if="showDevLogin" class="dev-zone">
        <view>
          <text class="caption">本地调试</text>
          <text class="dev-copy">当前环境没有微信授权能力时，可创建隔离的开发 session。</text>
        </view>
        <button class="dev-btn" :disabled="loading" data-testid="dev-login-button" @tap="devLogin">
          {{ loading ? '登录中...' : '开发登录' }}
        </button>
      </view>
      <view class="login-foot" :class="{ warning: !canUseWechatLogin && !showDevLogin }">
        <text class="caption">{{ footText }}</text>
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

const nickName = ref('林小微')
const loading = ref(false)
const errorText = ref('')
const showDevLogin = api.isDevAuthAvailable()
const canUseWechatLogin = canRequestWechatLogin()
const showNickInput = showDevLogin || !canUseWechatLogin
const primaryButtonText = computed(() => primaryLoginText())
const loginModeTitle = computed(() => canUseWechatLogin ? '微信授权登录' : (showDevLogin ? '开发环境登录' : '需要微信小程序环境'))
const footText = computed(() => canUseWechatLogin ? '登录成功后会进入首页；退出登录会撤销当前服务端 session。' : (showDevLogin ? '开发登录只用于本地调试，生产环境不会展示。' : '请使用微信小程序版本进入应用。'))
const hintText = ref(loginHintText())

onShow(() => {
  api.resetAuthRedirect()
  if (api.isLoggedIn()) {
    uni.switchTab({ url: '/pages/home/index' })
  }
})

async function login() {
  if (loading.value || !canUseWechatLogin) {
    errorText.value = '请在微信小程序中使用微信登录；H5 本地调试请使用开发登录。'
    return
  }
  loading.value = true
  errorText.value = ''
  try {
    await loginWithWechat()
    showToast('登录成功')
    uni.switchTab({ url: '/pages/home/index' })
  } catch (error) {
    const message = errorMessage(error, canUseWechatLogin ? '微信登录失败' : '开发登录失败')
    errorText.value = showDevLogin ? `${message}；可使用开发登录继续调试。` : message
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
    showToast('开发登录成功')
    uni.switchTab({ url: '/pages/home/index' })
  } catch (err) {
    const message = errorMessage(err, '开发登录失败')
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
    uni.login({
      provider: 'weixin',
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
  // #ifdef MP-WEIXIN
  return true
  // #endif
  return false
}

function primaryLoginText() {
  if (loading.value) return '登录中...'
  if (canUseWechatLogin) return '微信登录'
  return '请在微信小程序中登录'
}

function loginHintText() {
  if (canUseWechatLogin) return '微信授权会保存服务端 session token，后续请求仅使用当前账号数据。'
  if (showDevLogin) return '当前运行环境不支持微信授权，本地调试会使用开发登录创建服务端 session。'
  return '当前 H5 环境未开放真实微信授权，请使用微信小程序或开启本地开发登录。'
}

</script>

<style src="../../static/common.css"></style>
<style scoped>
.login-page { min-height: 100vh; padding: 72rpx 36rpx 56rpx; box-sizing: border-box; display: flex; flex-direction: column; justify-content: space-between; background: linear-gradient(155deg,#ecfeff 0%,#f8fbff 46%,#f1f8ee 100%); gap: 40rpx; }
.brand { padding-top: 48rpx; display: flex; flex-direction: column; gap: 22rpx; }
.brand-row { display: flex; align-items: center; gap: 22rpx; }
.brand-mark { width: 124rpx; height: 124rpx; border-radius: 36rpx; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 52rpx; font-weight: 950; background: linear-gradient(135deg,#0891b2,#059669); box-shadow: 0 24rpx 54rpx rgba(8,145,178,.24); }
.brand-name { margin-top: 4rpx; color: #14313b; font-size: 34rpx; font-weight: 950; }
.login-title { font-size: 54rpx; line-height: 1.14; color: #14313b; font-weight: 950; }
.login-panel { padding: 32rpx; border-radius: 32rpx; background: rgba(255,255,255,.86); box-shadow: 0 22rpx 58rpx rgba(20,78,99,.12); display: flex; flex-direction: column; gap: 22rpx; }
.panel-head { display: flex; flex-direction: column; gap: 8rpx; }
.field { display: flex; flex-direction: column; gap: 12rpx; }
.wechat-btn { min-height: 94rpx; border-radius: 28rpx; color: #fff; font-size: 30rpx; font-weight: 950; background: linear-gradient(135deg,#059669,#0891b2); display: flex; align-items: center; justify-content: center; gap: 16rpx; }
.wechat-btn[disabled], .wechat-btn.disabled { opacity: .58; }
.wechat-mark { padding: 6rpx 12rpx; border-radius: 14rpx; color: #059669; background: rgba(255,255,255,.92); font-size: 22rpx; font-weight: 950; }
.dev-btn { min-height: 82rpx; border-radius: 24rpx; color: #0e7490; font-size: 26rpx; font-weight: 900; background: rgba(236,254,255,.86); }
.hero-preview { margin-top: 10rpx; padding: 22rpx; border-radius: 28rpx; background: rgba(255,255,255,.62); display: flex; flex-direction: column; gap: 14rpx; box-shadow: 0 18rpx 42rpx rgba(20,78,99,.08); }
.preview-row { display: flex; align-items: center; justify-content: space-between; gap: 18rpx; }
.preview-label { flex: 0 0 auto; color: #0e7490; font-size: 22rpx; font-weight: 950; }
.preview-main { min-width: 0; color: #14313b; font-size: 24rpx; font-weight: 900; text-align: right; }
.dev-zone { padding: 22rpx; border-radius: 24rpx; background: rgba(247,252,253,.92); display: flex; align-items: center; justify-content: space-between; gap: 18rpx; }
.dev-zone > view { min-width: 0; display: flex; flex-direction: column; gap: 6rpx; }
.dev-copy { color: #6b778f; font-size: 22rpx; line-height: 1.45; }
.dev-zone .dev-btn { min-width: 168rpx; padding: 0 20rpx; }
.hint { font-size: 22rpx; line-height: 1.5; color: #6b778f; }
.login-foot { padding-top: 6rpx; border-top: 1px solid rgba(8,145,178,.12); }
.login-foot.warning { border-top-color: rgba(185,92,97,.18); }
.error-text { font-size: 22rpx; line-height: 1.5; color: #b95c61; font-weight: 800; }
</style>
