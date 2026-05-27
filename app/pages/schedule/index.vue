<template>
  <view class="page-wrap">
    <view class="header row between"><view><text class="caption">课程 · 待办 · 提醒</text><view class="title">今日安排</view></view><button class="bell" @tap="openAdd">+</button></view>
    <view class="week-row">
      <view v-for="day in days" :key="day.weekday" class="day" :class="{ active: weekday === day.weekday }" @tap="selectDay(day.weekday)"><text>{{ day.label }}</text><text>{{ day.date }}</text></view>
    </view>
    <view class="section-head row between"><text class="section-title">当日课程</text><text class="caption">导入课表</text></view>
    <view v-for="course in courses" :key="course.id" class="card"><text class="tag">{{ course.startTime }}</text><text class="section-title">{{ course.className }} · {{ course.title }}</text><text class="body">{{ course.note }} · {{ course.location }}</text></view>
    <view class="section-head row between"><text class="section-title">待办事项</text><text class="caption" @tap="openAdd">添加待办</text></view>
    <view v-for="item in reminders" :key="item.id" class="card"><view class="row between"><text class="tag">{{ formatTime(item.remindAt) }}</text><button class="ghost-btn done" @tap="complete(item.id)">完成</button></view><text class="section-title">{{ item.title }}</text><text class="body">{{ item.note }}</text></view>
  </view>
</template>
<script setup>
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'
const weekday = ref(new Date().getDay())
const courses = ref([])
const reminders = ref([])
const days = [{label:'日',date:'18',weekday:0},{label:'一',date:'12',weekday:1},{label:'二',date:'13',weekday:2},{label:'三',date:'14',weekday:3},{label:'四',date:'15',weekday:4},{label:'五',date:'16',weekday:5},{label:'六',date:'17',weekday:6}]
onShow(load)
async function load(){ courses.value = await api.courses(weekday.value); reminders.value = await api.reminders() }
async function selectDay(day){ weekday.value = day; await load() }
function formatTime(value){ return new Date(value).toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit'}) }
async function complete(id){ await api.completeReminder(id); showToast('已完成提醒'); await load() }
async function openAdd(){ const item = await api.createReminder({ title:'新的待办提醒', category:'个人事项', note:'从日程页快速添加', remindAt:new Date(Date.now()+3600000).toISOString() }); reminders.value.unshift(item); showToast('已添加待办') }
</script>
<style src="../../static/common.css"></style>
<style scoped>.page-wrap{padding:28rpx 0 120rpx}.header,.section-head{padding:0 32rpx 12rpx}.bell{width:88rpx;height:88rpx;border-radius:34rpx;background:rgba(255,255,255,.86);font-size:42rpx;color:#435376}.week-row{display:flex;gap:16rpx;padding:16rpx 32rpx;overflow-x:auto}.day{min-width:88rpx;height:116rpx;border-radius:34rpx;background:rgba(255,255,255,.72);display:flex;flex-direction:column;align-items:center;justify-content:center;gap:10rpx;color:#58627d;font-weight:900}.day.active{color:#fff;background:linear-gradient(145deg,#6f86df,#52b8cf)}.done{min-height:60rpx;padding:0 22rpx}</style>
