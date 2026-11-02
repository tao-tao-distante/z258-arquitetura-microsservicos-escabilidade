export interface Credentials {
  email: string;
  password: string;
}

export interface AuthUser {
  id: number;
  email: string;
  name: string;
}

export interface AuthResponse {
  id: number;
  email: string;
  name: string;
  token: string;
}

export interface ProfileUpdate {
  name: string;
}

export interface PasswordChange {
  currentPassword: string;
  newPassword: string;
}
