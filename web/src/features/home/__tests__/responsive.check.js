/* Copyright (C) 2023-2026 QuantumNous. Licensed under AGPL-3.0-or-later. */
// eslint-disable-next-line no-unused-vars -- Invoked by the Playwright CLI's run-code command.
async function checkHomeLayout(page) {
  const viewports = [
    { width: 1440, height: 900 },
    { width: 1366, height: 900 },
    { width: 1920, height: 1080 },
    { width: 1024, height: 900 },
    { width: 390, height: 844 },
    { width: 360, height: 780 },
  ]
  const results = []

  await page.waitForSelector('.unikeyx-hero-image')
  await page.waitForFunction(
    () => document.querySelector('.unikeyx-hero-image').naturalWidth > 0
  )

  for (const viewport of viewports) {
    await page.setViewportSize(viewport)
    await page.evaluate(() => window.scrollTo(0, 0))
    await page.evaluate(
      () =>
        new Promise((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(resolve))
        )
    )

    const layout = await page.evaluate(() => {
      const hero = document
        .querySelector('.unikeyx-hero')
        .getBoundingClientRect()
      const buttons = [
        ...document.querySelectorAll('.unikeyx-actions [data-slot="button"]'),
      ]
      const unreadableButtons = buttons
        .filter((button) => {
          const style = getComputedStyle(button)
          return style.backgroundColor === style.color
        })
        .map((button) => button.textContent.trim())
      const clippedButtons = buttons
        .filter((button) => {
          const bounds = button.getBoundingClientRect()
          const walker = document.createTreeWalker(button, NodeFilter.SHOW_TEXT)
          let node
          while ((node = walker.nextNode())) {
            if (!node.textContent.trim() || node.parentElement.closest('svg')) {
              continue
            }
            const range = document.createRange()
            range.selectNodeContents(node)
            const text = range.getBoundingClientRect()
            if (text.left < bounds.left - 1 || text.right > bounds.right + 1) {
              return true
            }
          }
          return (
            bounds.height < 44 || bounds.right > innerWidth || bounds.left < 0
          )
        })
        .map((button) => button.textContent.trim())
      const clippedNavigation = [
        ...document.querySelectorAll('header nav a[title]'),
      ]
        .filter(
          (link) => link.offsetParent && link.scrollWidth > link.clientWidth + 1
        )
        .map((link) => link.textContent.trim())
      const contact = document.querySelector('footer .custom-footer > div')
      const canvas = document.createElement('canvas')
      canvas.width = canvas.height = 1
      const context = canvas.getContext('2d', { willReadFrequently: true })
      const luminances = [
        getComputedStyle(contact).color,
        getComputedStyle(document.querySelector('.unikeyx-home'))
          .backgroundColor,
      ].map((color) => {
        context.fillStyle = color
        context.fillRect(0, 0, 1, 1)
        const channels = [...context.getImageData(0, 0, 1, 1).data]
          .slice(0, 3)
          .map((channel) => {
            const value = channel / 255
            return value <= 0.04045
              ? value / 12.92
              : ((value + 0.055) / 1.055) ** 2.4
          })
        return (
          channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722
        )
      })
      const contactContrast =
        (Math.max(...luminances) + 0.05) / (Math.min(...luminances) + 0.05)

      return {
        width: innerWidth,
        height: innerHeight,
        scrollWidth: document.documentElement.scrollWidth,
        heroBottom: Math.round(hero.bottom),
        providers: document.querySelectorAll('.unikeyx-providers li').length,
        unreadableButtons,
        clippedButtons,
        clippedNavigation,
        contactContrast: Math.round(contactContrast * 100) / 100,
      }
    })

    if (layout.scrollWidth > viewport.width) {
      throw new Error(`Horizontal overflow: ${JSON.stringify(layout)}`)
    }
    if (layout.heroBottom >= viewport.height) {
      throw new Error(
        `The next section is hidden below the hero: ${JSON.stringify(layout)}`
      )
    }
    if (
      layout.unreadableButtons.length ||
      layout.clippedButtons.length ||
      layout.clippedNavigation.length
    ) {
      throw new Error(
        `Unreadable or clipped controls: ${JSON.stringify(layout)}`
      )
    }
    if (layout.providers !== 7) {
      throw new Error(
        `Missing cloud or model provider: ${JSON.stringify(layout)}`
      )
    }
    if (layout.contactContrast < 4.5) {
      throw new Error(
        `The footer contact text lacks contrast: ${JSON.stringify(layout)}`
      )
    }

    await page.evaluate(() => window.scrollTo(0, 120))
    await page.waitForFunction(
      () =>
        document.querySelector('header nav').getBoundingClientRect().height < 65
    )
    await page.evaluate(
      () =>
        new Promise((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(resolve))
        )
    )
    const clippedScrolledNavigation = await page.evaluate(() =>
      [...document.querySelectorAll('header nav a[title]')]
        .filter(
          (link) => link.offsetParent && link.scrollWidth > link.clientWidth + 1
        )
        .map((link) => link.textContent.trim())
    )
    if (clippedScrolledNavigation.length) {
      throw new Error(
        `Scrolled navigation is clipped: ${clippedScrolledNavigation.join(', ')}`
      )
    }
    results.push(layout)
  }

  await page.evaluate(() => window.scrollTo(0, 0))
  return results
}
