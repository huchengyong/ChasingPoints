export interface LlmCurlInput {
  endpoint_url: string
  model: string
  max_tokens: number | null | undefined
}

// 单引号包裹并转义内部单引号，保证预览可直接粘贴到 shell 执行。
const shellQuote = (value: string) => `'${value.replace(/'/g, `'\\''`)}'`

// 按已填写的配置生成一条等价的 curl 预览；API Key 永远用占位符，页面本身拿不到明文。
export const buildLlmCurl = ({ endpoint_url, model, max_tokens }: LlmCurlInput): string => {
  const body = JSON.stringify(
    {
      model: model.trim() || '<model>',
      max_tokens: typeof max_tokens === 'number' && Number.isFinite(max_tokens) && max_tokens > 0 ? max_tokens : '<max_tokens>',
      messages: [{ role: 'user', content: 'Say hi' }]
    },
    null,
    2
  )

  return [
    `curl -X POST ${shellQuote(endpoint_url.trim() || '<endpoint_url>')} \\`,
    `  -H 'Authorization: Bearer <API_KEY>' \\`,
    `  -H 'Content-Type: application/json' \\`,
    `  -d ${shellQuote(body)}`
  ].join('\n')
}
