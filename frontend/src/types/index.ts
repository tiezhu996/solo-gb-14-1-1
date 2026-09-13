// 与后端 DTO 对齐的全局类型定义
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface User {
  id: string
  username: string
  email: string
  nickname: string
  avatar: string
  role: string
  status: string
  points: number
  solved_count: number
  streak_days: number
  created_at: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface Chapter {
  title: string
  content: string
  duration: number
}

export interface Course {
  id: string
  title: string
  description: string
  cover: string
  markdown: string
  chapters: Chapter[]
  difficulty: string
  status: string
  author_id: string
  author_name: string
  created_at: string
  updated_at: string
}

export interface TestCase {
  input: string
  output: string
}

export interface Problem {
  id: string
  title: string
  description: string
  difficulty: string
  languages: string[]
  tags: string[]
  test_cases?: TestCase[]
  time_limit: number
  status: string
  points: number
  accepted_count: number
  submit_count: number
  created_by_name: string
  created_at: string
  updated_at: string
}

export interface JudgeResult {
  test_case_index: number
  input: string
  expected: string
  actual: string
  passed: boolean
  error_message: string
}

export interface Submission {
  id: string
  user_id: string
  username: string
  problem_id: string
  problem_title: string
  language: string
  code: string
  status: string
  score: number
  points_awarded: number
  runtime_ms: number
  results: JudgeResult[]
  error_message: string
  created_at: string
}

export interface Discussion {
  id: string
  problem_id: string
  user_id: string
  username: string
  nickname: string
  title: string
  content: string
  code: string
  language: string
  is_best: boolean
  vote_count: number
  status: string
  created_at: string
}

export interface Achievement {
  id: string
  code: string
  name: string
  description: string
  icon: string
  earned_at?: string
}

export interface LeaderboardEntry {
  rank: number
  user_id: string
  nickname: string
  username: string
  points: number
  solved: number
}

export interface Dashboard {
  total_learning_min: number
  completed_courses: number
  solved_count: number
  total_submissions: number
  streak_days: number
  points: number
  language_dist: Record<string, number>
  daily_activity: Record<string, number>
}

export interface AuditLog {
  id: string
  user_id: string
  username: string
  action: string
  entity: string
  entity_id: string
  method: string
  path: string
  status_code: number
  ip: string
  request_id: string
  detail: string
  created_at: string
}

// 错题本记录（与后端 MistakeResponse 对齐）
export interface Mistake {
  id: string
  problem_id?: string
  title: string
  knowledge_points: string[]
  error_reason: string
  review_note: string
  mastery: string
  next_review_at: string
  last_reviewed_at?: string
  review_count: number
  due: boolean
  created_at: string
  updated_at: string
}

// 错题列表查询参数：按题目关键词、知识点、掌握状态、到期筛选
export interface MistakeListParams {
  q?: string
  knowledge_point?: string
  mastery?: string
  due_only?: boolean
  page?: number
  page_size?: number
}

// 收录/修改错题的表单载荷（修改时未提供的字段保留原值，掌握状态不变）
export interface MistakePayload {
  title?: string
  problem_id?: string
  knowledge_points?: string[]
  error_reason?: string
  review_note?: string
  mastery?: string
  next_review_at?: string
}

// 完成一次复习的载荷
export interface ReviewPayload {
  mastery: string
  next_review_at?: string
}
