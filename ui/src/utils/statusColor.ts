export const statusColor = (status: string) => {
  switch (status) {
    case 'draft': return 'blue'
    case 'valid': return 'green'
    case 'invalid': return 'red'
    default: return 'grey'
  }
}
