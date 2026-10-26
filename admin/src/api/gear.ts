import request from '@/utils/request'

export interface GearBrand {
  id: number
  name: string
  aliases: string
  sort: number
  status: number // 1=启用 0=停用
  product_count: number
}

export interface GearBrandListParams {
  keyword?: string
  page?: number
  page_size?: number
}

export interface GearBrandListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: GearBrand[]
}

export interface GearBrandSaveParams {
  id?: number
  name: string
  aliases?: string
  sort?: number
  status?: number
}

export interface GearSaveResult {
  code: number
  success: boolean
  message: string
  id?: number
}

export interface GearProduct {
  id: number
  category: string // cue/break_jump/tip/chalk
  brand_id: number
  brand_name: string
  name: string
  house: boolean
  cue_style: string
  roles: string[]
  length_mm: number
  weight_g: number
  tip_diameter_mm: number
  image_path: string
  sort: number
  status: number
}

export interface GearProductListParams {
  category?: string
  brand_id?: number
  keyword?: string
  status?: number // -1=全部 1=启用 0=停用
  page?: number
  page_size?: number
}

export interface GearProductListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: GearProduct[]
}

export interface GearProductSaveParams {
  id?: number
  category: string
  brand_id: number
  name: string
  cue_style?: string
  roles?: string[]
  length_mm?: number
  weight_g?: number
  tip_diameter_mm?: number
  image_path?: string
  sort?: number
  status?: number
}

export interface GearPendingGroup {
  custom_key: string
  category: string
  brand_name: string
  model_name: string
  cue_style: string
  roles: string[]
  count: number
  last_at: string
}

export interface GearPendingListParams {
  category?: string
  page?: number
  page_size?: number
}

export interface GearPendingListResult {
  code: number
  success: boolean
  message: string
  total: number
  list: GearPendingGroup[]
}

export interface GearPendingLinkParams {
  custom_key: string
  product_id: number
}

export interface GearPendingLinkResult {
  code: number
  success: boolean
  message: string
  updated: number
  remaining: number
}

// 品牌列表
export const getGearBrandList = (params: GearBrandListParams = {}): Promise<GearBrandListResult> => {
  return request.get('/api/admin/gear/brands', { params })
}

// 新增或修改品牌（id 为空表示新增）
export const saveGearBrand = (data: GearBrandSaveParams): Promise<GearSaveResult> => {
  return request.post('/api/admin/gear/brand/save', data)
}

// 型号列表
export const getGearProductList = (params: GearProductListParams = {}): Promise<GearProductListResult> => {
  return request.get('/api/admin/gear/products', { params })
}

// 新增或修改型号（id 为空表示新增）
export const saveGearProduct = (data: GearProductSaveParams): Promise<GearSaveResult> => {
  return request.post('/api/admin/gear/product/save', data)
}

// 待收录的未收录型号分组，按登记数倒序
export const getGearPendingList = (params: GearPendingListParams = {}): Promise<GearPendingListResult> => {
  return request.get('/api/admin/gear/pending', { params })
}

// 把一组未收录登记关联到目录型号；每次最多处理一批，remaining 大于 0 时可以再次执行
export const linkGearPending = (data: GearPendingLinkParams): Promise<GearPendingLinkResult> => {
  return request.post('/api/admin/gear/pending/link', data)
}
