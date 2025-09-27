import { apiFetch } from './client.ts'

interface RegisterPayload {
  email: string;
  first_name: string;
  last_name: string;
  plain_password: string
}

interface RegisterResponse {
  id: string;
}

type EmptyResponse = Record<string, never>;

export async function registerUser(payload: RegisterPayload): Promise<RegisterResponse> {
  return apiFetch<RegisterResponse>('/user/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function loginUser(email: string, password: string): Promise<EmptyResponse> {
  return apiFetch<EmptyResponse>('/user/login', {
    method: 'POST',
    body: JSON.stringify({email, password}),
  });
}

export async function logoutUser(): Promise<EmptyResponse> {
  return apiFetch<EmptyResponse>('/user/logout', {
    method: 'POST',
    body: JSON.stringify({}),
  });
}
