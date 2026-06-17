#!/bin/bash
# Semantic comparison of two YAML export outputs.
# Compares the structure and content of MCP Gateway CRs, ignoring:
#   - timestamps (exported-at annotations)
#   - exporter labels (mcp-gateway vs webhook-mcp-gateway)
#   - whitespace/formatting differences
#   - field ordering
#
# Usage:
#   scripts/compare-export-yaml.sh <file1.yaml> <file2.yaml>
#   scripts/compare-export-yaml.sh <(curl -s ...) <(curl -s ...)
#
# Exit code: 0 = semantically equal, 1 = differences found

set -uo pipefail

if [ $# -ne 2 ]; then
  echo "Usage: $0 <file1.yaml> <file2.yaml>"
  echo ""
  echo "Compares two YAML export outputs semantically, ignoring timestamps,"
  echo "exporter labels, whitespace, and field ordering."
  echo ""
  echo "Examples:"
  echo "  $0 built-in-export.yaml webhook-export.yaml"
  echo "  $0 <(curl -s -X POST -H 'X-User-Role: Admin' http://localhost:30080/api/data/v1/catalogs/my-cat/export-bindings/BIND1/run) \\"
  echo "     <(curl -s -X POST -H 'X-User-Role: Admin' http://localhost:30080/api/data/v1/catalogs/my-cat/export-bindings/BIND2/run)"
  exit 1
fi

FILE1="$1"
FILE2="$2"

python3 -c "
import sys, re

def parse_documents(text):
    '''Split multi-document YAML into individual documents.'''
    docs = []
    current = []
    for line in text.split('\n'):
        if line.strip() == '---':
            if current:
                docs.append('\n'.join(current))
            current = []
        else:
            current.append(line)
    if current:
        docs.append('\n'.join(current))
    return [d.strip() for d in docs if d.strip()]

def extract_fields(doc):
    '''Extract key fields from a YAML document into a comparable dict.'''
    fields = {}

    # Kind
    m = re.search(r'kind:\s*(\S+)', doc)
    if m: fields['kind'] = m.group(1)

    # Name (from metadata)
    m = re.search(r'name:\s*(\S+)', doc)
    if m: fields['name'] = m.group(1)

    # Namespace
    m = re.search(r'namespace:\s*(\S+)', doc)
    if m: fields['namespace'] = m.group(1)

    # apiVersion
    m = re.search(r'apiVersion:\s*(\S+)', doc)
    if m: fields['apiVersion'] = m.group(1)

    # Catalog label
    m = re.search(r'assethub\.io/catalog:\s*(\S+)', doc)
    if m: fields['catalog'] = m.group(1)

    # Spec fields (extract all key: value pairs under spec:)
    in_spec = False
    spec = {}
    spec_indent = 0
    for line in doc.split('\n'):
        stripped = line.lstrip()
        indent = len(line) - len(stripped)

        if stripped.startswith('spec:'):
            in_spec = True
            spec_indent = indent
            continue

        if in_spec:
            if indent <= spec_indent and stripped and not stripped.startswith('#'):
                in_spec = False
                continue
            # Collect spec content
            spec_line = stripped
            if spec_line.startswith('- '):
                # List item
                if 'tools' not in spec:
                    spec['tools'] = []
                spec['tools'].append(spec_line[2:].strip())
            elif ':' in spec_line and not spec_line.startswith('#'):
                key, _, val = spec_line.partition(':')
                key = key.strip()
                val = val.strip()
                if val and key not in ('group', 'kind', 'name'):
                    spec[key] = val
                elif key == 'name' and 'targetRef' in str(spec):
                    spec['targetRef.name'] = val
                elif not val:
                    # Nested object start
                    pass
                else:
                    spec[key] = val

    fields['spec'] = spec
    return fields

def unquote_yaml(val):
    '''Strip YAML quotes from a scalar value.'''
    if len(val) >= 2 and ((val[0] == '\"' and val[-1] == '\"') or (val[0] == \"'\" and val[-1] == \"'\")):
        return val[1:-1]
    return val

def normalize(fields):
    '''Remove fields that should be ignored in comparison.'''
    # Remove timestamp annotations
    fields.pop('exported-at', None)
    # Remove exporter label (differs by design)
    fields.pop('exporter', None)
    # Sort tools list if present
    if 'spec' in fields and 'tools' in fields['spec']:
        fields['spec']['tools'] = sorted(fields['spec']['tools'])
    # Strip YAML quotes from spec values
    if 'spec' in fields:
        for k, v in fields['spec'].items():
            if isinstance(v, str):
                fields['spec'][k] = unquote_yaml(v)
    return fields

def compare_docs(doc1, doc2):
    '''Compare two parsed documents. Returns list of differences.'''
    diffs = []
    all_keys = set(list(doc1.keys()) + list(doc2.keys()))

    for key in sorted(all_keys):
        v1 = doc1.get(key)
        v2 = doc2.get(key)
        if v1 != v2:
            if key == 'spec':
                # Deep compare spec
                spec_keys = set(list((v1 or {}).keys()) + list((v2 or {}).keys()))
                for sk in sorted(spec_keys):
                    sv1 = (v1 or {}).get(sk)
                    sv2 = (v2 or {}).get(sk)
                    if sv1 != sv2:
                        diffs.append(f'  spec.{sk}: {sv1!r} vs {sv2!r}')
            else:
                diffs.append(f'  {key}: {v1!r} vs {v2!r}')
    return diffs

# Read files
with open(sys.argv[1]) as f:
    text1 = f.read()
with open(sys.argv[2]) as f:
    text2 = f.read()

docs1 = parse_documents(text1)
docs2 = parse_documents(text2)

# Parse all documents
parsed1 = [normalize(extract_fields(d)) for d in docs1]
parsed2 = [normalize(extract_fields(d)) for d in docs2]

# Sort by kind+name for stable comparison
parsed1.sort(key=lambda d: (d.get('kind',''), d.get('name','')))
parsed2.sort(key=lambda d: (d.get('kind',''), d.get('name','')))

# Compare
if len(parsed1) != len(parsed2):
    print(f'DIFFERENT: {len(parsed1)} documents vs {len(parsed2)} documents')
    print(f'  File 1: {[d.get(\"kind\",\"?\") + \"/\" + d.get(\"name\",\"?\") for d in parsed1]}')
    print(f'  File 2: {[d.get(\"kind\",\"?\") + \"/\" + d.get(\"name\",\"?\") for d in parsed2]}')
    sys.exit(1)

has_diff = False
for i, (d1, d2) in enumerate(zip(parsed1, parsed2)):
    kind = d1.get('kind', '?')
    name = d1.get('name', '?')
    diffs = compare_docs(d1, d2)
    if diffs:
        has_diff = True
        print(f'DIFFERENT: {kind}/{name}')
        for diff in diffs:
            print(diff)
    else:
        print(f'EQUAL: {kind}/{name}')

if has_diff:
    sys.exit(1)
else:
    print('')
    print('All documents are semantically equal.')
    sys.exit(0)
" "$FILE1" "$FILE2"
