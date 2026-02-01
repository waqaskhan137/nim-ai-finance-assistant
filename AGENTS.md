# AGENTS.md - Nim Go SDK

Guidelines for AI agents working on this codebase.

## Project Overview

**nim-go-sdk** is a Go SDK for building AI-powered financial assistants using Anthropic's Claude AI. It implements agentic patterns from [Building Effective Agents](https://www.anthropic.com/engineering/building-effective-agents) including tool orchestration, sub-agents, and confirmation workflows.

## Build/Lint/Test Commands

### Go Backend
```bash
go build ./...                              # Build SDK
go fmt ./...                                # Format code
go vet ./...                                # Lint for issues
go mod tidy                                 # Clean dependencies
go test ./...                               # Run all tests
go test -v ./...                            # Verbose test output
go test -run TestFunctionName ./pkg/...     # Run single test
go test -run TestFoo -count=1 ./pkg/...     # Skip cache, run once
go test -race ./...                         # Race condition detection
go test -cover ./...                        # Coverage report

# Hackathon starter server
cd examples/hackathon-starter
go build -o server . && ./server            # Build and run
air                                         # Live reload (if installed)
```

### Frontend (examples/hackathon-starter/frontend)
```bash
npm install                                 # Install dependencies
npm run dev                                 # Dev server (port 5173)
npm run build                               # Production build
npm run preview                             # Preview production build
npx tsc --noEmit                            # Type check only
npx eslint src/                             # Lint (if configured)
```

## Code Style

### Go Imports (grouped with blank lines)
```go
import (
    "context"
    "encoding/json"
    
    "github.com/anthropics/anthropic-sdk-go"
    
    "github.com/becomeliminal/nim-go-sdk/core"
    "github.com/becomeliminal/nim-go-sdk/tools"
)
```

### Go Naming
| Type | Convention | Example |
|------|------------|---------|
| Packages | lowercase, single word | `trading`, `tools`, `engine` |
| Exported types | PascalCase | `SpendingInsight`, `ToolBuilder` |
| Unexported | camelCase | `parseTransaction`, `buildSchema` |
| Interfaces | Noun or -er suffix | `Agent`, `ToolHandler`, `AuditLogger` |
| JSON tags | snake_case | `json:"total_spent"` |

### Go Error Handling
```go
// Always wrap errors with context
if err != nil {
    return nil, fmt.Errorf("failed to parse transaction: %w", err)
}
// Return errors, never panic in library code
```

### Go Tool Definitions (fluent builder pattern)
```go
tools.New("tool_name").
    Description("What the tool does").
    Schema(tools.ObjectSchema(map[string]interface{}{
        "param": tools.StringProperty("Description"),
    }, "param")).
    RequiresConfirmation().  // For write operations
    HandlerFunc(handler).
    Build()
```

### TypeScript/React
```tsx
// Import order: React, external, local, styles
import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useToast } from './Toast'
import './PageName.css'

// Naming: Components=PascalCase, hooks=usePrefix, handlers=handlePrefix
// CSS classes: kebab-case (BEM-lite)
// Always define interfaces for API responses
interface ApiResponse { success: boolean; data: SomeType }
```

## Architecture

### Key Directories
| Path | Purpose |
|------|---------|
| `core/` | Interfaces: Agent, Tool, Context, types |
| `engine/` | Claude API integration, agentic loop, tool execution |
| `tools/` | Tool builder, schema helpers, Liminal tools |
| `subagent/` | Sub-agent framework, delegation patterns |
| `server/` | WebSocket server, streaming protocol |
| `store/` | Conversation persistence, confirmation storage |
| `examples/hackathon-starter/` | Full demo with trading tools |

### Agentic Patterns (from Anthropic's guidance)
1. **Augmented LLM**: Tools + retrieval + memory (see `engine/engine.go`)
2. **Prompt Chaining**: Sequential LLM calls with gates
3. **Routing**: Classify input, delegate to specialized handlers
4. **Orchestrator-Workers**: Main agent delegates to sub-agents (see `subagent/`)
5. **Evaluator-Optimizer**: Generate + evaluate in loop

### Tool Design Principles
- Give model enough context to use tools correctly
- Clear parameter names and descriptions (like good docstrings)
- Test tool usage extensively before deployment
- Make it hard to misuse (poka-yoke your tools)

## Frontend Design (Apple HIG)

Follow Apple Human Interface Guidelines for native-feeling UI:
- **Touch targets**: Minimum 44x44 points
- **Typography**: System fonts, support Dynamic Type
- **Colors**: Use semantic colors (`--label-primary`, `--tint-color`)
- **Dark Mode**: Full support required, test both themes
- **Feedback**: Loading states, toast notifications, animations
- **Spacing**: 8-point grid system

### CSS Variables
```css
/* Use semantic colors, not hardcoded values */
color: var(--label-primary);      /* Text */
color: var(--label-secondary);    /* Subtle text */
background: var(--grouped-bg);    /* Card backgrounds */
color: var(--positive-color);     /* Financial gains only */
color: var(--tint-color);         /* Interactive elements (blue) */
```

## Environment Variables
```bash
ANTHROPIC_API_KEY=sk-ant-...
LIMINAL_API_URL=https://api.liminal.cash
VITE_API_URL=http://localhost:8081
VITE_WS_URL=ws://localhost:8081/ws
```

## Common Patterns

**Demo Mode**: Check `trading.DemoMode` before real API calls
**Confirmation Flow**: Write operations pause for user approval
**Sub-agents**: Restricted capabilities, no confirmation rights
**Streaming**: Use `StreamCallback` for real-time responses
**Audit Logging**: All tool executions logged via `AuditLogger`

## Anti-Patterns to Avoid

- Custom controls when system controls exist
- Hardcoded colors instead of CSS variables
- Missing loading/error states
- Panic instead of error returns
- Skipping confirmation for write operations
- Overly complex agent architectures (start simple)
