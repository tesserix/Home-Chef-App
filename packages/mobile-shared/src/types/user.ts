// Types matching Go API response shapes exactly
// Source: apps/api/handlers/auth.go AuthResponse, UserResponse

export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  role: 'customer' | 'chef' | 'driver' | 'admin';
  avatar: string | null;
  fcmToken: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface AuthResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  firstName: string;
  lastName: string;
  phone?: string;
}

export interface OAuthLoginRequest {
  provider: 'google' | 'apple';
  token: string; // ID token from native SDK
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  token: string;
  password: string;
}

export interface ApiError {
  error: string;
  message: string;
}
