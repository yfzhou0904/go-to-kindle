# Documentation

These documents explain the current technical behavior of go-to-kindle. The
user-facing installation and usage guide remains in the repository
[README](../README.md).

## Start Here

- [Architecture](architecture.md): end-to-end flow and module boundaries.
- [Input](input.md): supported sources and input normalization.
- [Retrieval](retrieval.md): direct HTTP and headless-browser fetching.
- [Processing](processing.md): HTML, Markdown, images, and output naming.
- [TUI](tui.md): screens, navigation, and state transitions.
- [Delivery](delivery.md): local archives and SMTP delivery.
- [Configuration](configuration.md): config lifecycle and runtime paths.
- [Testing](testing.md): test organization, boundaries, and commands.

## Writing Guidelines

Use progressive disclosure: `architecture.md` should give a new contributor
the system model in a few minutes, while module pages go deeper on one concern.

Each module document should cover only what is useful:

1. Purpose and ownership.
2. Important concepts or inputs.
3. Lifecycle, decisions, and invariants.
4. Integration points using real source paths.
5. Links to related documents.

Avoid copying structs, function bodies, or details that are easier to discover
in code. Describe behavior that exists today, not planned features. Target
fewer than 100 lines per document; split a topic only after a focused page can
no longer remain concise.

