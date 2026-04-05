// Typed API functions for all auth endpoints
// Source: apps/api/routes/routes.go + apps/api/handlers/auth.go

import { AxiosInstance } from 'axios';
import {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  OAuthLoginRequest,
  ForgotPasswordRequest,
  ResetPasswordRequest,
} from '../types/user';

export async function loginWithEmail(
  client: AxiosInstance,
  req: LoginRequest
): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/login', req);
  return res.data;
}

export async function registerUser(
  client: AxiosInstance,
  req: RegisterRequest
): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/register', req);
  return res.data;
}

export async function oauthLogin(
  client: AxiosInstance,
  req: OAuthLoginRequest
): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/oauth', req);
  return res.data;
}

export async function refreshAuthToken(
  client: AxiosInstance,
  refreshToken: string
): Promise<{ accessToken: string; refreshToken: string }> {
  const res = await client.post('/auth/refresh', { refreshToken });
  return res.data;
}

export async function forgotPassword(
  client: AxiosInstance,
  req: ForgotPasswordRequest
): Promise<{ message: string }> {
  const res = await client.post('/auth/forgot-password', req);
  return res.data;
}

export async function resetPassword(
  client: AxiosInstance,
  req: ResetPasswordRequest
): Promise<{ message: string }> {
  const res = await client.post('/auth/reset-password', req);
  return res.data;
}

export async function logoutUser(
  client: AxiosInstance,
  refreshToken: string
): Promise<void> {
  await client.post('/auth/logout', { refreshToken });
}
