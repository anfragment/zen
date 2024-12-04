# `logger`

## Why We Need Different Setups for Production and Non-Production Builds on Windows

When running `wails build`, Wails builds the app with the [`-H windowsgui` linker flag](https://pkg.go.dev/cmd/link). This sets the [\SUBSYSTEM PE header variable](https://learn.microsoft.com/en-us/cpp/build/reference/subsystem-specify-subsystem?view=msvc-170&redirectedfrom=MSDN) to `WINDOWS`. Processes configured with this value have their standard I/O handles [closed by default](https://learn.microsoft.com/en-us/windows/console/getstdhandle?redirectedfrom=MSDN#remarks). However, during `wails dev`, the I/O handlers function as expected.

In development mode, we use `io.MultiWriter` to write logs to both `os.Stdout` and a file logger. The `io.MultiWriter` function [synchronously writes to each of its children](https://go.dev/src/io/multi.go#L83), meaning that if any single child hangs, the entire operation hangs. This becomes problematic in production because the closed `os.Stdout` handle will cause each log operation to hang indefinitely.

To avoid this issue, we log **exclusively to the filesystem in production**.
