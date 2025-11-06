import { apiFetch } from './client.ts'

type EmptyResponse = Record<string, never>

// --- IGA Session Types ---
export interface IgaSessionContextData {
  stakeholder_summary: string
  project_summary: string
  interview_goal: string
  interview_advice: string
}

export interface IgaSession {
  id: string
  user_id: string
  project_name: string
  context_data: IgaSessionContextData
  created_at: string
  updated_at: string
}

export interface CreateIgaSessionPayload {
  project_name: string
  context_data: IgaSessionContextData
}

export interface UpdateIgaSessionPayload {
  project_name?: string
  context_data?: IgaSessionContextData
}

// --- IGA Chat Types ---
export type IgaChatLogInstructionType = 'message' | 'refine-question' | 'refine-context'

export type IgaChatLogRole = 'user' | 'assistant'

export interface IgaChatLogMetadata {
  type: IgaChatLogInstructionType
}

export interface IgaChatLog {
  id: string
  iga_session_id: string
  role: IgaChatLogRole
  message_text: string
  metadata: IgaChatLogMetadata
  created_at: string
  is_optimistic: boolean
}

export interface SendChatPayload {
  message_text: string
  instruction: IgaChatLogInstructionType
}

// --- IGA Question Types ---
export type IgaScriptQuestionMetadata = Record<string, never>; // currently empty in Go, but keep for consistency

export interface IgaScriptQuestion {
  id: string
  iga_session_id: string
  readable_id: string
  question_text: string
  note_text: string
  sort_order: number
  metadata: IgaScriptQuestionMetadata
}

export interface CreateIgaScriptQuestionPayload {
  question_text: string
  note_text: string
  sort_order: number
}

export interface UpdateIgaScriptQuestionPayload {
  question_text?: string
  note_text?: string
  sort_order?: number
}

// --- IGA Session API Calls ---
export async function fetchIgaSessions(): Promise<IgaSession[]> {
  return apiFetch<IgaSession[]>('/iga', {
    method: 'GET',
  })
}

export async function createIgaSession(payload: CreateIgaSessionPayload): Promise<IgaSession> {
  return apiFetch<IgaSession>('/iga', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function fetchIgaSession(id: string): Promise<IgaSession> {
  return apiFetch<IgaSession>(`/iga/${id}`, {
    method: 'GET',
  })
}

export async function updateIgaSession(id: string, payload: UpdateIgaSessionPayload): Promise<IgaSession> {
  return apiFetch<IgaSession>(`/iga/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

// --- IGA Chat API Calls ---
export async function fetchIgaChatLog(sessionId: string): Promise<IgaChatLog[]> {
  return apiFetch<IgaChatLog[]>(`/iga/${sessionId}/chat`, {
    method: 'GET',
  })
}

export async function sendIgaChatMessage(sessionId: string, payload: SendChatPayload): Promise<IgaChatLog> {
  return apiFetch<IgaChatLog>(`/iga/${sessionId}/chat`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function retryIgaChatMessage(sessionId: string): Promise<IgaChatLog> {
  return apiFetch<IgaChatLog>(`/iga/${sessionId}/chat/retry`, {
    method: 'POST',
    body: JSON.stringify({}), // Go backend expects an empty JSON object for retryChatPayload
  })
}

// --- IGA Question API Calls ---
export async function fetchIgaQuestions(sessionId: string): Promise<IgaScriptQuestion[]> {
  return apiFetch<IgaScriptQuestion[]>(`/iga/${sessionId}/questions`, {
    method: 'GET',
  })
}

export async function createIgaQuestion(sessionId: string, payload: CreateIgaScriptQuestionPayload): Promise<IgaScriptQuestion> {
  return apiFetch<IgaScriptQuestion>(`/iga/${sessionId}/questions`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function updateIgaQuestion(sessionId: string, questionId: string, payload: UpdateIgaScriptQuestionPayload): Promise<IgaScriptQuestion> {
  return apiFetch<IgaScriptQuestion>(`/iga/${sessionId}/questions/${questionId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function deleteIgaQuestion(sessionId: string, questionId: string): Promise<EmptyResponse> {
  return apiFetch<EmptyResponse>(`/iga/${sessionId}/questions/${questionId}`, {
    method: 'DELETE',
  })
}

export async function reorderIgaQuestions(sessionId: string, orderedQuestionIds: string[]): Promise<EmptyResponse> {
  return apiFetch<EmptyResponse>(`/iga/${sessionId}/questions/reorder`, {
    method: 'PATCH',
    body: JSON.stringify({ordered_question_ids: orderedQuestionIds}),
  })
}
