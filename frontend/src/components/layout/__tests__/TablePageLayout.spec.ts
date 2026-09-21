import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../TablePageLayout.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('TablePageLayout responsive table scrolling', () => {
  it('does not disable the table horizontal scroll container in mobile mode', () => {
    const tableWrapperBlocks = Array.from(
      componentSource.matchAll(/([^{}]*:deep\(\.table-wrapper\)[^{}]*)\{([^{}]*)\}/g)
    )

    expect(tableWrapperBlocks.length).toBeGreaterThan(0)

    const baseBlock = tableWrapperBlocks.find(([selector]) => !selector.includes('.mobile-mode'))
    const mobileBlocks = tableWrapperBlocks.filter(([selector]) => selector.includes('.mobile-mode'))

    expect(baseBlock?.[2]).toContain('overflow-x-auto')
    expect(mobileBlocks.every(([, , declarations]) => !declarations.includes('overflow-visible'))).toBe(
      true
    )
  })

  // 魔改 #25：移动端必须解开桌面端的固定高度。
  // 桌面端用 calc(100vh - 64px - 4rem) 做表体内部滚动；移动端内容靠文档流撑开，
  // 若沿用固定高度，内容就会溢出到盒子之外，AppLayout 的备案 footer 会被排到
  // 一屏处(页面中段)，滚动时看起来像“卡在中间”，并遮住下方内容。
  it('releases the desktop fixed height in mobile mode', () => {
    // 桌面端固定高度必须保留：表体内部滚动方案不能回退。
    expect(componentSource).toContain('height: calc(100vh - 64px - 4rem)')

    // 移动端必须有一条把高度解开的规则。
    const mobileLayout = componentSource.match(/\.table-page-layout\.mobile-mode\s*\{([^}]*)\}/)

    expect(mobileLayout).not.toBeNull()
    expect(mobileLayout?.[1]).toMatch(/height:\s*auto/)
  })
})
