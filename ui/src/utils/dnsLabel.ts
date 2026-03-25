export const DNS_LABEL_RE = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/
export const isValidDnsLabel = (s: string) => DNS_LABEL_RE.test(s)
