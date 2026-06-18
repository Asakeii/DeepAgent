# deepAgent

`deepAgent` is a learning-oriented Go project skeleton inspired by the Eino-based `deer-go` agent flow.

## Planned Flow

```text
Console Input
  -> Planner
  -> ResearchTeam
  -> Researcher / Coder
  -> Reporter
```

## Directory Guide

- `conf/`: YAML configuration.
- `internal/model/`: shared state and plan models.
- `internal/infra/`: model, MCP, prompt, and output infrastructure.
- `internal/agent/`: Eino graph and agent nodes.
- `internal/prompts/`: prompt templates.
- `mcps/python/`: Python MCP server placeholder.
- `docs/rebuild-deer-go-guide.md`: step-by-step guide for rebuilding the `deer-go` design in this project.

## Run

```bash
cp conf/deep-agent.yaml.example conf/deep-agent.yaml
./run.sh
```
