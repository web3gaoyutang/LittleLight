<template>
  <view class="page-wrap">
    <view class="header">
      <text class="caption">AI 回复 · 重点关注</text>
      <view class="title">沟通助手</view>
    </view>

    <view class="card">
      <text class="section-title">生成家校草稿</text>
      <textarea class="textarea" v-model="issue" placeholder="描述学生情况、家长问题和沟通目标" />
      <view class="row picker-row">
        <picker class="picker" :range="styles" @change="style = styles[$event.detail.value]"><text>{{ style }}</text></picker>
        <picker class="picker" :range="tones" @change="tone = tones[$event.detail.value]"><text>{{ tone }}</text></picker>
      </view>
      <button class="primary-btn" @tap="generate">生成草稿</button>
    </view>

    <view class="section-head row between">
      <text class="section-title">可选草稿</text>
      <text class="tag">自动适配</text>
    </view>
    <view v-for="draft in drafts" :key="draft.id" class="card draft">
      <view class="row between">
        <text class="tag">{{ draft.version }} · {{ draft.style }}</text>
        <button class="ghost-btn small" @tap="copy(draft.content)">复制</button>
      </view>
      <text class="body">{{ draft.content }}</text>
    </view>

    <view class="section-head row between">
      <text class="section-title">重点关注</text>
      <button class="primary-btn small" @tap="addParent">新增家长</button>
    </view>
    <view v-for="parent in parents" :key="parent.id" class="card" @tap="selectParent(parent)">
      <view class="row between">
        <text class="section-title">{{ parent.parentName }}</text>
        <button class="ghost-btn mini" @tap.stop="editParent(parent)">编辑</button>
      </view>
      <text class="body">{{ parent.studentName }} · {{ parent.className }} · {{ parent.nextAction }}</text>
    </view>

    <view class="section-head row between">
      <text class="section-title">沟通记录</text>
      <button class="primary-btn small" @tap="addRecord">手动添加</button>
    </view>
    <view v-for="record in records" :key="record.id" class="card">
      <view class="row between">
        <text class="tag">{{ record.channel }} · {{ record.riskLevel }}</text>
        <view class="row action-row">
          <button class="ghost-btn mini" @tap="editRecord(record)">编辑</button>
          <button class="ghost-btn mini danger" @tap="removeRecord(record.id)">删除</button>
        </view>
      </view>
      <text class="section-title">{{ record.student }} · {{ record.reason }}</text>
      <text class="body">{{ record.summary }}</text>
      <text class="caption">{{ record.result }}</text>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'

const issue = ref('浩宇妈妈询问孩子最近在课堂上的专注度情况，希望语气温和一点，同时给出可执行建议。')
const styles = ['容易焦虑', '比较敏感', '沟通积极', '关注成绩']
const tones = ['温和', '正式', '简短', '坚定但礼貌']
const style = ref(styles[0])
const tone = ref(tones[0])
const drafts = ref([])
const parents = ref([])
const records = ref([])
const activeParent = ref(null)

onShow(async () => {
  await load()
  if (!drafts.value.length) await generate()
})

async function load() {
  parents.value = await api.parents()
  if (!activeParent.value && parents.value.length) activeParent.value = parents.value[0]
  records.value = await api.records(activeParent.value?.id)
}

async function selectParent(parent) {
  activeParent.value = parent
  records.value = await api.records(parent.id)
  showToast(`已切换到 ${parent.studentName}`)
}

async function addParent() {
  const parent = await api.createParent({
    studentName: '新学生',
    className: '待填写班级',
    parentName: '新家长',
    relationship: '家长',
    contact: '',
    communicationStyle: '沟通积极',
    riskLevel: 'low',
    importantNotes: '可在详情中补充背景信息',
    nextAction: '确认下一步沟通目标'
  })
  parents.value.unshift(parent)
  activeParent.value = parent
  showToast('已新增家长档案')
}

async function editParent(parent) {
  const data = { ...parent, nextAction: `${parent.nextAction || '继续跟进'}（已更新）` }
  const updated = await api.updateParent(parent.id, data)
  parents.value = parents.value.map((item) => item.id === updated.id ? updated : item)
  activeParent.value = updated
  showToast('已更新家长档案')
}

async function addRecord() {
  const parent = activeParent.value || parents.value[0]
  const record = await api.createRecord({
    parentId: parent?.id,
    student: parent?.studentName || '未关联学生',
    channel: '微信',
    reason: '手动记录',
    summary: '补充今天的沟通要点',
    result: '等待后续跟进',
    riskLevel: parent?.riskLevel || 'low',
    followUpAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()
  })
  records.value.unshift(record)
  showToast('已添加沟通记录')
}

async function editRecord(record) {
  const updated = await api.updateRecord(record.id, { ...record, summary: `${record.summary}（已编辑）` })
  records.value = records.value.map((item) => item.id === updated.id ? updated : item)
  showToast('已更新沟通记录')
}

async function removeRecord(id) {
  await api.deleteRecord(id)
  records.value = records.value.filter((item) => item.id !== id)
  showToast('已删除沟通记录')
}

async function generate() {
  drafts.value = await api.parentDrafts({ issue: issue.value, parentStyle: style.value, tone: tone.value })
  showToast('已生成多个沟通版本')
}

function copy(content) {
  uni.setClipboardData({ data: content })
  showToast('已复制草稿')
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header, .section-head { padding: 0 32rpx 12rpx; }
.picker-row { margin: 20rpx 0; }
.picker { flex: 1; }
.draft { border: 2rpx solid rgba(169,144,234,.28); }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.mini { min-height: 56rpx; padding: 0 18rpx; font-size: 22rpx; }
.danger { color: #b95c61; }
.action-row { gap: 10rpx; flex-wrap: wrap; justify-content: flex-end; }
</style>
