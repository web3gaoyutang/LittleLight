<template>
  <view class="card app-state" :class="[`state-${type}`, { compact }]">
    <view class="state-icon">{{ iconText }}</view>
    <view class="state-copy">
      <text v-if="title" class="section-title">{{ title }}</text>
      <text v-if="message" class="body">{{ message }}</text>
    </view>
    <button v-if="actionText" class="ghost-btn small retry" @tap="$emit('action')">{{ actionText }}</button>
  </view>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  type: { type: String, default: 'info' },
  title: { type: String, default: '' },
  message: { type: String, default: '' },
  actionText: { type: String, default: '' },
  compact: { type: Boolean, default: false }
})

defineEmits(['action'])

const iconText = computed(() => ({
  loading: '...',
  error: '!',
  empty: '+',
  info: 'i'
})[props.type] || 'i')
</script>

<style scoped>
.app-state {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: 20rpx;
}
.app-state.compact {
  margin-top: 16rpx;
  margin-bottom: 16rpx;
  padding: 24rpx;
}
.state-icon {
  flex: 0 0 auto;
  width: 52rpx;
  height: 52rpx;
  border-radius: 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #0e7490;
  background: rgba(236,254,255,.92);
  font-size: 22rpx;
  font-weight: 950;
}
.state-error .state-icon {
  color: #9f4b52;
  background: rgba(255,241,242,.94);
}
.state-empty .state-icon {
  color: #31505b;
  background: rgba(241,248,238,.94);
}
.state-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8rpx;
}
.retry {
  flex: 0 0 auto;
  align-self: center;
  min-height: 60rpx;
  padding: 0 24rpx;
}
</style>
