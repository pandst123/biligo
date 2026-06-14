export interface Health {
  status: string
  database: string
  time: string
}

export interface SessionSummary {
  status: string
  message: string
  accountCount: number
  configuredAccounts: number
  verifiedAccounts: number
}

export interface Account {
  id: number
  name: string
  cookiePreview: string
  hasCookie: boolean
  status: string
  note: string
  createdAt: string
  updatedAt: string
}

export interface AccountInput {
  name: string
  cookie: string
  note: string
}

export interface AccountCookieResponse {
  accountId: number
  cookie: string
  cookiePreview: string
}

export interface QRLoginStartResponse {
  ok: boolean
  loginUrl: string
  qrcodeKey: string
  qrImageDataUrl: string
  expiresInSeconds: number
  nextAction: string
}

export interface QRLoginPollInput {
  qrcodeKey: string
  accountName: string
  note: string
}

export interface QRLoginPollResponse {
  ok: boolean
  status: string
  message: string
  code?: number
  username?: string
  account?: Account
}

export interface CookieLoginResponse {
  ok: boolean
  loggedIn: boolean
  username?: string
  message: string
  account?: Account
}

export interface AccountVerifyResponse {
  ok: boolean
  loggedIn: boolean
  accountId: number
  username?: string
  message: string
  account?: Account
}

export interface TicketProjectHistory {
  projectId: number
  projectName: string
  venueName: string
  venueAddress: string
  startAt: string
  endAt: string
  updatedAt: string
}

export interface TicketProjectFetchInput {
  projectInput: string
  accountId: number
}

export interface TicketAccountContextInput {
  projectInput: string
  accountId: number
}

export interface TicketAccountContext {
  projectId: number
  username: string
  phone: string
  buyers: TicketBuyer[]
  addresses: TicketAddress[]
}

export interface TicketProject {
  projectId: number
  projectName: string
  projectUrl: string
  username: string
  phone: string
  venueName: string
  venueAddress: string
  startAt: string
  endAt: string
  isHotProject: boolean
  hasETicket: boolean
  salesDates: string[]
  ticketOptions: TicketOption[]
  buyers: TicketBuyer[]
  addresses: TicketAddress[]
}

export interface TicketOption {
  value: string
  display: string
  projectId: number
  screenId: number
  skuId: number
  screenName: string
  ticketLevel: string
  price: number
  priceText: string
  saleStatus: string
  saleStart: string
  isHotProject: boolean
  linkId?: number
}

export interface TicketBuyer {
  id?: number
  name: string
  personalId: string
  tel?: string
  raw?: Record<string, unknown>
}

export interface TicketAddress {
  id: number
  name: string
  phone: string
  prov: string
  city: string
  area: string
  addr: string
  fullAddress: string
  raw?: Record<string, unknown>
}

export interface Task {
  id: number
  name: string
  accountId: number
  accountName: string
  projectId: number
  projectName: string
  screenId: number
  skuId: number
  sessionName: string
  ticketLevel: string
  ticketDisplay: string
  ticketPrice: number
  saleStart: string
  saleStatus: string
  linkId: number
  isHotProject: boolean
  orderType: number
  payMoney: number
  buyerInfo: TicketBuyer[]
  buyer: string
  tel: string
  deliverInfo?: TicketAddress
  phone: string
  orderId: string
  paymentUrl: string
  paymentQrImageDataUrl: string
  lastCheckedAt: string
  quantity: number
  startAt: string
  endAt: string
  pollIntervalSeconds: number
  status: string
  lastMessage: string
  createdAt: string
  updatedAt: string
}

export interface TaskInput {
  name: string
  accountId: number
  projectId: number
  projectName: string
  screenId: number
  skuId: number
  sessionName: string
  ticketLevel: string
  ticketDisplay: string
  ticketPrice: number
  saleStart: string
  saleStatus: string
  linkId: number
  isHotProject: boolean
  orderType: number
  payMoney: number
  buyerInfo: TicketBuyer[]
  buyer: string
  tel: string
  deliverInfo?: TicketAddress
  phone: string
  quantity: number
  startAt: string
  endAt: string
  pollIntervalSeconds: number
}

export interface TaskLog {
  id: number
  taskId: number
  level: string
  message: string
  createdAt: string
}

export interface EventSnapshot {
  tasks: Task[]
  logs: TaskLog[]
}

export const API_BASE = import.meta.env.VITE_API_BASE ?? ''

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  })

  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`
    try {
      const data = await response.json()
      if (data?.error) {
        message = data.error
      }
    } catch {
      // Keep the HTTP status message.
    }
    throw new Error(message)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export const api = {
  health: () => request<Health>('/api/health'),
  session: () => request<SessionSummary>('/api/auth/session'),
  startQRLogin: () =>
    request<QRLoginStartResponse>('/api/auth/qr/start', { method: 'POST' }),
  pollQRLogin: (payload: QRLoginPollInput) =>
    request<QRLoginPollResponse>('/api/auth/qr/poll', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  loginWithCookie: (payload: AccountInput) =>
    request<CookieLoginResponse>('/api/auth/cookie-login', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  listAccounts: () => request<Account[]>('/api/accounts'),
  createAccount: (payload: AccountInput) =>
    request<Account>('/api/accounts', { method: 'POST', body: JSON.stringify(payload) }),
  updateAccount: (id: number, payload: AccountInput) =>
    request<Account>(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  getAccountCookie: (id: number) => request<AccountCookieResponse>(`/api/accounts/${id}/cookie`),
  verifyAccount: (id: number) =>
    request<AccountVerifyResponse>(`/api/accounts/${id}/verify`, { method: 'POST' }),
  deleteAccount: (id: number) => request<void>(`/api/accounts/${id}`, { method: 'DELETE' }),

  listTicketProjectHistory: () => request<TicketProjectHistory[]>('/api/ticket-projects/history'),
  fetchTicketProject: (payload: TicketProjectFetchInput) =>
    request<TicketProject>('/api/ticket-projects/fetch', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  fetchTicketAccountContext: (payload: TicketAccountContextInput) =>
    request<TicketAccountContext>('/api/ticket-projects/account-context', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  listTasks: () => request<Task[]>('/api/tasks'),
  createTask: (payload: TaskInput) =>
    request<Task>('/api/tasks', { method: 'POST', body: JSON.stringify(payload) }),
  updateTask: (id: number, payload: TaskInput) =>
    request<Task>(`/api/tasks/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteTask: (id: number) => request<void>(`/api/tasks/${id}`, { method: 'DELETE' }),
  dispatchTask: (id: number) => request<Task>(`/api/tasks/${id}/dispatch`, { method: 'POST' }),
  pauseTask: (id: number) => request<Task>(`/api/tasks/${id}/pause`, { method: 'POST' }),
  listLogs: (taskId?: number) =>
    request<TaskLog[]>(taskId ? `/api/logs?task_id=${taskId}` : '/api/logs'),
}
