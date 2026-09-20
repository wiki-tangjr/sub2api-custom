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

describe('ContactEntries (#23)', () => {
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

  it('shows the hover card on mouseenter and hides it on mouseleave', async () => {
    const wrapper = mountEntries([entry({ display: 'hover' })])
    const cell = wrapper.find('div.relative')

    await cell.trigger('mouseenter')
    await nextTick()
    expect(wrapper.find('.z-40').exists()).toBe(true)

    await cell.trigger('mouseleave')
    await nextTick()
    expect(wrapper.find('.z-40').exists()).toBe(false)
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
      expect(wrapper.find('.z-40').exists()).toBe(false)
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
