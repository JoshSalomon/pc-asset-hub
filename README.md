# AI Asset Hub

> **Work in Progress.** This project is under active development and currently runs only on local [kind](https://kind.sigs.k8s.io/) clusters. It is not yet ready for production use. This README will be updated as the project matures.

AI Asset Hub is a metadata-driven management system for AI assets deployed on OpenShift clusters. It is a component of [Project Catalyst](https://github.com/project-catalyst).

The system manages assets such as models, MCP servers, tools, guardrails, evaluators, and prompts — but the list of asset types is not hardcoded. Entity types, their attributes, and the associations between them are defined dynamically through a configuration layer, making the system extensible to any asset type without code changes.

## Getting Started

See [DEPLOYMENT.md](DEPLOYMENT.md) for full deployment instructions.

Quick start on a local kind cluster:

```bash
./scripts/kind-deploy.sh
```

This builds all images, creates a kind cluster, and deploys the full stack. Once complete:

- **API server:** http://localhost:30080
- **UI:** http://localhost:30000

## Documentation

| Document | Description |
|----------|-------------|
| [PRD.md](PRD.md) | Product requirements, user stories, and design decisions |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Deployment guide for kind clusters |
| [docs/architecture.md](docs/architecture.md) | System architecture, data model, and technology stack |

## License

See [LICENSE](LICENSE).
