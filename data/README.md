# Command data

This directory contains the command specifications used by git-hint.

The JSON files are the editable, canonical output of the Fig importer. They
are embedded into the binary as a safe default, but local files next to the
installed binary may override them when a user wants to customize text or
completion behavior.

`index.json` is runtime state and is generated from commands available in the
current machine (`PATH` plus shell builtins). It is not a command
specification and should not be imported from Fig.

The runtime resolution order is:

1. embedded/editable Fig specification;
2. local specification generated or edited by the user;
3. `CrawlOne` fallback for commands not covered by Fig.

