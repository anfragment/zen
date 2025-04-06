# Changelog

## v0.9.0

### What's New

- **Multi-language support**: Zen now supports multiple languages, with more on the way. You can switch your preferred language in the settings. Huge thanks to @kamalovk for laying the groundwork for this feature.
- **Background self-updates**: Zen can now check for and apply updates automatically in the background at startup. You can enable this behavior in the settings.
- **Minimized startup**: When autostart is enabled on Windows, Zen now launches minimized to the system tray - keeping things quiet until you need them. Thanks to @Zanphar for the suggestion.
- **Scriptlet enhancements**: Numerous improvements to scriptlets, including new additions and stability upgrades to existing ones.
- **Internal filtering engine improvements**: The filtering engine now supports precise exceptions, which allows for more unwanted content to be blocked.
- **Higher resolution icons on Windows**: Zen now features sharper, high-resolution icons on Windows, thanks to @TobseF.
- **ARM64 builds for Linux**: Native ARM64 builds are now available for Linux users.
- **System proxy configuration via PAC**: Zen now configures the system proxy using a PAC file, resolving issues with networking in built-in Windows apps and improving overall security.
- **Join our Discord community**: We've launched a Discord server! Come say hi, share tips, and stay up to date with the latest on Zen: <https://discord.gg/jSzEwby7JY>. You'll also find the link on our website: <https://zenprivacy.net>.

### New Contributors

- @kamalovk made their first contribution: <https://github.com/ZenPrivacy/zen-desktop/pull/269>
- @TobseF made their first contribution: <https://github.com/ZenPrivacy/zen-desktop/pull/267>

**Full Changelog**: <https://github.com/ZenPrivacy/zen-desktop/compare/v0.8.0...v0.9.0>

## v0.8.0

### What's New

- **Performance Improvements**: We rewrote our proxy so that it no longer waits for the entire response before starting to pass data to the browser. Expect 1.5–2× improvements in page download times.
- Minor enhancements to content blocking and privacy preservation.

Thank you for using Zen!

**Full Changelog**: <https://github.com/ZenPrivacy/zen-desktop/compare/v0.7.2...v0.8.0>

## v0.7.2

### What's New

- **Character Encoding Fix**: Improved character encoding detection to handle websites with non-standard encodings more gracefully. Many thanks to @2372281891 for reporting the issue.

Thank you for using Zen!

**Full Changelog**: <https://github.com/ZenPrivacy/zen-desktop/compare/v0.7.1...v0.7.2>

## v0.7.1

### What's New

- **Navigator API Bug Fix**: Resolved a critical issue that impacted the stability of websites using the Navigator API.

Thank you for using Zen!

**Full Changelog**: <https://github.com/ZenPrivacy/zen-desktop/compare/v0.7.0...v0.7.1>

## v0.7.0

### What's New

- **Cosmetic Filtering**: Annoying and intrusive elements on webpages are now automatically blocked for a cleaner browsing experience.
- **JavaScript Rule Injection**: JS rules expand on scriptlets and offer advanced ad-blocking and privacy-preserving capabilities in the most complex cases.
- **Windows System Tray Icon Stability**: Resolved an issue where the tray icon could become unresponsive after prolonged use on Windows.
- Various stability improvements and bug fixes.

Happy 2025 and thank you for using Zen!

**Full Changelog**: <https://github.com/ZenPrivacy/zen-desktop/compare/v0.6.1...v0.7.0>

## v0.6.0

### What's New

- **Scriptlets**: Introducing scriptlets—advanced ad-blocking tool designed to handle cases where regular filtering is insufficient.
  - **First-Party Self-Update**: We've completely rewritten our self-updating system for improved stability. Future macOS updates will now be delivered seamlessly without requiring a reinstallation of the app. Special thanks to @AitakattaSora for implementing this feature.
  - **Custom Filter List Backup**: Advanced users can now easily back up and restore their custom filter lists. Many thanks to @Noahnut for your contribution.
  - **Rules Editor**: A new tab in the app allows you to add custom filter rules directly inside the app.
  - **Export Application Logs**: Logs are now written to disk, making it easier for the development team to diagnose and resolve issues. Thank you to @AitakattaSora for implementing this feature.
  - **Improved Linux Support**: The app now starts without errors on non-GNOME systems. You can now manually configure the HTTP proxy on a per-app basis if needed. Thanks to @AitakattaSora for this enhancement.
  - **Improved Windows Support**: The app now shuts down gracefully and resets the system proxy during system shutdown, preventing internet disruptions at startup.
  - Various stability improvements and bug fixes.
  
Warning: On macOS, the app will not function properly after the update. Please visit our homepage, [zenprivacy.net](https://zenprivacy.net), to manually download the latest version. Future updates will be delivered seamlessly.

Thank you for using Zen!

**Full Changelog**: <https://github.com/ZenPrivacy/zen-desktop/compare/v0.5.0...v0.6.0>
