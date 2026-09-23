# Send Safari articles to Kindle from iPhone

## context

- [`README.md`](../../README.md) describes the current Mac CLI workflow and supported content.
- [`docs/input.md`](../../docs/input.md) describes the inputs accepted by the processing pipeline.
- [`docs/delivery.md`](../../docs/delivery.md) describes the generated HTML attachment and Kindle delivery flow.

## issue

The tool works flawlessly on a MacBook, but it is not available from iPhone Safari's share menu. A hosted version is not an equivalent substitute: many websites block requests from data-center IP addresses, and a Mac normally sleeping in a bag cannot act as an always-on relay.

As a result, sending an article encountered on iPhone currently requires leaving the phone workflow or using a different conversion path.

## improvement

A reader browsing an article in Safari should be able to:

1. Open Safari's share menu and choose Go to Kindle.
2. Have the content of the page they are viewing prepared as a Kindle-friendly document, including pages that rendered dynamically or depend on the reader's signed-in Safari session where the platform permits it.
3. Review the extracted title and relevant options before continuing.
4. See a pre-addressed mail compose screen with the prepared document attached.
5. Tap Send themselves, retaining explicit control over delivery.

The experience should retain as much of the existing content quality as the iPhone platform allows, degrade clearly when page content or images cannot be captured, and avoid requiring an always-on Mac or hosted scraping service.

## out of scope

- Unattended or silent email delivery.
- A remotely hosted article-scraping service.
- Keeping the Mac awake as a relay.
- Guaranteeing extraction of DRM media, inaccessible cross-origin frames, or content Safari has not loaded.
