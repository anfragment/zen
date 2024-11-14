# Code conventions
This document outlines key code conventions followed in this project.

## Go
1. __Avoid utility packages__ <br />
  Do not use generic utility package names like `base`, `util`, or `common`. For more context, see [Dave Cheney's post explaining why](https://dave.cheney.net/2019/01/08/avoid-package-names-like-base-util-or-common).
2. __Error wrapping__ <br />
  Errors should __always__ be wrapped with `fmt.Errorf` with the `%w` directive. The wrapped message must provide enough context to help identify the error’s source at a glance. Avoid prefixes like "failed to" or "error" in messages, as they add no meaningful context.
    - __Bad Example__ <br />
      A generic message that doesn’t provide specific context:
      ```go
      if _, err := os.Create(configFileName); err != nil {
        return fmt.Errorf("failed to create file: %v", err)
      }
      ```
    - __Good Example__ <br />
    A more specific message that clarifies the error’s context:
      ```go
      if _, err := os.Create(configFileName); err != nil {
        return fmt.Errorf("create config file: %w", err)
      }
      ```
3. __Struct constructors__ <br />
  Struct constructors are specifically created for use within `app.go` and `main.go`. They may initialize dependency fields with meaningful defaults and use static variables from other packages. An implication of this is that __public tests__ should use constructors to initialize structs, and __private tests__ may create a struct manually.

### See also
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
