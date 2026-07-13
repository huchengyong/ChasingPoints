export function resolvePublicValue(name: string, fallback: string): string {
  return process.env[name] || fallback
}
