<template>
  <view class="app-dock" data-testid="app-dock">
    <button class="dock-item" :class="{ active: current === 'home' }" @tap="go('/pages/home/index')">
      <AppIcon name="home" />
      <text>首页</text>
    </button>
    <button class="dock-item" :class="{ active: current === 'communication' }" @tap="go('/pages/communication/index')">
      <AppIcon name="message" />
      <text>沟通</text>
    </button>
    <view class="dock-mic-slot" aria-hidden="true"></view>
    <button class="dock-item" :class="{ active: current === 'schedule' }" @tap="go('/pages/schedule/index')">
      <AppIcon name="calendar" />
      <text>日程</text>
    </button>
    <button class="dock-item" :class="{ active: current === 'profile' }" @tap="go('/pages/profile/index')">
      <AppIcon name="user" />
      <text>我的</text>
    </button>
  </view>
</template>

<script setup>
import AppIcon from './AppIcon.vue'

defineProps({
  current: { type: String, default: 'home' }
})

function go(url) {
  uni.switchTab({ url })
}
</script>

<style scoped>
.app-dock {
  position: fixed;
  left: 50%;
  bottom: calc(10px + env(safe-area-inset-bottom, 0px));
  z-index: 4900;
  transform: translateX(-50%);
  width: min(calc(100% - 28px), 360px);
  height: 72px;
  padding: 8px 10px;
  border-radius: 28px;
  box-sizing: border-box;
  display: grid;
  grid-template-columns: 1fr 1fr 64px 1fr 1fr;
  align-items: center;
  gap: 1px;
  border: 1px solid rgba(255,255,255,.72);
  background: linear-gradient(180deg, rgba(255,255,255,.88), rgba(246,250,255,.74));
  box-shadow: 0 1px 0 rgba(255,255,255,.92) inset, 0 14px 34px rgba(73,91,146,.13);
  backdrop-filter: blur(24px) saturate(1.18);
}

.app-dock::before {
  content: "";
  position: absolute;
  inset: -14px 4px -10px;
  z-index: -1;
  pointer-events: none;
  border-radius: 34px;
  background: linear-gradient(180deg, rgba(250,250,255,0), rgba(250,250,255,.68) 54%, rgba(250,250,255,.92));
}

.dock-item {
  min-width: 0;
  height: 56px;
  padding: 0;
  border-radius: 22px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  color: #767a96;
  font-size: 10.5px;
  line-height: 1.15;
  font-weight: 900;
  background: transparent;
  transition: transform .18s ease, background .18s ease, box-shadow .18s ease, color .18s ease;
}

.dock-item .app-icon {
  width: 27px;
  height: 27px;
  padding: 4px;
  box-sizing: border-box;
}

.dock-item.active {
  color: #3f5a94;
  background: rgba(239,244,255,.86);
  box-shadow: inset 0 1px 0 rgba(255,255,255,.92), 0 8px 18px rgba(73,91,146,.08);
}

.dock-item:active {
  transform: translateY(1px);
}

.dock-mic-slot {
  width: 64px;
  height: 56px;
  pointer-events: none;
}
</style>
