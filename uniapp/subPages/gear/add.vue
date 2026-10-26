<template>
	<view class="gear-add-page" :class="{ 'dark-mode': isDarkMode }">
		<view v-if="!isPick" class="category-tabs">
			<view
				v-for="tab in createTabs"
				:key="tab.value"
				class="category-tab"
				:class="{ active: category === tab.value }"
				@tap="switchCategory(tab.value)"
			>
				<text>{{ tab.label }}</text>
			</view>
		</view>
		<view v-else class="pick-banner">
			<text class="pick-title">选择{{ categoryLabel }}</text>
			<text class="pick-clear" @tap="pickNone">不使用</text>
		</view>

		<view class="search-box">
			<uni-icons type="search" size="18" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
			<input
				class="search-input"
				:value="keyword"
				confirm-type="search"
				placeholder="搜索品牌或型号"
				placeholder-class="search-placeholder"
				@input="onKeywordInput"
				@confirm="applyKeyword"
			/>
			<text v-if="keyword" class="search-clear" @tap="clearKeyword">清除</text>
		</view>

		<scroll-view v-if="brands.length" scroll-x class="brand-scroll" :show-scrollbar="false">
			<view class="brand-row">
				<text class="brand-chip" :class="{ active: !brandId }" @tap="selectBrand(0)">全部品牌</text>
				<text
					v-for="brand in brands"
					:key="brand.id"
					class="brand-chip"
					:class="{ active: brandId === brand.id }"
					@tap="selectBrand(brand.id)"
				>{{ brand.name }}</text>
			</view>
		</scroll-view>

		<view v-if="houseProduct && !keyword && !brandId" class="house-row" @tap="selectProduct(houseProduct)">
			<view class="house-image"><gearImage :subject="houseProduct" fill /></view>
			<view class="house-copy">
				<text class="house-title">{{ houseProduct.name }}</text>
				<text class="house-desc">{{ houseHint }}</text>
			</view>
			<uni-icons type="right" size="16" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
		</view>

		<view v-if="listLoading && !products.length" class="state-block">
			<uni-icons type="spinner-cycle" size="32" color="#E0AE12"></uni-icons>
			<text class="state-title">加载型号中...</text>
		</view>
		<view v-else-if="listError && !products.length" class="state-block">
			<uni-icons type="info" size="44" :color="isDarkMode ? '#d7c89b' : '#9A8C67'"></uni-icons>
			<text class="state-title">型号加载失败</text>
			<text class="state-desc">{{ listError }}</text>
			<button class="primary-btn" @tap="reloadList">重试</button>
		</view>
		<view v-else-if="!products.length" class="empty-tip">
			<text class="empty-title">没有找到匹配的型号</text>
			<text v-if="!isPick" class="empty-desc">可以换个关键词，或在下方登记未收录型号。</text>
		</view>
		<view v-else class="product-list">
			<view v-for="product in products" :key="product.id" class="product-row" @tap="selectProduct(product)">
				<view class="product-image"><gearImage :subject="product" fill /></view>
				<view class="product-copy">
					<text class="product-title">{{ productTitle(product) }}</text>
					<text v-if="productSubtitle(product)" class="product-subtitle">{{ productSubtitle(product) }}</text>
				</view>
				<uni-icons type="right" size="16" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
			</view>
			<text v-if="listLoading" class="list-tip">加载中...</text>
			<text v-else-if="listError" class="list-tip list-tip--action" @tap="loadMore">加载失败，点击重试</text>
			<text v-else-if="!hasMore" class="list-tip">已经到底了</text>
		</view>

		<template v-if="!isPick">
			<view class="unlisted-entry" @tap="toggleUnlisted">
				<view class="unlisted-copy">
					<text class="unlisted-title">找不到我的型号？</text>
					<text class="unlisted-desc">先登记为「未收录型号」，我们收录后会自动更新</text>
				</view>
				<uni-icons :type="showUnlisted ? 'up' : 'down'" size="16" :color="isDarkMode ? '#9F926E' : '#9A8C67'"></uni-icons>
			</view>

			<view v-if="showUnlisted" class="card unlisted-form">
				<text class="form-label">品牌</text>
				<view class="chip-row">
					<text
						v-for="brand in brands"
						:key="brand.id"
						class="chip"
						:class="{ active: form.brandId === brand.id }"
						@tap="chooseFormBrand(brand.id)"
					>{{ brand.name }}</text>
					<text class="chip" :class="{ active: !form.brandId }" @tap="chooseFormBrand(0)">其他品牌</text>
				</view>
				<input
					v-if="!form.brandId"
					class="form-input"
					:value="form.brandName"
					:maxlength="brandMax"
					placeholder="填写品牌名称"
					placeholder-class="search-placeholder"
					@input="onBrandNameInput"
				/>

				<text class="form-label">型号名称</text>
				<input
					class="form-input"
					:value="form.modelName"
					:maxlength="modelMax"
					placeholder="例如：杆身系列 + 型号"
					placeholder-class="search-placeholder"
					@input="onModelNameInput"
				/>

				<template v-if="category === 'cue'">
					<text class="form-label">杆型</text>
					<view class="chip-row">
						<text
							v-for="option in cueStyleOptions"
							:key="option.value"
							class="chip"
							:class="{ active: form.cueStyle === option.value }"
							@tap="chooseCueStyle(option.value)"
						>{{ option.label }}</text>
					</view>
				</template>
				<template v-else>
					<text class="form-label">身份（可多选）</text>
					<view class="chip-row">
						<text
							v-for="option in roleOptions"
							:key="option.value"
							class="chip"
							:class="{ active: form.roles.includes(option.value) }"
							@tap="chooseRole(option.value)"
						>{{ option.label }}</text>
					</view>
				</template>

				<text class="form-hint">你填写的品牌和型号名称只有你自己能看到，其他人会看到「未收录型号」。</text>
				<button class="primary-btn submit-btn" :disabled="submitting" @tap="submitUnlisted">
					{{ submitting ? '登记中...' : '登记这件装备' }}
				</button>
			</view>
		</template>
	</view>
</template>

<script setup>
import { computed, onUnmounted, ref } from 'vue'
import { onLoad, onReachBottom } from '@dcloudio/uni-app'
import gearImage from '@/components/gearImage.vue'
import { createGearItem, getGearBrands, getGearProducts } from '@/api/gear.js'
import { useGearStore } from '@/store/gear.js'
import { useUserStore } from '@/store/user.js'
import { usePageTheme } from '@/utils/page-theme.js'
import {
	GEAR_CATEGORY,
	GEAR_CATEGORY_LABELS,
	formatGearCueStyle,
	formatGearRoles,
	formatGearSpecs,
	gearProductTitle
} from '@/utils/gear-display.js'
import {
	CUE_STYLE_OPTIONS,
	GEAR_BRAND_NAME_MAX,
	GEAR_MODEL_NAME_MAX,
	ROLE_OPTIONS,
	buildUnlistedGearPayload,
	toggleGearRole
} from '@/utils/gear-form.js'

const PAGE_SIZE = 20
const SEARCH_DEBOUNCE_MS = 350

const { isDarkMode } = usePageTheme()
const userStore = useUserStore()
const gearStore = useGearStore()

const createTabs = [
	{ value: GEAR_CATEGORY.CUE, label: '球杆' },
	{ value: GEAR_CATEGORY.BREAK_JUMP, label: '冲杆 / 跳杆' }
]
const cueStyleOptions = CUE_STYLE_OPTIONS
const roleOptions = ROLE_OPTIONS
const brandMax = GEAR_BRAND_NAME_MAX
const modelMax = GEAR_MODEL_NAME_MAX

const isPick = ref(false)
const pickToken = ref('')
const category = ref(GEAR_CATEGORY.CUE)

const brands = ref([])
const brandId = ref(0)
const keyword = ref('')
const houseProduct = ref(null)
const products = ref([])
const page = ref(1)
const hasMore = ref(false)
const listLoading = ref(false)
const listError = ref('')

const showUnlisted = ref(false)
const submitting = ref(false)
const form = ref({ brandId: 0, brandName: '', modelName: '', cueStyle: '', roles: [] })

const categoryLabel = computed(() => GEAR_CATEGORY_LABELS[category.value] || '装备')
const houseHint = computed(() => (
	category.value === GEAR_CATEGORY.CHALK ? '用球房的巧克也能记录' : '用球房的公杆打球也能记录'
))

const productTitle = gearProductTitle
const productSubtitle = (product) => {
	const parts = []
	if (product.category === GEAR_CATEGORY.CUE) {
		const style = formatGearCueStyle(product.cue_style)
		if (style) parts.push(style)
	} else if (product.category === GEAR_CATEGORY.BREAK_JUMP) {
		const roles = formatGearRoles(product.roles)
		if (roles) parts.push(roles)
	}
	for (const spec of formatGearSpecs(product)) {
		parts.push(spec.key === 'weight' ? spec.value.split('（')[0] : spec.value)
	}
	return parts.join(' · ')
}

const currentIdentity = () => ({
	userId: userStore.userId,
	authGeneration: userStore.authGeneration
})

const sameIdentity = (identity) => (
	identity.userId === userStore.userId && identity.authGeneration === userStore.authGeneration
)

// 每次换类别、品牌或关键词都递增：旧请求的响应不能写进新的列表。
let queryVersion = 0
let searchTimer = null

const showError = (error, fallback) => {
	uni.showToast({ title: error?.message || fallback, icon: 'none' })
}

const loadBrands = async () => {
	const identity = currentIdentity()
	const version = queryVersion
	try {
		const res = await getGearBrands({ category: category.value })
		if (!sameIdentity(identity) || version !== queryVersion) return
		brands.value = res?.success ? (res.list || []) : []
	} catch (error) {
		if (!sameIdentity(identity) || version !== queryVersion) return
		// 品牌筛选只是辅助：读取失败不影响直接搜索与浏览型号。
		brands.value = []
	}
}

const loadHouse = async () => {
	houseProduct.value = null
	if (category.value !== GEAR_CATEGORY.CUE && category.value !== GEAR_CATEGORY.CHALK) return
	const identity = currentIdentity()
	const version = queryVersion
	try {
		const res = await getGearProducts({ category: category.value, house: true, page: 1, page_size: 1 })
		if (!sameIdentity(identity) || version !== queryVersion) return
		houseProduct.value = res?.success ? (res.list?.[0] || null) : null
	} catch (error) {
		if (!sameIdentity(identity) || version !== queryVersion) return
		houseProduct.value = null
	}
}

const loadProducts = async ({ append = false } = {}) => {
	const identity = currentIdentity()
	const version = queryVersion
	const nextPage = append ? page.value + 1 : 1
	listLoading.value = true
	listError.value = ''
	try {
		// 小程序的 uni.request 会把值为 undefined 的 GET 参数拼成字符串 "undefined"，后端解析 brand_id 会返回 400，所以只带有值的字段。
		const params = { category: category.value, page: nextPage, page_size: PAGE_SIZE }
		if (brandId.value) params.brand_id = brandId.value
		const word = keyword.value.trim()
		if (word) params.keyword = word
		const res = await getGearProducts(params)
		if (!sameIdentity(identity) || version !== queryVersion) return
		if (!res?.success) throw new Error(res?.message || '型号加载失败')
		const list = (res.list || []).filter((item) => !item.house || keyword.value.trim() || brandId.value)
		products.value = append ? products.value.concat(list) : list
		page.value = nextPage
		hasMore.value = (res.total || 0) > nextPage * PAGE_SIZE
	} catch (error) {
		if (!sameIdentity(identity) || version !== queryVersion) return
		listError.value = error?.message || '请检查网络后重试'
		if (!append) products.value = []
	} finally {
		if (version === queryVersion) listLoading.value = false
	}
}

const reloadList = () => {
	queryVersion += 1
	page.value = 1
	hasMore.value = false
	products.value = []
	loadProducts()
}

// 还有下一页，或上一次加载下一页失败（点击重试）时才继续加载。
const loadMore = () => {
	if (listLoading.value) return
	if (!hasMore.value && !listError.value) return
	loadProducts({ append: true })
}

const resetForm = () => {
	form.value = { brandId: 0, brandName: '', modelName: '', cueStyle: '', roles: [] }
}

const startCategory = () => {
	queryVersion += 1
	brandId.value = 0
	keyword.value = ''
	products.value = []
	brands.value = []
	page.value = 1
	hasMore.value = false
	showUnlisted.value = false
	resetForm()
	loadBrands()
	loadHouse()
	loadProducts()
}

const switchCategory = (value) => {
	if (category.value === value) return
	category.value = value
	startCategory()
}

const selectBrand = (id) => {
	if (brandId.value === id) return
	brandId.value = id
	reloadList()
}

const onKeywordInput = (event) => {
	keyword.value = event?.detail?.value || ''
	if (searchTimer) clearTimeout(searchTimer)
	searchTimer = setTimeout(reloadList, SEARCH_DEBOUNCE_MS)
}

const applyKeyword = () => {
	if (searchTimer) clearTimeout(searchTimer)
	reloadList()
}

const clearKeyword = () => {
	keyword.value = ''
	applyKeyword()
}

const finishPick = (product) => {
	if (pickToken.value) uni.$emit(`gear:pick:${pickToken.value}`, { product })
	uni.navigateBack()
}

const pickNone = () => {
	finishPick(null)
}

const afterCreated = (item) => {
	gearStore.invalidate()
	const target = item?.id
		? `/subPages/gear/item?id=${item.id}&added=1`
		: '/subPages/gear/index'
	uni.redirectTo({ url: target })
}

const createFromProduct = async (product) => {
	if (submitting.value) return
	const identity = currentIdentity()
	submitting.value = true
	try {
		const res = await createGearItem({ category: category.value, product_id: product.id })
		if (!sameIdentity(identity)) return
		if (!res?.success) throw new Error(res?.message || '添加失败')
		afterCreated(res.item)
	} catch (error) {
		if (!sameIdentity(identity)) return
		showError(error, '添加失败，请重试')
	} finally {
		submitting.value = false
	}
}

const selectProduct = (product) => {
	if (isPick.value) {
		finishPick(product)
		return
	}
	uni.showModal({
		title: '添加装备',
		content: `添加「${gearProductTitle(product)}」到我的装备？`,
		confirmText: '添加',
		success: (result) => {
			if (result.confirm) createFromProduct(product)
		}
	})
}

const toggleUnlisted = () => {
	showUnlisted.value = !showUnlisted.value
}

const chooseFormBrand = (id) => {
	form.value = { ...form.value, brandId: id, brandName: id ? '' : form.value.brandName }
}

const onBrandNameInput = (event) => {
	form.value = { ...form.value, brandName: event?.detail?.value || '' }
}

const onModelNameInput = (event) => {
	form.value = { ...form.value, modelName: event?.detail?.value || '' }
}

const chooseCueStyle = (value) => {
	form.value = { ...form.value, cueStyle: value }
}

const chooseRole = (value) => {
	form.value = { ...form.value, roles: toggleGearRole(form.value.roles, value) }
}

const submitUnlisted = async () => {
	if (submitting.value) return
	const built = buildUnlistedGearPayload({ ...form.value, category: category.value })
	if (!built.ok) {
		uni.showToast({ title: built.message, icon: 'none' })
		return
	}
	const identity = currentIdentity()
	submitting.value = true
	try {
		const res = await createGearItem(built.payload)
		if (!sameIdentity(identity)) return
		if (!res?.success) throw new Error(res?.message || '登记失败')
		afterCreated(res.item)
	} catch (error) {
		if (!sameIdentity(identity)) return
		showError(error, '登记失败，请重试')
	} finally {
		submitting.value = false
	}
}

onLoad((options) => {
	if (!userStore.isLoggedIn || !userStore.userId) {
		uni.navigateTo({ url: '/pages/login/login' })
		return
	}
	isPick.value = options?.mode === 'pick'
	pickToken.value = String(options?.token || '')
	const requested = String(options?.category || '')
	const allowed = isPick.value
		? [GEAR_CATEGORY.TIP, GEAR_CATEGORY.CHALK]
		: [GEAR_CATEGORY.CUE, GEAR_CATEGORY.BREAK_JUMP]
	category.value = allowed.includes(requested) ? requested : allowed[0]
	uni.setNavigationBarTitle({ title: isPick.value ? `选择${GEAR_CATEGORY_LABELS[category.value]}` : '添加装备' })
	startCategory()
})

onReachBottom(() => {
	if (hasMore.value) loadMore()
})

onUnmounted(() => {
	if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style lang="scss" scoped>
@import './add.scss';
</style>
