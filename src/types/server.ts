export interface Server {
  baseUrl: string;
  token: string;
  insecure?: boolean;
}

export interface ServerConfig {
  baseUrl: string;
  insecure?: boolean;
}