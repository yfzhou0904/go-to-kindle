# Documentation

These documents explain how to use, change, and understand go-to-kindle. The root [README](../README.md) is a concise entry point for users and coding agents; details live here.

## Start Here

- [Architecture](architecture.md): end-to-end flow and module boundaries.
- [Usage](usage.md): installation, email setup, controls, and troubleshooting.
- [Development](development.md): project layout and contributor workflow.
- [Input](input.md): supported sources and input normalization.
- [Retrieval](retrieval.md): direct HTTP and headless-browser fetching.
- [Processing](processing.md): HTML, Markdown, images, and output naming.
- [TUI](tui.md): screens, navigation, and state transitions.
- [Delivery](delivery.md): local archives and SMTP delivery.
- [Configuration](configuration.md): config lifecycle and runtime paths.
- [Testing](testing.md): test organization, boundaries, and commands.

## Writing Guidelines

Use progressive disclosure: `architecture.md` should give a new contributor the system model in a few minutes, while module pages go deeper on one concern.

Each module document should cover only what is useful:

1. Purpose and ownership.
2. Important concepts or inputs.
3. Lifecycle, decisions, and invariants.
4. Integration points using real source paths.
5. Links to related documents.

Avoid copying structs, function bodies, or details that are easier to discover in code. Describe behavior that exists today, not planned features. Target fewer than 100 lines per document; split a topic only after a focused page can no longer remain concise.
