# TUI

The Bubble Tea model in `tui.go` presents the complete workflow and owns all
interactive state. Retrieval, processing, and email delivery run as commands so
the interface can show progress without blocking updates.

## Screens

1. **Input** accepts a URL or file path, offers clipboard Markdown, and exposes
   image and browser options.
2. **Retrieval** waits for the selected source to be loaded.
3. **Post-processing** extracts and prepares the article.
4. **Edit** shows source, language, word count, and image count while allowing
   the title to be changed. A default-on checkbox previews and controls the
   published, updated, and sent date line.
5. **Sending** archives the final title and emails the attachment.
6. **Completion** reports success or an error.

Completion messages carry the results of asynchronous commands back into the
state machine. Errors remain part of the model and are rendered on the relevant
screen.

## Navigation

Tab and arrow keys move among controls, Space toggles binary options, and Enter
activates the clipboard action, submits a non-empty URL/file value, or sends
from the review screen.
Escape from the edit screen returns to the existing input configuration rather
than creating a fresh model. `Ctrl+C` exits from any screen.

## State Invariants

- CLI-prefilled input and debug state survive returning from review.
- Browser configuration is hidden when it does not apply to clipboard input.
- Changing the title updates both the article metadata and eventual archive
  filename.
- Date context is enabled for each new article and may be disabled per send.
- The sent timestamp is captured once when the review screen is submitted.
- Terminal width controls wrapping but does not change workflow state.

See [architecture.md](architecture.md) for module boundaries and
[delivery.md](delivery.md) for the final command.
