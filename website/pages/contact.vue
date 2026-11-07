<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import PageHero from '../components/PageHero.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import { contactInfo } from '../data/contact'
import { siteConfig } from '../data/site'
import { usePageSeo } from '../composables/usePageSeo'

usePageSeo({
  title: contactInfo.title,
  description: contactInfo.description,
  path: '/contact'
})

const categoryOptions = [
  { label: '意见反馈', value: 'feedback' },
  { label: '投诉', value: 'complaint' },
  { label: '举报', value: 'report' }
]

const ticketForm = reactive({
  category: 'complaint',
  content: '',
  contact: ''
})
const isSubmitting = ref(false)
const submitMessage = ref('')
const submitState = ref<'success' | 'error' | ''>('')

const feedbackApiUrl = computed(() => {
  const baseUrl = contactInfo.feedbackApiBaseUrl.replace(/\/+$/, '')
  return `${baseUrl}/api/feedback/create`
})

async function submitFeedbackTicket() {
  const content = ticketForm.content.trim()
  const contact = ticketForm.contact.trim()
  submitMessage.value = ''
  submitState.value = ''

  if (!content) {
    submitMessage.value = '请填写投诉举报或反馈内容。'
    submitState.value = 'error'
    return
  }

  isSubmitting.value = true
  try {
    const response = await fetch(feedbackApiUrl.value, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        source: 'website',
        category: ticketForm.category,
        content,
        contact
      })
    })
    const result = await response.json().catch(() => ({}))
    if (!response.ok || !result.success) {
      throw new Error(result.message || '提交失败，请稍后重试。')
    }

    ticketForm.category = 'complaint'
    ticketForm.content = ''
    ticketForm.contact = ''
    submitState.value = 'success'
    submitMessage.value = '已提交，我们会尽快处理。'
  } catch (error) {
    submitState.value = 'error'
    submitMessage.value = error instanceof Error
      ? `${error.message} 也可以发送邮件至 ${contactInfo.email}。`
      : `提交失败，也可以发送邮件至 ${contactInfo.email}。`
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div>
    <SiteHeader :links="siteConfig.navLinks" />

    <main class="page-shell">
      <PageHero
        eyebrow="沟通与合作"
        :title="contactInfo.title"
        :description="contactInfo.description"
      />

      <section class="contact-card">
        <div>
          <p class="contact-card__label">联系邮箱</p>
          <a :href="`mailto:${contactInfo.email}`">{{ contactInfo.email }}</a>
        </div>

        <div>
          <p class="contact-card__label">座机</p>
          <a :href="`tel:${contactInfo.phone}`">{{ contactInfo.phone }}</a>
        </div>

        <div>
          <p class="contact-card__label">合作说明</p>
          <p>{{ contactInfo.businessNote }}</p>
        </div>
      </section>

      <section class="feedback-panel">
        <div class="feedback-panel__intro">
          <p class="feedback-panel__eyebrow">在线提交</p>
          <h2>投诉举报与意见反馈</h2>
          <p>{{ contactInfo.complaintNote }}</p>
        </div>

        <form class="feedback-form" @submit.prevent="submitFeedbackTicket">
          <label>
            <span>提交类型</span>
            <select v-model="ticketForm.category">
              <option v-for="item in categoryOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </label>

          <label>
            <span>内容</span>
            <textarea
              v-model="ticketForm.content"
              maxlength="2000"
              rows="6"
              placeholder="请写清投诉举报或反馈事项"
            />
          </label>

          <label>
            <span>联系方式</span>
            <input
              v-model="ticketForm.contact"
              maxlength="128"
              type="text"
              placeholder="手机号或邮箱，选填"
            >
          </label>

          <button type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? '提交中...' : '提交' }}
          </button>

          <p v-if="submitMessage" class="submit-message" :class="submitState">
            {{ submitMessage }}
          </p>
        </form>
      </section>
    </main>

    <SiteFooter :links="siteConfig.legalLinks" />
  </div>
</template>

<style scoped>
.page-shell {
  width: min(960px, calc(100vw - 32px));
  margin: 0 auto;
  padding-bottom: 72px;
}

.contact-card {
  display: grid;
  gap: 18px;
  padding: 28px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.contact-card__label {
  margin: 0 0 10px;
  color: var(--ui-brand-strong);
  font-weight: 600;
}

.contact-card a,
.contact-card p:last-child {
  color: rgba(31, 26, 16, 0.78);
  line-height: 1.8;
}

.feedback-panel {
  display: grid;
  gap: 24px;
  margin-top: 24px;
  padding: 28px;
  border-radius: 24px;
  background: var(--ui-surface-card);
  border: 1px solid rgba(31, 26, 16, 0.08);
}

.feedback-panel__intro {
  display: grid;
  gap: 10px;
}

.feedback-panel__intro h2,
.feedback-panel__intro p {
  margin: 0;
}

.feedback-panel__intro h2 {
  color: var(--ui-text-primary);
  font-size: 24px;
}

.feedback-panel__intro p {
  color: rgba(31, 26, 16, 0.72);
  line-height: 1.8;
}

.feedback-panel__eyebrow {
  color: var(--ui-brand-strong);
  font-weight: 700;
}

.feedback-form {
  display: grid;
  gap: 18px;
}

.feedback-form label {
  display: grid;
  gap: 8px;
  color: var(--ui-text-primary);
  font-weight: 600;
}

.feedback-form select,
.feedback-form textarea,
.feedback-form input {
  width: 100%;
  border: 1px solid var(--ui-border-default);
  border-radius: 12px;
  box-sizing: border-box;
  color: var(--ui-text-primary);
  font: inherit;
}

.feedback-form select,
.feedback-form input {
  height: 44px;
  padding: 0 12px;
}

.feedback-form textarea {
  padding: 12px;
  resize: vertical;
  line-height: 1.7;
}

.feedback-form button {
  width: 132px;
  height: 44px;
  border: 0;
  border-radius: 12px;
  background: var(--ui-surface-accent);
  color: #ffffff;
  cursor: pointer;
  font-weight: 700;
}

.feedback-form button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.submit-message {
  margin: 0;
  line-height: 1.7;
}

.submit-message.success {
  color: #16794c;
}

.submit-message.error {
  color: #b42318;
}
</style>
