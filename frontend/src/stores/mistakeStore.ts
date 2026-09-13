// 错题本状态
import { create } from 'zustand'
import type { Mistake, MistakeListParams, MistakePayload, ReviewPayload } from '../types'
import * as mistakeApi from '../api/mistake'

interface MistakeState {
  mistakes: Mistake[]
  total: number
  due: Mistake[]
  knowledgePoints: string[]
  loading: boolean
  fetchMistakes: (params?: MistakeListParams) => Promise<void>
  fetchDue: () => Promise<void>
  fetchKnowledgePoints: () => Promise<void>
  createMistake: (payload: MistakePayload) => Promise<Mistake>
  updateMistake: (id: string, payload: MistakePayload) => Promise<Mistake>
  deleteMistake: (id: string) => Promise<void>
  reviewMistake: (id: string, payload: ReviewPayload) => Promise<Mistake>
}

export const useMistakeStore = create<MistakeState>((set) => ({
  mistakes: [],
  total: 0,
  due: [],
  knowledgePoints: [],
  loading: false,
  fetchMistakes: async (params) => {
    set({ loading: true })
    try {
      const data = await mistakeApi.listMistakes(params || {})
      set({ mistakes: data.list, total: data.total })
    } finally {
      set({ loading: false })
    }
  },
  fetchDue: async () => {
    const data = await mistakeApi.listDueMistakes()
    set({ due: data.list })
  },
  fetchKnowledgePoints: async () => {
    const points = await mistakeApi.getKnowledgePoints()
    set({ knowledgePoints: points })
  },
  createMistake: async (payload) => {
    return await mistakeApi.createMistake(payload)
  },
  updateMistake: async (id, payload) => {
    return await mistakeApi.updateMistake(id, payload)
  },
  deleteMistake: async (id) => {
    await mistakeApi.deleteMistake(id)
  },
  reviewMistake: async (id, payload) => {
    return await mistakeApi.reviewMistake(id, payload)
  },
}))
