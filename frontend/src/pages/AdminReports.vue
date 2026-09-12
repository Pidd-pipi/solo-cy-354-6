<template>
  <div class="page">
    <h2>举报处理</h2>
    <el-form inline class="filters">
      <el-form-item label="状态">
        <el-select v-model="status" style="width: 140px" @change="load">
          <el-option v-for="s in REPORT_STATUSES" :key="s.value" :label="s.label" :value="s.value" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="load">刷新</el-button>
      </el-form-item>
    </el-form>
    <el-table :data="reports" v-loading="loading">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="举报对象" width="140">
        <template #default="{ row }">
          {{ reportTargetTypeLabel(row.target_type) }} #{{ row.target_id }}
        </template>
      </el-table-column>
      <el-table-column label="关联商品" width="110">
        <template #default="{ row }">商品 #{{ row.product_id }}</template>
      </el-table-column>
      <el-table-column label="理由" width="110">
        <template #default="{ row }">{{ reportReasonLabel(row.reason) }}</template>
      </el-table-column>
      <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip />
      <el-table-column label="举报人" width="90">
        <template #default="{ row }">用户 #{{ row.reporter_id }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="reportStatusType(row.status) as any" size="small">{{ reportStatusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="举报时间" width="150">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="处理结果" min-width="160">
        <template #default="{ row }">
          <template v-if="row.handled_at">
            <div>{{ row.result || '—' }}</div>
            <div class="handled-at">{{ formatDateTime(row.handled_at) }} · 管理员 #{{ row.handled_by }}</div>
          </template>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <el-button size="small" @click="reject(row)">驳回</el-button>
            <el-button size="small" type="danger" @click="takedown(row)">下架商品</el-button>
          </template>
          <span v-else>已处理</span>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && reports.length === 0" description="暂无举报记录" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listAdminReports, handleReport } from '../api/report'
import { REPORT_STATUSES, reportReasonLabel, reportStatusLabel, reportStatusType, reportTargetTypeLabel } from '../constants/report'
import { formatDateTime } from '../utils/dateFormat'
import type { Report } from '../types'

const reports = ref<Report[]>([])
const loading = ref(false)
const status = ref('pending')

async function load() {
  loading.value = true
  try {
    const res = await listAdminReports({ page: 1, page_size: 100, status: status.value })
    reports.value = res.data.items
  } finally {
    loading.value = false
  }
}

async function reject(row: Report) {
  const { value } = await ElMessageBox.prompt('请输入驳回原因（可选）', `驳回举报 #${row.id}`, {
    confirmButtonText: '确认驳回',
    cancelButtonText: '取消',
    inputPlaceholder: '例如：证据不足',
    inputValidator: () => true,
  }).catch(() => ({ value: undefined as string | undefined }))
  if (value === undefined) return
  await handleReport(row.id, { action: 'reject', result: value || '举报不成立' })
  ElMessage.success('举报已驳回')
  await load()
}

async function takedown(row: Report) {
  const { value } = await ElMessageBox.prompt(`下架商品 #${row.product_id} 并结案该举报？`, `下架商品（举报 #${row.id}）`, {
    confirmButtonText: '确认下架',
    cancelButtonText: '取消',
    inputPlaceholder: '处理结果，例如：违规商品已下架',
    inputValidator: () => true,
  }).catch(() => ({ value: undefined as string | undefined }))
  if (value === undefined) return
  await handleReport(row.id, { action: 'takedown', result: value || '违规商品已下架' })
  ElMessage.success('商品已下架，举报处理完成')
  await load()
}

onMounted(load)
</script>

<style scoped>
.filters {
  margin-bottom: 8px;
}
.handled-at {
  color: #909399;
  font-size: 12px;
}
</style>
