import { isParseResponse, type ParseResponse } from './floorplanUi'

export type ProjectSummary = {
  id: string
  name: string
  revision: number
  createdAt: string
  updatedAt: string
  sourceImageURL: string
}

export type ProjectDetail = ProjectSummary & {
  document: ParseResponse
  sourceImageContentType: string
  sourceImageSize: number
}

type ErrorEnvelope = { error?: { message?: unknown } | unknown }

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isProjectSummary(value: unknown): value is ProjectSummary {
  return isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.name === 'string' &&
    typeof value.revision === 'number' &&
    Number.isInteger(value.revision) &&
    typeof value.createdAt === 'string' &&
    typeof value.updatedAt === 'string' &&
    typeof value.sourceImageURL === 'string'
}

export function isProjectDetail(value: unknown): value is ProjectDetail {
  if (!isProjectSummary(value) || !isRecord(value)) return false
  const record: Record<string, unknown> = value
  return isParseResponse(record.document) &&
    typeof record.sourceImageContentType === 'string' &&
    typeof record.sourceImageSize === 'number' &&
    Number.isFinite(record.sourceImageSize) &&
    record.sourceImageSize > 0
}

async function responseError(response: Response): Promise<Error> {
  const text = await response.text()
  let body: ErrorEnvelope | null = null
  try {
    body = text ? JSON.parse(text) as ErrorEnvelope : null
  } catch {
    // Keep a useful error when an intermediary returned non-JSON.
  }
  const message = isRecord(body?.error) && typeof body.error.message === 'string'
    ? body.error.message
    : text.trim() || response.statusText || '请求失败'
  return new Error(`HTTP ${response.status}: ${message}`)
}

async function parseJSON(response: Response): Promise<unknown> {
  if (!response.ok) throw await responseError(response)
  return response.json()
}

export async function listProjects(signal?: AbortSignal): Promise<ProjectSummary[]> {
  const body = await parseJSON(await fetch('/api/projects', { signal }))
  if (!Array.isArray(body) || !body.every(isProjectSummary)) {
    throw new Error('服务返回的项目列表无效')
  }
  return body
}

export async function getProject(id: string, signal?: AbortSignal): Promise<ProjectDetail> {
  const body = await parseJSON(await fetch(`/api/projects/${encodeURIComponent(id)}`, { signal }))
  if (!isProjectDetail(body)) throw new Error('服务返回的项目文档无效')
  return body
}

export async function createProject(name: string, document: ParseResponse, sourceImage: File, signal?: AbortSignal): Promise<ProjectDetail> {
  const form = new FormData()
  form.append('name', name)
  form.append('document', JSON.stringify(document))
  form.append('source_image', sourceImage)
  const body = await parseJSON(await fetch('/api/projects', { method: 'POST', body: form, signal }))
  if (!isProjectDetail(body)) throw new Error('服务返回的已创建项目无效')
  return body
}

export async function updateProject(id: string, name: string, document: ParseResponse, expectedRevision: number, signal?: AbortSignal): Promise<ProjectDetail> {
  const body = await parseJSON(await fetch(`/api/projects/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, document, expectedRevision }),
    signal,
  }))
  if (!isProjectDetail(body)) throw new Error('服务返回的已保存项目无效')
  return body
}
