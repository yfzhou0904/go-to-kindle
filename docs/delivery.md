# Delivery

Delivery persists the processed article under the application archive and sends
that HTML file as an SMTP attachment. `internal/repositories/` owns rendering
and file writes; `mail/` owns message construction and transport.

## Archive Lifecycle

Processing writes an initial archive file using the extracted title. On send,
`processAndSend` in `main.go` compares the reviewed filename with the initial
path. If the title changed, it writes the article under the new name and removes
the old archive file. Otherwise it refreshes the existing file.

Archive HTML includes the article title and Kindle-oriented styling around the
processed content. When enabled on the review screen, final rendering prepends
available published and updated dates plus the sent date in `YYYY/M/D` format.
The initial pre-review archive has no date line; a successful send rewrites it
to match the delivered attachment. An SMTP failure removes the unconfirmed
sent date from the archive. Files remain in the archive after successful
delivery.

## Email

`mail.SendEmailWithAttachment` builds a MIME email with the HTML file attached,
encodes the attachment content and filename for non-ASCII text, opens an
implicit TLS connection, authenticates with SMTP PLAIN, and sends to the
configured Kindle address. SMTP failures are returned to the TUI and do not
remove the local archive.

Configuration values are documented in [configuration.md](configuration.md).
Content construction is documented in [processing.md](processing.md).
