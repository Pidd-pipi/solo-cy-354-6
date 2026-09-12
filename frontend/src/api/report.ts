import request from '../utils/request'
import type { PageResult, Report } from '../types'

export function createReport(data: { target_type: string; target_id: number; reason: string; description: string }) {
  return request.post<never, { code: number; message: string; data: Report }>('/reports', data)
}

export function listAdminReports(params: { page?: number; page_size?: number; status?: string }) {
  return request.get<never, { code: number; message: string; data: PageResult<Report> }>('/admin/reports', { params })
}

export function handleReport(id: number, data: { action: string; result?: string }) {
  return request.post<never, { code: number; message: string; data: Report }>(`/admin/reports/${id}/handle`, data)
}
