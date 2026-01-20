# MissionControl Web UI

React-based dashboard for MissionControl agent orchestration.

## Development

```bash
# Install dependencies
npm install

# Start dev server (connects to orchestrator at localhost:8080)
npm run dev

# Run tests
npm test

# Run E2E tests (requires running orchestrator)
npm run test:e2e

# Build for production
npm run build

# Lint
npm run lint
```

## Architecture

- **Framework**: React 18 + TypeScript + Vite
- **Styling**: Tailwind CSS
- **State**: Zustand stores
- **Testing**: Vitest + Playwright

## Key Components

- `KingChat` - Chat interface for King mode
- `AgentCard` - Individual agent status display
- `ZoneGroup` - Agent grouping by zone
- `ProjectWizard` - New project setup flow
- `WorkflowMatrix` - Persona/workflow configuration

## Stores

- `useStore` - Main app state (agents, zones, messages)
- `useMissionStore` - V5 mission state
- `useWorkflowStore` - Workflow phase tracking
- `useKnowledgeStore` - Token/knowledge management

## WebSocket Events

The UI connects to `/ws` and handles events including:
- `agent_spawned`, `agent_stopped` - Agent lifecycle
- `king_message`, `king_question` - King communication
- `tokens_updated` - Usage tracking
- `zone_created`, `zone_updated`, `zone_deleted` - Zone management
