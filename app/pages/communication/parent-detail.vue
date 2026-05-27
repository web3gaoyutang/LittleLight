<template>
  <view class="page-wrap">
    <view class="header row between">
      <view>
        <text class="caption">学生 · 班级 · 风险 · 跟进</text>
        <view class="title">{{ form.parentName || '家长档案' }}</view>
      </view>
      <button class="ghost-btn small" @tap="load">刷新</button>
    </view>

    <view class="card">
      <view class="grid">
        <view class="field">
          <text class="caption">学生</text>
          <input class="input" v-model="form.studentName" placeholder="学生姓名" />
        </view>
        <view class="field">
          <text class="caption">班级</text>
          <input class="input" v-model="form.className" placeholder="班级" />
        </view>
        <view class="field">
          <text class="caption">家长称呼</text>
          <input class="input" v-model="form.parentName" placeholder="家长称呼" />
        </view>
        <view class="field">
          <text class="caption">关系</text>
          <input class="input" v-model="form.relationship" placeholder="关系" />
        </view>
        <view class="field">
          <text class="caption">联系方式</text>
          <input class="input" v-model="form.contact" placeholder="手机号或微信" />
        </view>
        <view class="field">
          <text class="caption">沟通风格</text>
          <picker class="picker" :range="styles" @change="form.communicationStyle = styles[$event.detail.value]">
            <text>{{ form.communicationStyle || '选择风格' }}</text>
          </picker>
        </view>
      </view>

      <view class="field">
        <text class="caption">风险等级</text>
        <view class="risk-row">
          <button v-for="risk in risks" :key="risk.value" class="risk-btn" :class="{ active: form.riskLevel === risk.value }" @tap="form.riskLevel = risk.value">
            {{ risk.label }}
          </button>
        </view>
      </view>

      <view class="field">
        <text class="caption">重点备注</text>
        <textarea class="textarea" v-model="form.importantNotes" placeholder="睡眠、作业、情绪、家庭沟通背景等" />
      </view>
      <view class="field">
        <text class="caption">下一步动作</text>
        <textarea class="textarea short" v-model="form.nextAction" placeholder="下一次跟进要做什么" />
      </view>
      <button class="primary-btn save-btn" @tap="save">保存档案</button>
    </view>

    <view class="section-head row between">
      <text class="section-title">沟通记录</text>
      <button class="primary-btn small" @tap="addRecord">添加记录</button>
    </view>
    <view v-for="record in records" :key="record.id" class="card record">
      <view class="row between">
        <text class="tag">{{ record.channel }} · {{ riskText(record.riskLevel) }}</text>
        <text class="caption">{{ formatTime(record.followUpAt) }}</text>
      </view>
      <text class="section-title">{{ record.reason }}</text>
      <text class="body">{{ record.summary }}</text>
      <text class="caption">{{ record.result }}</text>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { api } from '../../api/client'
import { showToast } from '../../utils/ui'

const id = ref('')
const form = ref({})
const records = ref([])
const styles = ['容易焦虑', '比较敏感', '沟通积极', '关注成绩']
const risks = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' }
]

onLoad(async (query) => {
  id.value = query?.id || ''
  await load()
})

async function load() {
  if (!id.value) return
  form.value = await api.parent(id.value)
  records.value = await api.records(id.value)
}

async function save() {
  form.value = await api.updateParent(id.value, form.value)
  showToast('已保存家长档案')
}

async function addRecord() {
  const record = await api.createRecord({
    parentId: id.value,
    student: form.value.studentName || '未填写学生',
    channel: '微信',
    reason: '详情页补充',
    summary: form.value.nextAction || '补充本次沟通要点',
    result: '待跟进',
    riskLevel: form.value.riskLevel || 'low',
    followUpAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()
  })
  records.value.unshift(record)
  showToast('已添加沟通记录')
}

function riskText(value) {
  return ({ low: '低风险', medium: '中风险', high: '高风险' })[value] || '低风险'
}

function formatTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}
</script>

<style src="../../static/common.css"></style>
<style scoped>
.page-wrap { padding: 28rpx 0 120rpx; }
.header, .section-head { padding: 0 32rpx 12rpx; }
.grid { display: grid; grid-template-columns: 1fr; gap: 18rpx; }
.field { display: flex; flex-direction: column; gap: 10rpx; margin-bottom: 22rpx; }
.input, .picker { min-height: 82rpx; }
.textarea.short { min-height: 120rpx; }
.risk-row { display: flex; gap: 14rpx; }
.risk-btn { flex: 1; min-height: 70rpx; border-radius: 24rpx; background: rgba(255,255,255,.72); color: #53607a; font-weight: 900; }
.risk-btn.active { background: linear-gradient(135deg,#6f86df,#52b8cf); color: #fff; }
.save-btn { width: 100%; margin-top: 10rpx; }
.small { min-height: 64rpx; padding: 0 24rpx; font-size: 24rpx; }
.record { display: flex; flex-direction: column; gap: 14rpx; }
</style>
