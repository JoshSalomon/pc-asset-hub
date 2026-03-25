import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import InstanceDetailPanel from './InstanceDetailPanel'
import type { EntityInstance, ReferenceDetail } from '../types'

const mockInstance: EntityInstance = {
  id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'my-server',
  description: 'A server instance', version: 2,
  attributes: [
    { name: 'endpoint', type: 'string', value: 'https://example.com' },
    { name: 'status', type: 'enum', value: 'active' },
  ],
  parent_chain: [
    { instance_id: 'p1', instance_name: 'parent-srv', entity_type_name: 'cluster' },
  ],
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-02T00:00:00Z',
}

const mockInstanceNoParent: EntityInstance = {
  ...mockInstance,
  parent_chain: [],
}

const mockInstanceNoAttrs: EntityInstance = {
  ...mockInstance,
  attributes: [],
  parent_chain: [],
}

const mockForwardRefs: ReferenceDetail[] = [
  { link_id: 'l1', association_name: 'uses-model', association_type: 'directional',
    instance_id: 'i4', instance_name: 'gpt-4', entity_type_name: 'model' },
]

const mockReverseRefs: ReferenceDetail[] = [
  { link_id: 'l2', association_name: 'monitored-by', association_type: 'directional',
    instance_id: 'i5', instance_name: 'monitor-1', entity_type_name: 'guardrail' },
]

// T-20.11: renders instance name as heading
test('T-20.11: renders instance name as heading', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
})

// T-20.12: renders description
test('T-20.12: renders description', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText('A server instance')).toBeVisible()
})

// T-20.13: renders version info
test('T-20.13: renders version info', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText(/Version 2/)).toBeVisible()
})

// T-20.14: renders attributes table
test('T-20.14: renders attributes table', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByRole('heading', { name: 'Attributes' })).toBeVisible()
  await expect.element(page.getByText('https://example.com').first()).toBeVisible()
  await expect.element(page.getByText('active')).toBeVisible()
})

// T-20.15: no attributes section when empty
test('T-20.15: no attributes section when empty', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstanceNoAttrs}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  const attrHeading = page.getByRole('heading', { name: 'Attributes' })
  expect(attrHeading.elements().length).toBe(0)
})

// T-20.16: renders parent chain breadcrumb
test('T-20.16: renders parent chain breadcrumb', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText('test-catalog').first()).toBeVisible()
  await expect.element(page.getByText('cluster: parent-srv').first()).toBeVisible()
})

// T-20.17: no breadcrumb when no parent chain
test('T-20.17: no breadcrumb when no parent chain', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstanceNoParent}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  // The instance name should still be there
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // But no parent chain breadcrumb entries (cluster: parent-srv)
  expect(page.getByText('cluster: parent-srv').elements().length).toBe(0)
})

// T-20.18: renders forward references
test('T-20.18: renders forward references', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={mockForwardRefs}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText('Forward References')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'gpt-4' })).toBeVisible()
})

// T-20.19: renders reverse references
test('T-20.19: renders reverse references', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={mockReverseRefs}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText('Referenced By')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'monitor-1' })).toBeVisible()
})

// T-20.20: clicking reference calls onNavigateToRef
test('T-20.20: clicking reference calls onNavigateToRef', async () => {
  const onNav = vi.fn()
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={mockForwardRefs}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={onNav}
    />
  )
  await page.getByRole('button', { name: 'gpt-4' }).click()
  expect(onNav).toHaveBeenCalledWith('i4')
})

// T-20.21: shows "No references" when both empty
test('T-20.21: shows no references message when both empty', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText('No references.')).toBeVisible()
})

// T-20.22: shows loading spinner for refs
test('T-20.22: shows refs loading spinner', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={true}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByLabelText('Loading references')).toBeVisible()
})

// T-20.23: renders both forward and reverse refs together
test('T-20.23: renders both forward and reverse refs', async () => {
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="test-catalog"
      forwardRefs={mockForwardRefs}
      reverseRefs={mockReverseRefs}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  await expect.element(page.getByText('Forward References')).toBeVisible()
  await expect.element(page.getByText('Referenced By')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'gpt-4' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'monitor-1' })).toBeVisible()
})

// T-20.24: attribute with null value shows dash
test('T-20.24: attribute with null value shows dash', async () => {
  const inst: EntityInstance = {
    ...mockInstance,
    attributes: [{ name: 'optField', type: 'string', value: null as unknown as string }],
  }
  render(
    <InstanceDetailPanel
      instance={inst}
      catalogName="test-catalog"
      forwardRefs={[]}
      reverseRefs={[]}
      refsLoading={false}
      onNavigateToRef={() => {}}
    />
  )
  // The dash character used for null values
  await expect.element(page.getByText('\u2014')).toBeVisible()
})

// Coverage: clicking reverse ref navigation button calls onNavigateToRef
test('clicking reverse ref link calls onNavigateToRef', async () => {
  const onNav = vi.fn()
  render(
    <InstanceDetailPanel
      instance={mockInstance}
      catalogName="my-catalog"
      forwardRefs={[]}
      reverseRefs={[{ link_id: 'l1', association_name: 'monitored-by', association_type: 'directional', instance_id: 'i99', instance_name: 'monitor-1', entity_type_name: 'monitor' }]}
      refsLoading={false}
      onNavigateToRef={onNav}
    />
  )
  await page.getByRole('button', { name: 'monitor-1' }).click()
  expect(onNav).toHaveBeenCalledWith('i99')
})
