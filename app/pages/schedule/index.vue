<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">课程 · 待办 · 提醒</text>
        <view class="title">今日安排</view>
      </view>
      <button class="bell" @tap="addReminder">+</button>
    </view>

    <view class="week-row">
      <view v-for="day in days" :key="day.iso" class="day" :class="{ active: selectedDay === day.iso }" @tap="selectDay(day)">
        <text>{{ day.label }}</text>
        <text>{{ day.date }}</text>
      </view>
    </view>

    <view class="section-head row between">
      <text class="section-title">本周日程</text>
      <button class="ghost-btn small" @tap="importSchedule">导入 Excel 课表</button>
    </view>
    <view class="section-head row between compact">
      <text class="caption">当日课程</text>
      <button class="primary-btn small" @tap="addCourse">添加日程</button>
    </view>
    <view v-for="course in courses" :key="course.id" class="card">
      <view class="row between">
        <text class="tag">{{ course.startTime }} - {{ course.endTime }}</text>
        <button class="ghost-btn mini" @tap="removeCourse(course.id)">删除</button>
      </view>
      <text class="section-title">{{ course.className }} · {{ course.title }}</text>
      <text class="body">{{ course.note }} · {{ course.location }}</text>
    </view>

    <view class="section-head row between">
      <text class="section-title">待办事项</text>
      <button class="primary-btn small" @tap="addReminder">添加待办</button>
    </view>
    <view v-for="item in reminders" :key="item.id" class="card">
      <view class="row between">
        <text class="tag">{{ formatTime(item.remindAt) }} · {{ statusText(item.status) }}</text>
        <view class="row action-row">
          <button class="ghost-btn mini" @tap="snooze(item.id)">延后</button>
          <button class="ghost-btn mini" @tap="complete(item.id)">完成</button>
          <button class="ghost-btn mini danger" @tap="removeReminder(item.id)">删除</button>
        </view>
      </view>
      <text class="section-title">{{ item.title }}</text>
      <text class="body">{{ item.note }}</text>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'

const today = new Date()
const selectedDay = ref(toISODate(today))
const weekday = ref(today.getDay())
const courses = ref([])
const reminders = ref([])
const days = buildWeek(today)

onShow(load)

async function load() {
  courses.value = await api.courses(weekday.value)
  reminders.value = await api.reminders(selectedDay.value)
}

async function selectDay(day) {
  selectedDay.value = day.iso
  weekday.value = day.weekday
  await load()
}

async function addCourse() {
  const item = await api.createCourse({
    title: '新的课程安排',
    className: '待选择班级',
    location: '待填写地点',
    weekday: weekday.value,
    startTime: '10:00',
    endTime: '10:45',
    note: '从日程页快速添加'
  })
  courses.value.push(item)
  showToast('已添加到当日课程')
}

async function removeCourse(id) {
  await api.deleteCourse(id)
  showToast('已删除课程')
  await load()
}

async function addReminder() {
  const remindAt = new Date(`${selectedDay.value}T17:00:00`)
  const item = await api.createReminder({
    title: '新的待办提醒',
    category: '个人事项',
    note: '从日程页快速添加，可进入详情继续编辑',
    remindAt: remindAt.toISOString()
  })
  reminders.value.unshift(item)
  showToast('已添加待办')
}

async function complete(id) {
  await api.completeReminder(id)
  showToast('已完成提醒')
  await load()
}

async function snooze(id) {
  const until = new Date(Date.now() + 30 * 60 * 1000).toISOString()
  await api.snoozeReminder(id, until)
  showToast('已延后 30 分钟')
  await load()
}

async function removeReminder(id) {
  await api.deleteReminder(id)
  showToast('已删除提醒')
  await load()
}

function importSchedule() {
  uni.chooseFile({
    count: 1,
    extension: ['.xlsx', '.xls'],
    success: () => showToast('已选择 Excel，后续将进入导入确认')
  })
}

function formatTime(value) {
  return new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function statusText(status) {
  return ({ pending: '待处理', done: '已完成', snoozed: '已延后' })[status] || '待处理'
}

function buildWeek(base) {
  const labels = ['日', '一', '二', '三', '四', '五', '六']
  const start = new Date(base)
  start.setDate(base.getDate() - base.getDay())
  return labels.map((label, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    return { label, date: String(date.getDate()).padStart(2, '0'), weekday: index, iso: toISODate(date) }
  })
}

function toISODate(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header, .section-head { padding: 0 32rpx 12rpx; }
.compact { padding-bottom: 0; }
.bell { width: 88rpx; height: 88rpx; border-radius: 34rpx; background: rgba(255,255,255,.86); font-size: 42rpx; color: #435376; }
.week-row { display: flex; gap: 16rpx; padding: 16rpx 32rpx; overflow-x: auto; }
.day { min-width: 88rpx; height: 116rpx; border-radius: 34rpx; background: rgba(255,255,255,.72); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10rpx; color: #58627d; font-weight: 900; }
.day.active { color: #fff; background: linear-gradient(145deg,#6f86df,#52b8cf); }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
</style>
