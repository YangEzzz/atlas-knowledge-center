export function canManage(roles?: string[]) {
  return roles?.some((role) => role === "admin" || role === "owner") ?? false;
}
