export interface Credentials {
  email: string;
  password: string;
}

export interface AuthUser {
  id: number;
  email: string;
}

export interface AuthResponse {
  id: number;
  email: string;
  token: string;
}
