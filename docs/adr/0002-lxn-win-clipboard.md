# Replace Wails clipboard with lxn/win for richer format support and event-driven capture

jPaste originally used Wails v3's built-in `app.Clipboard.Text()` / `SetText()` which only handles `CF_UNICODETEXT`. To support multi-format clipboard (HTML, RTF, images), capture source application identity, and eliminate polling in favor of `WM_CLIPBOARDUPDATE` event-driven monitoring, we replaced the Wails clipboard layer with direct Win32 API calls via `github.com/lxn/win`.

## Considered Options

- **Wails v3 built-in clipboard API**: Only exposes plain text (`CF_UNICODETEXT`), cannot enumerate available formats, cannot detect clipboard owner. Polling-based read with 1-second intervals loses data during rapid copy sequences. Rejected for all three functional gaps.

- **lxn/win with AddClipboardFormatListener on Wails HWND** (reuse the WebView2 host window): Zero window overhead, but requires extracting Wails v3 alpha's internal HWND — a fragile dependency on an unstable API. Rejected because the project already has a proven pattern for message-only windows (used by the notification service).

- **lxn/win with independent message-only window** (chosen): A hidden `HWND_MESSAGE` window with `AddClipboardFormatListener` + `GetMessage` loop, communicating changes via Go channel. Completely decoupled from Wails internals, identical pattern to the existing notification module.
