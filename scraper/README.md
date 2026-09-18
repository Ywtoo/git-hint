# Scraper

The scraper is a fallback and maintenance tool, not the primary source of
command knowledge.

- `parser/`: parses help text into the internal command tree.
- `crawler.go`, `exec.go`: crawls external commands on demand.
- `builtin.go`, `man_help.go`, `bash_help.go`: builtin documentation fallback.
- `discover.go`, `registry.go`: discovers commands and builds `index.json`.
- `writer.go`: writes generated command data.

Fig data is imported offline by `tools/fig/importer` and should take priority
over crawler output whenever both exist.
