import { readFileSync } from 'node:fs'

// pages.json 用整行 // 注释做条件编译（如 // #ifdef MP-WEIXIN），去掉这些行后即可按 JSON 解析；
// 解析结果同时包含各平台专属的键，适合静态断言。
const stripLineComments = (source) => source
  .split('\n')
  .filter((line) => !line.trimStart().startsWith('//'))
  .join('\n')

export const readPagesJson = (url = new URL('../../pages.json', import.meta.url)) => (
  JSON.parse(stripLineComments(readFileSync(url, 'utf8')))
)
