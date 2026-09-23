# Preliminary plan

## crux

Capture the page Safari has already rendered rather than request the article again from a separate host. This preserves the phone's network origin and, where iOS exposes the rendered page, its dynamic and signed-in browsing context. Delivery ends in the system mail compose screen: the user reviews the addressed message and taps Send.

This is preliminary planning. The first prototype must validate Safari page capture and attachment handoff before the native product structure is treated as settled.

## product flow

1. The reader invokes Go to Kindle from Safari's share menu.
2. The share experience captures the page URL, title, and rendered article material.
3. Extraction presents a concise review screen with an editable title, image choice, and date-context choice.
4. Processing creates the same class of self-contained, Kindle-oriented HTML document described in `docs/processing.md` and `docs/delivery.md`.
5. The system mail composer opens with the configured Kindle address, article title, and HTML attachment populated.
6. The reader taps Send or cancels. Prepared documents may be retained locally for preview or retry, subject to a later product decision.

## platform prototype

Create an iOS proof of concept, expected under `ios/`, with a containing app and Safari-visible share extension. Before building the full interface, prove these boundaries against a representative fixture set:

- Safari can pass the current URL, title, and useful rendered page content to the extension.
- Readability extraction can operate on that rendered content without refetching the page.
- The extension can generate an HTML file and present it as an attachment in the system mail composer.
- Cancel and send-completion states can be represented honestly; opening the composer is not itself proof of delivery.
- Typical article sizes fit within extension execution and memory constraints.

Test public, JavaScript-heavy, signed-in, lazy-loaded, and image-heavy pages. Record failures by capability rather than adding site-specific workarounds during the prototype.

## native app responsibilities

The containing app is expected to own onboarding and durable preferences, including the Kindle destination address and processing defaults. It should also explain the approved-sender requirement for Kindle email delivery.

The share experience is expected to own capture, extraction status, review, document generation, and mail-composer presentation. Shared processing and document models should live in a module usable by both targets rather than in extension UI code.

Exact Xcode targets, package boundaries, persistence choice, and minimum iOS version remain to be decided after the prototype.

## capability mapping

Use the existing implementation as the behavioral reference, not as a requirement to embed Go on iOS:

- `input.go:10-24` defines the normalized HTML/Markdown distinction.
- `retrieval.go:153-172` connects normalized input to article and Markdown processing.
- `postprocessing/postprocessing.go:31-75` defines readability extraction, naming, and cleanup sequencing.
- `postprocessing/postprocessing.go:78-136` makes image acquisition an explicit resolver boundary; the iOS equivalent must tolerate images that cannot be downloaded outside Safari's session.
- `postprocessing/postprocessing.go:139-179` documents media cleanup and link behavior to compare in native output.
- `docs/delivery.md` defines final HTML/date behavior. Native delivery diverges intentionally by handing the attachment to the system composer instead of calling `mail/mail.go:17-133` directly.

Candidate extraction technology, including bundled Readability JavaScript versus a native implementation, should be selected from prototype results. Reusing fixtures and expected behaviors is preferred over forcing the current Go dependency graph into an iOS extension.

## images and degraded behavior

Image support requires a separate validation pass. Page markup may expose image URLs without exposing authenticated bytes to the extension. The first implementation should distinguish:

- embedded or publicly downloadable images that can be resized and included;
- images unavailable outside Safari's cookie or page execution context;
- lazy content that was not loaded when sharing began.

Failed images must not block a readable text document. The review/result UI should report omissions succinctly. Capturing credentials or attempting to export Safari's session is out of scope.

## tests

- Build a native fixture corpus from `testdata/` and relevant cases in `postprocessing/*_test.go`.
- Add extraction tests for title, body, dates, links, and malformed input.
- Add rendering snapshots or structural assertions for Kindle-oriented HTML.
- Exercise attachment filename and non-ASCII title handling.
- Manually test the share flow on a physical iPhone; simulator-only results are insufficient for Safari and Mail integration.
- Verify cancel, unavailable Mail account, and attachment-generation failure paths.

## verification

The eventual PRs should record exact Xcode build and test commands once the project and scheme names exist. Each user-facing milestone also requires a physical-device smoke test from Safari share invocation through presentation of the populated mail composer.

No Go checks are required for this planning-only PR.

## decided

- The target experience is a native iPhone app available from Safari's share menu.
- Processing should prefer Safari's already-rendered page over remote scraping.
- Delivery uses the system mail compose screen; the user explicitly taps Send.
- No VPS or awake-Mac dependency is part of the desired flow.

## open questions

- Minimum supported iOS version and deployment strategy.
- Whether the initial release includes images or ships text-first.
- Whether prepared documents and canceled drafts appear in an in-app archive.
- Whether Markdown/text/file sharing belongs in the first milestone.
- Which extraction approach best survives real Safari pages within extension limits.
- What completion language is accurate given the mail composer's available callbacks.

## out of scope

- Implementing the iOS app in this planning PR.
- Silent SMTP delivery or SMTP credential storage.
- A hosted fetch service or Mac relay.
- Desktop CLI behavior changes.
- Site-specific extraction adapters before the general prototype is evaluated.
