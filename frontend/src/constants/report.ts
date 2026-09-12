export const REPORT_TARGET_TYPES = [
  { value: 'product', label: '商品' },
  { value: 'trade_order', label: '交易订单' },
] as const

export const REPORT_REASONS = [
  { value: 'fraud', label: '诈骗行为' },
  { value: 'fake', label: '虚假信息' },
  { value: 'prohibited', label: '违禁物品' },
  { value: 'abuse', label: '辱骂骚扰' },
  { value: 'other', label: '其他' },
] as const

export const REPORT_STATUSES = [
  { value: 'pending', label: '待处理', type: 'warning' },
  { value: 'rejected', label: '已驳回', type: 'info' },
  { value: 'resolved', label: '已处理', type: 'success' },
] as const

export const REPORT_ACTIONS = [
  { value: 'reject', label: '驳回举报' },
  { value: 'takedown', label: '下架商品' },
] as const

export function reportTargetTypeLabel(value: string): string {
  return REPORT_TARGET_TYPES.find((t) => t.value === value)?.label ?? value
}

export function reportReasonLabel(value: string): string {
  return REPORT_REASONS.find((r) => r.value === value)?.label ?? value
}

export function reportStatusLabel(value: string): string {
  return REPORT_STATUSES.find((s) => s.value === value)?.label ?? value
}

export function reportStatusType(value: string): string {
  return REPORT_STATUSES.find((s) => s.value === value)?.type ?? 'info'
}
