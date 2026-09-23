import { describe, expect, it } from 'vitest'
import settingsViewSource from '../SettingsView.vue?raw'

describe('SettingsView custom menu open_mode config', () => {
  it('declares open_mode in custom_menu_items type and form state', () => {
    expect(settingsViewSource).toContain('open_mode?: "iframe" | "new_tab"')
    expect(settingsViewSource).toContain('open_mode: "iframe"')
  })

  it('renders openMode select control with iframe and new_tab options in the grid', () => {
    expect(settingsViewSource).toContain('t("admin.settings.customMenu.openMode")')
    expect(settingsViewSource).toContain(':model-value="item.open_mode ?? \'iframe\'"')
    expect(settingsViewSource).toContain(':options="customMenuOpenModeOptions"')
    expect(settingsViewSource).toContain("item.open_mode = $event as 'iframe' | 'new_tab'")
    expect(settingsViewSource).toContain('{ value: "iframe", label: t("admin.settings.customMenu.openModeIframe") }')
    expect(settingsViewSource).toContain('{ value: "new_tab", label: t("admin.settings.customMenu.openModeNewTab") }')
    expect(settingsViewSource).toContain('t("admin.settings.customMenu.openModeIframe")')
    expect(settingsViewSource).toContain('t("admin.settings.customMenu.openModeNewTab")')
  })

  it('normalizes legacy custom menu items without open_mode to iframe on load', () => {
    expect(settingsViewSource).toContain(
      'open_mode: item.open_mode === "new_tab" ? "new_tab" : "iframe"',
    )
  })
})
