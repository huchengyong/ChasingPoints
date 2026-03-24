export const unwrapBusinessData = <T>(data: T): T extends { data: infer U } ? U : T => {
  if (data && typeof data === 'object' && Object.prototype.hasOwnProperty.call(data, 'data')) {
    return (data as unknown as { data: unknown }).data as T extends { data: infer U } ? U : T
  }

  return data as T extends { data: infer U } ? U : T
}
