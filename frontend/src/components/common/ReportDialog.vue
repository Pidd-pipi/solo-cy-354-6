<template>
  <el-dialog v-model="visible" title="发起举报" width="440px" :close-on-click-modal="false">
    <el-alert type="warning" :closable="false" class="tip">
      举报对象：{{ targetLabel }}。同一对象仅保留一条待处理举报，请如实填写。
    </el-alert>
    <el-form label-width="70px" class="form">
      <el-form-item label="理由" required>
        <el-select v-model="form.reason" placeholder="请选择举报理由" style="width: 100%">
          <el-option v-for="r in REPORT_REASONS" :key="r.value" :label="r.label" :value="r.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="说明" required>
        <el-input v-model="form.description" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="请描述具体情况（必填，500 字以内）" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="danger" :loading="submitting" @click="submit">提交举报</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { REPORT_REASONS, reportTargetTypeLabel } from '../../constants/report'
import { createReport } from '../../api/report'

const props = defineProps<{ targetType: string; targetId: number }>()
const emit = defineEmits<{ (e: 'reported'): void }>()

const visible = ref(false)
const submitting = ref(false)
const form = reactive({ reason: '', description: '' })

const targetLabel = computed(() => `${reportTargetTypeLabel(props.targetType)} #${props.targetId}`)

function open() {
  form.reason = ''
  form.description = ''
  visible.value = true
}

async function submit() {
  if (!form.reason) {
    ElMessage.warning('请选择举报理由')
    return
  }
  if (!form.description.trim()) {
    ElMessage.warning('请填写举报说明')
    return
  }
  submitting.value = true
  try {
    await createReport({
      target_type: props.targetType,
      target_id: props.targetId,
      reason: form.reason,
      description: form.description.trim(),
    })
    ElMessage.success('举报已提交，等待管理员处理')
    visible.value = false
    emit('reported')
  } finally {
    submitting.value = false
  }
}

defineExpose({ open })
</script>

<style scoped>
.tip {
  margin-bottom: 12px;
}
.form {
  margin-top: 4px;
}
</style>
