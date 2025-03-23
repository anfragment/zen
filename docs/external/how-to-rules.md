# How to write rules

## FAQ

### How do I whitelist (allow) requests to a domain?

To whitelist requests to a domain, use the `@@` prefix.

For example, to whitelist Firefox's telemetry, use:

```plaintext
@@||incoming.telemetry.mozilla.org
```

> [!IMPORTANT]  
> Whitelisting does **not** prevent requests from being *proxied* by Zen. To exclude domains from proxying, use the **Ignored Hosts** feature in Settings.

## See also

Zen provides partial compatibility with popular filter list formats (EasyList, uBlock Origin, AdGuard, etc.), with near-complete compatibility targeted for the v1.0 release. Full documentation of supported features is a work in progress, but you can refer to the following resources for more information:

- [AdGuard – How to create your own ad filters](https://adguard.com/kb/general/ad-filtering/create-own-filters)
- [Adblock Plus – Filter cheatsheet](https://adblockplus.org/filter-cheatsheet)
