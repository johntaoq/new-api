export function getStaticHomePath(hostname: string): string | null {
  return hostname === 'api.99off.net' ? '/' : null
}
