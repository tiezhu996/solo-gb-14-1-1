// 错题本：收录错题、记录错误原因与复盘结论，按题目/知识点/掌握状态查询，
// 设置下次复习日期，到期在列表中提醒；修改或移除记录不影响当前掌握状态。
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMistakeStore } from '../../stores/mistakeStore'
import DataTable, { type Column } from '../../components/DataTable'
import Pagination from '../../components/Pagination'
import ConfirmDialog from '../../components/ConfirmDialog'
import StatusBadge from '../../components/StatusBadge'
import { usePagination } from '../../hooks/usePagination'
import { MASTERY, MASTERY_LABELS } from '../../constants'
import type { Mistake } from '../../types'

// 表单状态：知识点用逗号分隔的文本编辑
interface MistakeForm {
  title: string
  knowledgePointsText: string
  error_reason: string
  review_note: string
  mastery: string
  next_review_at: string
}

const emptyForm: MistakeForm = {
  title: '',
  knowledgePointsText: '',
  error_reason: '',
  review_note: '',
  mastery: MASTERY.unmastered,
  next_review_at: '',
}

function todayStr(): string {
  return new Date().toLocaleDateString('sv-SE')
}

function addDaysStr(days: number): string {
  const d = new Date()
  d.setDate(d.getDate() + days)
  return d.toLocaleDateString('sv-SE')
}

function parsePoints(text: string): string[] {
  return text
    .split(/[,，、\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

export default function Mistakes() {
  const {
    mistakes, total, due, knowledgePoints, loading,
    fetchMistakes, fetchDue, fetchKnowledgePoints,
    createMistake, updateMistake, deleteMistake, reviewMistake,
  } = useMistakeStore()
  const { page, pageSize, setTotal, onPageChange } = usePagination(10)

  // 筛选条件：题目关键词 / 知识点 / 掌握状态 / 只看到期
  const [q, setQ] = useState('')
  const [appliedQ, setAppliedQ] = useState('')
  const [knowledgePoint, setKnowledgePoint] = useState('')
  const [mastery, setMastery] = useState('')
  const [dueOnly, setDueOnly] = useState(false)
  const [error, setError] = useState('')

  // 弹窗状态：收录/编辑表单、复习、删除确认
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Mistake | null>(null)
  const [form, setForm] = useState<MistakeForm>(emptyForm)
  const [reviewing, setReviewing] = useState<Mistake | null>(null)
  const [reviewForm, setReviewForm] = useState({ mastery: '', next_review_at: '' })
  const [deleting, setDeleting] = useState<Mistake | null>(null)

  const load = useCallback(async () => {
    await fetchMistakes({
      page,
      page_size: pageSize,
      q: appliedQ || undefined,
      knowledge_point: knowledgePoint || undefined,
      mastery: mastery || undefined,
      due_only: dueOnly || undefined,
    })
  }, [fetchMistakes, page, pageSize, appliedQ, knowledgePoint, mastery, dueOnly])

  useEffect(() => {
    load().then(() => {})
  }, [load])

  useEffect(() => {
    setTotal(total)
  }, [total, setTotal])

  useEffect(() => {
    fetchDue().then(() => {})
    fetchKnowledgePoints().then(() => {})
  }, [fetchDue, fetchKnowledgePoints])

  // 操作后刷新列表、到期提醒与知识点清单
  const refreshAll = async () => {
    await Promise.all([load(), fetchDue(), fetchKnowledgePoints()])
  }

  const run = async (fn: () => Promise<unknown>) => {
    setError('')
    try {
      await fn()
      await refreshAll()
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    }
  }

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm, next_review_at: addDaysStr(1) })
    setFormOpen(true)
  }

  // 编辑时预填当前值：掌握状态默认保持现状，不修改即保留
  const openEdit = (m: Mistake) => {
    setEditing(m)
    setForm({
      title: m.title,
      knowledgePointsText: m.knowledge_points.join('，'),
      error_reason: m.error_reason,
      review_note: m.review_note,
      mastery: m.mastery,
      next_review_at: m.next_review_at,
    })
    setFormOpen(true)
  }

  const submitForm = () => {
    if (!form.title.trim()) {
      setError('题目不能为空')
      return
    }
    const payload = {
      title: form.title.trim(),
      knowledge_points: parsePoints(form.knowledgePointsText),
      error_reason: form.error_reason,
      review_note: form.review_note,
      mastery: form.mastery,
      next_review_at: form.next_review_at || undefined,
    }
    setFormOpen(false)
    run(async () => {
      if (editing) {
        await updateMistake(editing.id, payload)
      } else {
        await createMistake(payload)
      }
    }).then(() => {})
  }

  const openReview = (m: Mistake) => {
    setReviewing(m)
    setReviewForm({ mastery: m.mastery, next_review_at: '' })
  }

  const submitReview = () => {
    if (!reviewing) return
    const id = reviewing.id
    setReviewing(null)
    run(async () => {
      await reviewMistake(id, {
        mastery: reviewForm.mastery,
        next_review_at: reviewForm.next_review_at || undefined,
      })
    }).then(() => {})
  }

  const confirmDelete = () => {
    if (!deleting) return
    const id = deleting.id
    setDeleting(null)
    run(async () => {
      await deleteMistake(id)
    }).then(() => {})
  }

  const columns: Column<Mistake>[] = [
    {
      key: 'title',
      title: '题目',
      render: (m) => (
        <div className="max-w-xs">
          <div className="flex items-center gap-1.5 font-medium text-gray-800">
            <span className="truncate">{m.title}</span>
            {m.problem_id && (
              <Link to={`/problems/${m.problem_id}`} className="shrink-0 text-brand-600 hover:underline" title="查看原题">
                ↗
              </Link>
            )}
          </div>
          {m.error_reason && <p className="mt-0.5 truncate text-xs text-gray-400">错因：{m.error_reason}</p>}
        </div>
      ),
    },
    {
      key: 'knowledge_points',
      title: '知识点',
      render: (m) => (
        <div className="flex max-w-[180px] flex-wrap gap-1">
          {m.knowledge_points.length === 0 && <span className="text-gray-300">-</span>}
          {m.knowledge_points.map((kp) => (
            <span key={kp} className="rounded-full bg-sky-50 px-2 py-0.5 text-xs text-sky-700">
              {kp}
            </span>
          ))}
        </div>
      ),
    },
    {
      key: 'mastery',
      title: '掌握状态',
      render: (m) => <StatusBadge value={m.mastery} kind="mastery" />,
    },
    {
      key: 'next_review_at',
      title: '下次复习',
      render: (m) => (
        <div className="whitespace-nowrap">
          <span className={m.due ? 'font-semibold text-rose-600' : 'text-gray-600'}>{m.next_review_at}</span>
          {m.due && (
            <span className="ml-1.5 rounded-full bg-rose-100 px-2 py-0.5 text-xs font-medium text-rose-700">
              已到期
            </span>
          )}
        </div>
      ),
    },
    {
      key: 'review_count',
      title: '复习次数',
      render: (m) => <span className="text-gray-600">{m.review_count}</span>,
    },
    {
      key: 'actions',
      title: '操作',
      render: (m) => (
        <div className="flex gap-3 whitespace-nowrap text-sm">
          <button onClick={() => openReview(m)} className="text-emerald-600 hover:underline">
            复习
          </button>
          <button onClick={() => openEdit(m)} className="text-brand-600 hover:underline">
            编辑
          </button>
          <button onClick={() => setDeleting(m)} className="text-rose-600 hover:underline">
            移除
          </button>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">错题本</h1>
          <p className="mt-1 text-sm text-gray-500">收录做错的题目，记录错因与复盘结论，按掌握状态排期复习</p>
        </div>
        <button
          onClick={openCreate}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
        >
          + 收录错题
        </button>
      </div>

      {/* 到期复习提醒 */}
      {due.length > 0 && (
        <div className="flex items-center justify-between rounded-xl border border-rose-200 bg-rose-50 px-4 py-3">
          <p className="text-sm text-rose-700">
            🔔 你有 <span className="font-bold">{due.length}</span> 条错题已到复习时间，记得及时复盘巩固
          </p>
          <button
            onClick={() => {
              setDueOnly(true)
              onPageChange(1)
            }}
            className="rounded-lg bg-rose-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-rose-700"
          >
            只看到期
          </button>
        </div>
      )}

      {error && (
        <div className="flex items-center justify-between rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700">
          <span>{error}</span>
          <button onClick={() => setError('')} className="text-amber-500 hover:text-amber-700">
            ✕
          </button>
        </div>
      )}

      {/* 筛选：题目关键词 / 知识点 / 掌握状态 / 到期 */}
      <div className="flex flex-wrap items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
        <div className="flex gap-2">
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                setAppliedQ(q)
                onPageChange(1)
              }
            }}
            placeholder="搜索题目关键词..."
            className="w-52 rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
          <button
            onClick={() => {
              setAppliedQ(q)
              onPageChange(1)
            }}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-100"
          >
            搜索
          </button>
        </div>
        <select
          value={knowledgePoint}
          onChange={(e) => {
            setKnowledgePoint(e.target.value)
            onPageChange(1)
          }}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
        >
          <option value="">全部知识点</option>
          {knowledgePoints.map((kp) => (
            <option key={kp} value={kp}>
              {kp}
            </option>
          ))}
        </select>
        <div className="flex gap-1.5">
          {['', MASTERY.unmastered, MASTERY.learning, MASTERY.mastered].map((v) => (
            <button
              key={v || 'all'}
              onClick={() => {
                setMastery(v)
                onPageChange(1)
              }}
              className={`rounded-lg px-3 py-1.5 text-sm ${mastery === v ? 'bg-brand-600 text-white' : 'border border-gray-300 hover:bg-gray-100'}`}
            >
              {v === '' ? '全部' : MASTERY_LABELS[v]}
            </button>
          ))}
        </div>
        <label className="flex cursor-pointer items-center gap-1.5 text-sm text-gray-600">
          <input
            type="checkbox"
            checked={dueOnly}
            onChange={(e) => {
              setDueOnly(e.target.checked)
              onPageChange(1)
            }}
            className="h-4 w-4"
          />
          只看到期
        </label>
      </div>

      <DataTable columns={columns} rows={mistakes} rowKey={(m) => m.id} loading={loading} />
      <Pagination page={page} pageSize={pageSize} total={total} onChange={onPageChange} />

      {/* 收录/编辑弹窗 */}
      {formOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setFormOpen(false)}>
          <div
            className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-lg font-bold text-gray-800">{editing ? '编辑错题' : '收录错题'}</h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">题目 *</label>
                <input
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  placeholder="题目标题或简述"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">知识点（逗号分隔）</label>
                <input
                  value={form.knowledgePointsText}
                  onChange={(e) => setForm({ ...form, knowledgePointsText: e.target.value })}
                  placeholder="如：动态规划，背包问题"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">错误原因</label>
                <textarea
                  value={form.error_reason}
                  onChange={(e) => setForm({ ...form, error_reason: e.target.value })}
                  placeholder="当时为什么做错了？"
                  rows={2}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">复盘结论</label>
                <textarea
                  value={form.review_note}
                  onChange={(e) => setForm({ ...form, review_note: e.target.value })}
                  placeholder="正确思路、易错点、下次注意什么"
                  rows={2}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="mb-1 block text-sm font-medium text-gray-700">掌握状态</label>
                  <select
                    value={form.mastery}
                    onChange={(e) => setForm({ ...form, mastery: e.target.value })}
                    className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                  >
                    {Object.values(MASTERY).map((v) => (
                      <option key={v} value={v}>
                        {MASTERY_LABELS[v]}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex-1">
                  <label className="mb-1 block text-sm font-medium text-gray-700">下次复习日期</label>
                  <input
                    type="date"
                    value={form.next_review_at}
                    min={todayStr()}
                    onChange={(e) => setForm({ ...form, next_review_at: e.target.value })}
                    className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                  />
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setFormOpen(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100">
                取消
              </button>
              <button
                onClick={submitForm}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
              >
                {editing ? '保存修改' : '收录'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 复习弹窗：更新掌握状态并排期下次复习 */}
      {reviewing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setReviewing(null)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-lg font-bold text-gray-800">完成复习</h3>
            <p className="mt-1 truncate text-sm text-gray-500">{reviewing.title}</p>
            {reviewing.review_note && (
              <p className="mt-2 rounded-lg bg-gray-50 p-3 text-xs text-gray-600">复盘结论：{reviewing.review_note}</p>
            )}
            <div className="mt-4">
              <label className="mb-1 block text-sm font-medium text-gray-700">复习后掌握状态</label>
              <div className="flex gap-2">
                {Object.values(MASTERY).map((v) => (
                  <button
                    key={v}
                    onClick={() => setReviewForm({ ...reviewForm, mastery: v })}
                    className={`flex-1 rounded-lg px-3 py-2 text-sm ${reviewForm.mastery === v ? 'bg-brand-600 text-white' : 'border border-gray-300 hover:bg-gray-100'}`}
                  >
                    {MASTERY_LABELS[v]}
                  </button>
                ))}
              </div>
            </div>
            <div className="mt-4">
              <label className="mb-1 block text-sm font-medium text-gray-700">下次复习日期（留空按掌握状态自动排期）</label>
              <input
                type="date"
                value={reviewForm.next_review_at}
                min={todayStr()}
                onChange={(e) => setReviewForm({ ...reviewForm, next_review_at: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              />
              <div className="mt-2 flex gap-2">
                {[
                  { label: '明天', days: 1 },
                  { label: '3 天后', days: 3 },
                  { label: '7 天后', days: 7 },
                ].map((q) => (
                  <button
                    key={q.days}
                    onClick={() => setReviewForm({ ...reviewForm, next_review_at: addDaysStr(q.days) })}
                    className="rounded-lg border border-gray-300 px-3 py-1 text-xs hover:bg-gray-100"
                  >
                    {q.label}
                  </button>
                ))}
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setReviewing(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100">
                取消
              </button>
              <button
                onClick={submitReview}
                className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-700"
              >
                完成复习
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 移除确认 */}
      <ConfirmDialog
        open={!!deleting}
        title="移除错题"
        message={`确定把「${deleting?.title}」从错题本移除吗？该操作不可恢复，其他记录的掌握状态不受影响。`}
        confirmText="移除"
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </div>
  )
}
