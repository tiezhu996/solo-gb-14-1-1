// 错题本 API
import request from '../utils/request'
import type { Mistake, MistakeListParams, MistakePayload, PageData, ReviewPayload } from '../types'

export function listMistakes(params: MistakeListParams) {
  return request.get<unknown, PageData<Mistake>>('/mistakes', { params })
}

export function listDueMistakes() {
  return request.get<unknown, PageData<Mistake>>('/mistakes/due')
}

export function getKnowledgePoints() {
  return request.get<unknown, string[]>('/mistakes/knowledge-points')
}

export function createMistake(payload: MistakePayload) {
  return request.post<unknown, Mistake>('/mistakes', payload)
}

export function updateMistake(id: string, payload: MistakePayload) {
  return request.put<unknown, Mistake>(`/mistakes/${id}`, payload)
}

export function deleteMistake(id: string) {
  return request.delete<unknown, unknown>(`/mistakes/${id}`)
}

export function reviewMistake(id: string, payload: ReviewPayload) {
  return request.post<unknown, Mistake>(`/mistakes/${id}/review`, payload)
}
