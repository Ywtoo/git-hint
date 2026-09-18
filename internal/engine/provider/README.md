# Providers

Providers turn semantic placeholders from imported command specifications into
runtime suggestions.

Examples include `<branch>`, `<commit>`, `<remote>`, `<file>`, and `<dir>`.

The Fig importer must either map a generator to a known placeholder/provider
or record it in `FIG_UNCONVERTED.md`; generators must never disappear silently.

