export interface ServerConfig {
  baseUrl: string;
  insecure?: boolean;
}

export interface Server {
  config: ServerConfig;
  token: string;
}
