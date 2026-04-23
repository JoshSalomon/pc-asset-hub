import {
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Alert,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import type { MigrationReport } from '../types'

interface Props {
  isOpen: boolean
  report: MigrationReport
  entityTypeName: string
  onConfirm: () => void
  onCancel: () => void
}

export default function MigrationPreviewModal({ isOpen, report, entityTypeName, onConfirm, onCancel }: Props) {
  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={onCancel}>
      <ModalHeader title="Migration Preview" />
      <ModalBody>
        <p style={{ marginBottom: '1rem' }}>
          Changing the pinned version of <strong>{entityTypeName}</strong> will
          affect <strong>{report.affected_instances}</strong> instance(s)
          in <strong>{report.affected_catalogs}</strong> catalog(s):
        </p>
        {report.catalog_breakdown && report.catalog_breakdown.length > 0 && (
          <ul style={{ marginBottom: '1rem', marginTop: 0 }}>
            {report.catalog_breakdown.map(ci => (
              <li key={ci.catalog_name}><strong>{ci.catalog_name}</strong>: {ci.instance_count} instance(s)</li>
            ))}
          </ul>
        )}

        {report.warnings.length > 0 && (
          <div style={{ marginBottom: '1rem' }}>
            {report.warnings.map((w, i) => (
              <Alert
                key={i}
                variant="warning"
                isInline
                title={warningTitle(w.type, w.attribute, w.old_type, w.new_type)}
                style={{ marginBottom: '0.5rem' }}
              >
                Affects {w.affected_instances} instance(s)
              </Alert>
            ))}
          </div>
        )}

        {report.attribute_mappings.length > 0 && (
          <Table variant="compact" aria-label="Attribute mappings">
            <Thead>
              <Tr>
                <Th>Old Attribute</Th>
                <Th>New Attribute</Th>
                <Th>Action</Th>
              </Tr>
            </Thead>
            <Tbody>
              {report.attribute_mappings.map((m, i) => (
                <Tr key={i}>
                  <Td>{m.old_name || '—'}</Td>
                  <Td>{m.new_name || '—'}</Td>
                  <Td>{m.action}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}

        {report.affected_instances === 0 && report.warnings.length === 0 && (
          <p>No instance data will be affected by this change.</p>
        )}
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={onConfirm}>Apply Change</Button>
        <Button variant="link" onClick={onCancel}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}

function warningTitle(type: string, attribute: string, oldType?: string, newType?: string): string {
  switch (type) {
    case 'deleted_attribute':
      return `Attribute "${attribute}" removed — existing values will be orphaned`
    case 'type_changed':
      return `Attribute "${attribute}" type changed (${oldType} → ${newType}) — existing values may be invalid`
    case 'new_required':
      return `New required attribute "${attribute}" — existing instances have no value`
    case 'renamed':
      return `Attribute renamed: "${oldType}" → "${newType}" (matched by position)`
    default:
      return `${type}: ${attribute}`
  }
}
