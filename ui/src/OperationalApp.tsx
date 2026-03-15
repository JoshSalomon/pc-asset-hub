import '@patternfly/patternfly/patternfly.css'
import { useEffect, useState } from 'react'
import { Routes, Route } from 'react-router-dom'
import {
  Page,
  Masthead,
  MastheadMain,
  MastheadBrand,
  MastheadContent,
  Toolbar,
  ToolbarItem,
  ToolbarContent,
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
} from '@patternfly/react-core'
import { setAuthRole } from './api/client'
import type { Role } from './types'
import OperationalCatalogListPage from './pages/operational/OperationalCatalogListPage'
import OperationalCatalogDetailPage from './pages/operational/OperationalCatalogDetailPage'

const ROLES: Role[] = ['RO', 'RW', 'Admin', 'SuperAdmin']

function OperationalApp() {
  const [role, setRole] = useState<Role>('RO')
  const [roleSelectOpen, setRoleSelectOpen] = useState(false)

  useEffect(() => {
    setAuthRole(role)
  }, [role])

  return (
    <Page
      masthead={
        <Masthead>
          <MastheadMain>
            <MastheadBrand>AI Asset Hub — Data Viewer</MastheadBrand>
          </MastheadMain>
          <MastheadContent>
            <Toolbar>
              <ToolbarContent>
                <ToolbarItem>
                  <Select
                    isOpen={roleSelectOpen}
                    selected={role}
                    onSelect={(_e: React.MouseEvent | undefined, value: string | number | undefined) => {
                      setRole(value as Role)
                      setRoleSelectOpen(false)
                    }}
                    onOpenChange={setRoleSelectOpen}
                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                      <MenuToggle ref={toggleRef} onClick={() => setRoleSelectOpen(!roleSelectOpen)} isExpanded={roleSelectOpen}>
                        Role: {role}
                      </MenuToggle>
                    )}
                  >
                    {ROLES.map((r) => (
                      <SelectOption key={r} value={r}>{r}</SelectOption>
                    ))}
                  </Select>
                </ToolbarItem>
              </ToolbarContent>
            </Toolbar>
          </MastheadContent>
        </Masthead>
      }
    >
      <Routes>
        <Route path="/catalogs/:name" element={<OperationalCatalogDetailPage role={role} />} />
        <Route path="*" element={<OperationalCatalogListPage role={role} />} />
      </Routes>
    </Page>
  )
}

export default OperationalApp
