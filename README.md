<p align="center">
  <picture>
    <img src="https://github.com/anfragment/zen/blob/master/assets/appicon.png?raw=true" alt="Zen's Blue Shield Logo" width="150" />
  </picture>
</p>

<h3 align="center">
  Zen: Your Comprehensive Ad-Blocker and Privacy Guard
</h3>
<blockquote align="center">
There is, simply, no way, to ignore privacy. Because a citizenry’s freedoms are interdependent, to surrender your own privacy is really to surrender everyone’s.

Edward Snowden, Permanent Record
</blockquote>

![GitHub License](https://img.shields.io/github/license/anfragment/zen)
![GitHub release](https://img.shields.io/github/v/release/anfragment/zen)
![GitHub download counter](https://img.shields.io/github/downloads/anfragment/zen/total)

Zen is an open-source system-wide ad-blocker and privacy guard for Windows, macOS, and Linux. It works by setting up a proxy that intercepts HTTP requests from all applications, and blocks those serving ads, tracking scripts that monitor your behavior, malware, and other unwanted content. By operating at the system level, Zen can protect against threats that browser extensions cannot, such as trackers embedded in desktop applications and operating system components. Zen comes with many pre-installed filters, but also allows you to easily add hosts files and EasyList-style filters, enabling you to tailor your protection to your specific needs.

## Downloads

During the first run, Zen will prompt you to install a root certificate. This is required for Zen to be able to intercept and modify HTTPS requests. This certificate is generated locally and never leaves your device.

### Windows

- x64: [💾 Installer](https://github.com/anfragment/zen/releases/latest/download/Zen-amd64-installer.exe) | [📦 Portable](https://github.com/anfragment/zen/releases/latest/download/Zen_windows_amd64.zip)
- ARM64: [💾 Installer](https://github.com/anfragment/zen/releases/latest/download/Zen-arm64-installer.exe) | [📦 Portable](https://github.com/anfragment/zen/releases/latest/download/Zen_windows_arm64.zip)

Unsure which version to download? Click on 'Start' and type 'View processor info'. The 'System type' field under 'Device specifications' will tell you which one you need.

### macOS

- x64 (Intel): [💾 Installer](https://github.com/anfragment/zen/releases/latest/download/Zen-amd64.dmg) | [📦 Portable](https://github.com/anfragment/zen/releases/latest/download/Zen_darwin_amd64.tar.gz)
- ARM64 (Apple Silicon): [💾 Installer](https://github.com/anfragment/zen/releases/latest/download/Zen-arm64.dmg) | [📦 Portable](https://github.com/anfragment/zen/releases/latest/download/Zen_darwin_arm64.tar.gz)

Unsure which version to download? Learn at [Apple's website](https://support.apple.com/en-us/HT211814).

### Linux

- AUR: [👾 zen-adblocker-bin](https://aur.archlinux.org/packages/zen-adblocker-bin)
- x64: [📦 Portable](https://github.com/anfragment/zen/releases/latest/download/Zen_linux_amd64.tar.gz)

On Linux, automatic proxy configuration is currently only supported on GNOME-based desktop environments.

## Screenshots

<table>
  <thead>
    <tr>
        <th>Request history</th>
        <th>Filter list manager</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>
        Request history shows all requests blocked by Zen. Each request can be inspected to see which filter and rule blocked it.
      </td>
      <td>
        Zen comes with many pre-installed filters. You can also add your own by providing a URL to a hosts file or an EasyList-style filter.
      </td>
    </tr>
    <tr>
      <td align="center" valign="top"><img src="https://github.com/anfragment/zen/blob/master/assets/screenshots/main-window.png?raw=true" alt="Zen's main window"/></td>
      <td align="center" valign="top"><img src="https://github.com/anfragment/zen/blob/master/assets/screenshots/filter-lists.png?raw=true" alt="Zen's filter list manager"/></td>
    </tr>
  </tbody>
</table>

## Development

Follow the [getting started guide](docs/internal/index.md#getting-started) to begin working on Zen development. If you have any questions, feel free to ask in the [Discussions](https://github.com/anfragment/zen/discussions/categories/q-a).

## Contributing

Zen needs your help! You can file bug reports, suggest features, and submit pull requests. Please refer to the [Contributing Guidelines](CONTRIBUTING.md) for more information.

## License

This project is licensed under the [MIT License](https://github.com/anfragment/zen/blob/master/LICENSE). Some code and assets included with Zen are licensed under different terms. For more information, see the [COPYING](https://github.com/anfragment/zen/blob/master/COPYING.md) file.
