import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Card,
  CardTitle,
  CardBody,
  CardHeader,
  Gallery,
  GalleryItem,
  Label,
  Spinner,
  Alert,
  EmptyState,
  EmptyStateBody,
  Divider,
  Icon,
} from '@patternfly/react-core'
import CubesIcon from '@patternfly/react-icons/dist/esm/icons/cubes-icon'
import DatabaseIcon from '@patternfly/react-icons/dist/esm/icons/database-icon'
import { api, setAuthRole } from '../api/client'
import type { Catalog, Role } from '../types'
import { statusColor } from '../utils/statusColor'

export default function LandingPage({ role }: { role: Role }) {
  const navigate = useNavigate()
  const [catalogs, setCatalogs] = useState<Catalog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadCatalogs = useCallback(async () => {
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const res = await api.catalogs.list()
      setCatalogs(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalogs')
    } finally {
      setLoading(false)
    }
  }, [role])

  useEffect(() => { loadCatalogs() }, [loadCatalogs])

  return (
    <>
      <PageSection>
        <Title headingLevel="h1" style={{ marginBottom: '0.5rem' }}>AI Asset Hub</Title>
        <p style={{ color: '#6a6e73' }}>Manage entity type schemas and browse catalog data.</p>
      </PageSection>

      {/* Schema Management Section */}
      <PageSection>
        <Title headingLevel="h2" style={{ marginBottom: '1rem' }}>
          <Icon style={{ marginRight: '0.5rem', verticalAlign: '-0.125em' }}><CubesIcon /></Icon>
          Schema Management
        </Title>
        <Gallery hasGutter minWidths={{ default: '300px' }}>
          <GalleryItem>
            <Card
              isClickable
              onClick={() => navigate('/schema')}
              style={{ cursor: 'pointer' }}
            >
              <CardHeader>
                <CardTitle>Entity Types &amp; Model</CardTitle>
              </CardHeader>
              <CardBody>
                Define entity types, attributes, associations, enums, and catalog versions. View the model diagram.
              </CardBody>
            </Card>
          </GalleryItem>
        </Gallery>
      </PageSection>

      <Divider />

      {/* Catalogs Section */}
      <PageSection>
        <Title headingLevel="h2" style={{ marginBottom: '1rem' }}>
          <Icon style={{ marginRight: '0.5rem', verticalAlign: '-0.125em' }}><DatabaseIcon /></Icon>
          Catalogs
        </Title>

        {loading && <Spinner aria-label="Loading catalogs" />}

        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

        {!loading && !error && catalogs.length === 0 && (
          <EmptyState>
            <EmptyStateBody>No catalogs available. Create one from Schema Management.</EmptyStateBody>
          </EmptyState>
        )}

        {catalogs.length > 0 && (
          <Gallery hasGutter minWidths={{ default: '300px' }}>
            {catalogs.map(catalog => (
              <GalleryItem key={catalog.id}>
                <Card
                  isClickable
                  onClick={() => navigate(`/catalogs/${catalog.name}`)}
                  style={{ cursor: 'pointer' }}
                >
                  <CardHeader>
                    <CardTitle>
                      <Icon style={{ marginRight: '0.5rem', verticalAlign: '-0.125em' }}><DatabaseIcon /></Icon>
                      {catalog.name}
                    </CardTitle>
                  </CardHeader>
                  <CardBody>
                    <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '0.5rem' }}>
                      <Label color={statusColor(catalog.validation_status)} isCompact>
                        {catalog.validation_status || 'unknown'}
                      </Label>
                      {catalog.published && (
                        <Label color="green" isCompact>Published</Label>
                      )}
                    </div>
                    {catalog.description && <p style={{ marginBottom: '0.5rem' }}>{catalog.description}</p>}
                    <p style={{ color: '#6a6e73', fontSize: '0.875rem' }}>
                      Version: {catalog.catalog_version_label || catalog.catalog_version_id}
                    </p>
                  </CardBody>
                </Card>
              </GalleryItem>
            ))}
          </Gallery>
        )}
      </PageSection>
    </>
  )
}
