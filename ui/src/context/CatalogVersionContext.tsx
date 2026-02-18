import { createContext, useContext, useState, type ReactNode } from 'react'

interface CatalogVersionContextType {
  selectedCatalogVersion: string | null
  setSelectedCatalogVersion: (id: string | null) => void
}

const CatalogVersionContext = createContext<CatalogVersionContextType | undefined>(undefined)

export function CatalogVersionProvider({ children }: { children: ReactNode }) {
  const [selectedCatalogVersion, setSelectedCatalogVersion] = useState<string | null>(null)
  return (
    <CatalogVersionContext.Provider value={{ selectedCatalogVersion, setSelectedCatalogVersion }}>
      {children}
    </CatalogVersionContext.Provider>
  )
}

export function useCatalogVersion(): CatalogVersionContextType {
  const ctx = useContext(CatalogVersionContext)
  if (!ctx) throw new Error('useCatalogVersion must be used within CatalogVersionProvider')
  return ctx
}
