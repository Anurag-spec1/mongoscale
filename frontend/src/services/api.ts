import { useAuthStore } from '../store/auth';

const BASE_URL = '/api/v1';

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

async function request<T>(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const token = useAuthStore.getState().token;
  const headers: Record<string, string> = {};
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let response: Response;
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw error;
    }
    throw new ApiError(0, 'network_error', 'The API is unreachable');
  }
  let payload: Envelope<T> | null = null;
  try {
    payload = (await response.json()) as Envelope<T>;
  } catch {
    payload = null;
  }
  if (!response.ok || !payload?.success) {
    if (response.status === 401 && token) {
      useAuthStore.getState().logout();
    }
    throw new ApiError(
      response.status,
      payload?.error?.code ?? 'request_failed',
      payload?.error?.message ?? `Request failed with status ${response.status}`,
    );
  }
  return payload.data as T;
}

export const api = {
  get: <T>(path: string, signal?: AbortSignal) => request<T>('GET', path, undefined, signal),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {}),
  postEmpty: <T>(path: string) => request<T>('POST', path),
  patch: <T>(path: string, body: unknown) => request<T>('PATCH', path, body),
};

export function query(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') {
      search.set(key, String(value));
    }
  });
  const text = search.toString();
  return text ? `?${text}` : '';
}
