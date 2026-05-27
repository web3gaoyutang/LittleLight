<template>
  <view class="page-wrap">
    <view class="header"><text class="caption">AI 回复 · 重点关注</text><view class="title">沟通助手</view></view>
    <view class="card">
      <text class="section-title">生成家校草稿</text>
      <textarea class="textarea" v-model="issue" placeholder="描述学生情况、家长问题和沟通目标" />
      <view class="row picker-row">
        <picker class="picker" :range="styles" @change="style = styles[$event.detail.value]"><text>{{ style }}</text></picker>
        <picker class="picker" :range="tones" @change="tone = tones[$event.detail.value]"><text>{{ tone }}</text></picker>
      </view>
      <button class="primary-btn" @tap="generate">生成草稿</button>
    </view>
    <view class="section-head row between"><text class="section-title">可选草稿</text><text class="tag">自动适配</text></view>
    <view v-for="draft in drafts" :key="draft.id" class="card draft">
      <view class="row between"><text class="tag">{{ draft.version }} · {{ draft.style }}</text><button class="ghost-btn small" @tap="copy(draft.content)">复制</button></view>
      <text class="body">{{ draft.content }}</text>
    </view>
    <view class="section-head row between"><text class="section-title">重点关注</text><text class="caption">家长档案</text></view>
    <view v-for="parent in parents" :key="parent.id" class="card"><text class="section-title">{{ parent.parentName }}</text><text class="body">{{ parent.studentName }} · {{ parent.className }} · {{ parent.nextAction }}</text></view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'
const issue = ref('浩宇妈妈询问孩子最近在课堂上的专注度情况，希望语气温和一点，同时给出可执行建议。')
const styles = ['容易焦虑','比较敏感','沟通积极','关注成绩']
const tones = ['温和','正式','简短','坚定但礼貌']
const style = ref(styles[0])
const tone = ref(tones[0])
const drafts = ref([])
const parents = ref([])
onShow(async () => { parents.value = await api.parents(); if (!drafts.value.length) await generate() })
async function generate() { drafts.value = await api.parentDrafts({ issue: issue.value, parentStyle: style.value, tone: tone.value }); showToast('已生成多个沟通版本') }
function copy(content) { uni.setClipboardData({ data: content }); showToast('已复制草稿') }
</script>

<style src="../../static/common.css"></style>
<style scoped>.page-wrap{padding:28rpx 0 120rpx}.header,.section-head{padding:0 32rpx 12rpx}.picker-row{margin:20rpx 0}.picker{flex:1}.draft{border:2rpx solid rgba(169,144,234,.28)}.small{min-height:64rpx;padding:0 24rpx}</style>
