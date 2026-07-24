import { expect, test } from '@playwright/test'

const baseURL = process.env.HOMEVOX_E2E_BASE_URL ?? 'http://127.0.0.1:18088'
test.use({ baseURL, viewport: { width: 1440, height: 960 } })

test('captures four real product-flow states from the production shell', async ({ page }, testInfo) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: '导入真实户型图' })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('issue-19-import-ai.png'), fullPage: true })

  await page.goto('/?e2e=wall-fixture')
  await page.getByRole('button', { name: '校正 2D' }).click()
  await expect(page.getByLabel('2D 墙体编辑器')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('issue-19-2d-correction.png'), fullPage: true })

  await page.getByRole('button', { name: '生成 3D' }).click()
  await expect(page.getByRole('heading', { name: '确认 3D 空间' })).toBeVisible()
  await expect(page.getByRole('button', { name: '完成并打开 3D' })).toBeVisible()
  await expect(page.getByLabel('2D 墙体编辑器')).toHaveCount(0)
  await page.screenshot({ path: testInfo.outputPath('issue-19-3d-confirm.png'), fullPage: true })

  await page.getByRole('button', { name: '完成并打开 3D' }).click()
  await expect(page.getByLabel('2D 墙体编辑器')).toBeVisible()
  await expect(page.getByLabel('3D 户型预览')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('issue-19-linked-workspace.png'), fullPage: true })
})

test('uses stable wall IDs for 2D selection and exposes only product-facing inspector context', async ({ page }) => {
  await page.goto('/?e2e=wall-fixture')
  await page.getByRole('button', { name: '2D/3D 联动' }).click()
  await page.getByTestId('wall-hit-wall-1').click({ position: { x: 80, y: 1 }, force: true })
  await expect(page.getByTestId('selected-wall-id')).toHaveText('wall-1')
  await expect(page.getByRole('button', { name: '添加门' })).toBeEnabled()
  await page.getByTestId('opening-handle-window-1').click()
  await expect(page.getByTestId('selected-opening-id')).toContainText('window-1')
  await expect(page.getByTestId('selected-opening-id')).toContainText('wall-2')

  const accessibility = await page.locator('body').ariaSnapshot()
  expect(accessibility).not.toMatch(/(?:WASM|Grid|triangles|fallback|结构化 JSON)/i)
  await expect(page.locator('pre')).toHaveCount(0)
})

test('keeps 3D confirmation separate and fails closed with a user-facing geometry message', async ({ page }) => {
  await page.goto('/?e2e=invalid-opening')
  await page.getByRole('button', { name: '生成 3D' }).click()
  await expect(page.getByRole('alert')).toContainText('当前开口数据无法生成 3D')
  await page.getByRole('button', { name: '返回 2D 校正' }).click()
  await expect(page.getByLabel('2D 墙体编辑器')).toBeVisible()
})
