/// <reference types="vite/client" />
declare const GITHUB_RUNTIME_PERMANENT_NAME: string
declare const BASE_KV_SERVICE_URL: string

// Enable JSON imports
declare module "*.json" {
  const value: Record<string, unknown>;
  export default value;
}