---
id: "mem_de11fc07"
topic: "Go import ordering for golangci-lint (goimports)"
tags:
  - go
  - lint
  - imports
  - gotcha
phase: 0
difficulty: 0.3
created_at: "2026-02-15T21:17:32.216117+00:00"
created_session: 1
---
## Import Ordering Rule

The `goimports` linter (configured in `.golangci.yml` with `local-prefixes: github.com/project-catalyst/pc-asset-hub`) requires three import groups separated by blank lines:

```go
import (
    // Group 1: Standard library
    "context"
    "time"

    // Group 2: Third-party packages
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "gorm.io/gorm"

    // Group 3: Local packages (matching local-prefixes)
    domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
    "github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)
```

Mixing third-party and local imports in the same group causes lint failures. This was the most frequent lint issue during Phase A — every new file needed its imports arranged correctly.
