---
id: "mem_b5639da7"
topic: "PF6 Select interaction pattern in Playwright tests"
tags:
  - playwright
  - pf6
  - testing
  - select
  - dropdown
phase: 0
difficulty: 0.6
created_at: "2026-05-07T15:12:29.812921+00:00"
created_session: 17
---
PF6 v6 Select components render options as MenuItem buttons, not native `<option>` elements.

`getByRole('option', ...)` times out because PF6 MenuItem uses `role="menuitem"` in most contexts. Multiple test iterations were wasted discovering this.

**Pattern:**
- Open: `page.getByText('Select child type...').click()`
- Select: `page.getByText(optionName).click()`
- If option text appears elsewhere on the page (e.g., tree browser): `page.locator('button.pf-v6-c-menu__item').filter({ hasText: 'name' }).click()`

Reference: `ui/src/components/LinkModal.browser.test.tsx` and `ui/src/components/AddChildModal.browser.test.tsx` show the working pattern.
