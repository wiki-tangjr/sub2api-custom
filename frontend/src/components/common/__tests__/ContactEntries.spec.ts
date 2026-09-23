import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ContactEntries from '../ContactEntries.vue'
import type { ContactEntry } from '@/types'

const showSuccess = vi.fn()
const showError = vi.fn()

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError }),
}))

function entry(overrides: Partial<ContactEntry> = {}): ContactEntry {
  return {
    id: 'a',
    enabled: true,
    label: 'Label',
    icon_type: 'emoji',
    icon: '💬',
    type: 'link',
    url: 'https://example.com',
    display: 'modal',
    open_target: 'new_tab',
    sort_order: 0,
    ...overrides,
  }
}

function mountEntries(entries: ContactEntry[], props: Record<string, unknown> = {}) {
  return mount(ContactEntries, {
    props: { entries, ...props },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /></div>' },
        Icon: true,
      },
    },
  })
}

describe('ContactEntries (#23/#24/#29)', () => {
  afterEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
  })

  it('renders without any group heading when no entry is grouped', () => {
    const wrapper = mountEntries([entry(), entry({ id: 'b', label: 'Second' })])

    expect(wrapper.text()).toContain('Label')
    expect(wrapper.text()).toContain('Second')
    expect(wrapper.findAll('p').filter((p) => p.text() === 'Official')).toHaveLength(0)
  })

  it('groups adjacent entries that share the same group name', () => {
    const wrapper = mountEntries([
      entry({ id: 'a', label: 'Telegram', group: 'Official' }),
      entry({ id: 'b', label: 'WeChat', group: 'Official' }),
      entry({ id: 'c', label: 'Support', group: 'Support', type: 'text', value: 'id-1' }),
    ])

    const headings = wrapper.findAll('p').map((p) => p.text())
    expect(headings).toContain('Official')
    expect(headings).toContain('Support')
  })

  it('keeps ungrouped entries visible alongside grouped ones', () => {
    const wrapper = mountEntries([
      entry({ id: 'a', label: 'First' }),
      entry({ id: 'b', label: 'Second', group: 'Official' }),
    ])

    expect(wrapper.text()).toContain('First')
    expect(wrapper.text()).toContain('Second')
  })

  it('drops disabled entries and invalid link entries', () => {
    const wrapper = mountEntries([
      entry({ id: 'a', label: 'Hidden', enabled: false }),
      entry({ id: 'b', label: 'Bad', url: 'javascript:alert(1)' }),
      entry({ id: 'c', label: 'Good' }),
    ])

    expect(wrapper.text()).not.toContain('Hidden')
    expect(wrapper.text()).not.toContain('Bad')
    expect(wrapper.text()).toContain('Good')
  })

  it('renders a consistent contact sheet for mixed entry types', () => {
    const wrapper = mountEntries([
      entry({ id: 'text', label: 'WeChat', type: 'text', value: 'shiyu_tv', description: '工作时间内回复' }),
      entry({ id: 'link', label: 'Telegram', group: '官方群组' }),
    ], { variant: 'sheet' })

    expect(wrapper.findAll('[data-testid="contact-sheet-entry"]')).toHaveLength(2)
    expect(wrapper.find('[data-testid="contact-copy-button"]').attributes('aria-label')).toBe('common.copy')
    expect(wrapper.find('a').attributes('href')).toBe('https://example.com/')
    expect(wrapper.find('a').attributes('target')).toBe('_blank')
    expect(wrapper.text()).toContain('shiyu_tv')
    expect(wrapper.text()).toContain('官方群组')
  })

  it('shows the hover card on mouseenter and closes it shortly after mouseleave', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mountEntries([entry({ display: 'hover' })])
      const cell = wrapper.find('div.relative')

      await cell.trigger('mouseenter')
      await nextTick()
      expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(true)

      await cell.trigger('mouseleave')
      await nextTick()
      // 魔改 #24: 关闭是延时的，给鼠标从按钮移动到卡片留出时间。
      expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(true)

      vi.advanceTimersByTime(300)
      await nextTick()
      expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })

  it('keeps the hover card open when the pointer briefly leaves and returns', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mountEntries([entry({ display: 'hover' })])
      const cell = wrapper.find('div.relative')

      await cell.trigger('mouseenter')
      await nextTick()
      expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(true)

      // 模拟鼠标从按钮移向卡片：中途触发一次 mouseleave 后立刻重新进入。
      await cell.trigger('mouseleave')
      vi.advanceTimersByTime(60)
      await cell.trigger('mouseenter')
      vi.advanceTimersByTime(300)
      await nextTick()

      expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(true)
      expect(wrapper.find('button').exists()).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })

  it('opens the hover panel when the trigger receives keyboard focus', async () => {
    const wrapper = mountEntries([entry({ display: 'hover' })])
    const cell = wrapper.find('div.relative')

    await cell.trigger('focusin')
    await nextTick()

    expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(true)
    expect(wrapper.find('.contact-hover-panel').classes()).toContain('contact-hover-card')
  })

  it('opens the modal when a hover entry is tapped on a device without hover support', async () => {
    const original = window.matchMedia
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })) as unknown as typeof window.matchMedia

    try {
      const wrapper = mountEntries([entry({ display: 'hover' })])
      await nextTick()
      await wrapper.find('button').trigger('click')
      await nextTick()

      expect(wrapper.find('a').attributes('href')).toBe('https://example.com/')
      expect(wrapper.find('[data-testid="contact-hover-panel"]').exists()).toBe(false)
    } finally {
      window.matchMedia = original
    }
  })

  it('keeps hover entries click-inert on devices that support hover', async () => {
    const wrapper = mountEntries([entry({ display: 'hover' })])
    await nextTick()
    await wrapper.find('button').trigger('click')
    await nextTick()

    expect(wrapper.find('a').exists()).toBe(false)
    expect(wrapper.find('button').exists()).toBe(true)
  })
})
