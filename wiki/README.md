# Wiki navigation source

This directory stores the source for the repository's thin GitHub Wiki navigation layer.

The Wiki is intentionally **not** a second documentation tree. Canonical product, installation, security and operations documentation remains under [`docs/`](../docs/). Wiki pages should contain orientation, short descriptions and links to canonical documents, not copied procedures.

## Pages

- `Home.md` — English navigation home.
- `Inicio.md` — Brazilian Portuguese navigation home.
- `_Sidebar.md` — compact cross-language navigation.

## Publish

The repository includes a safe synchronization helper that publishes only the three managed navigation files and preserves any other Wiki pages:

~~~bash
bash scripts/publish-wiki.sh
~~~

Check whether the native Wiki differs without pushing anything:

~~~bash
bash scripts/publish-wiki.sh --check
~~~

The script derives the `.wiki.git` remote from the existing `origin` remote and reuses the GitHub authentication already configured in the environment. It does not embed credentials. If the Wiki remote has not been initialized yet or the current Git identity/authentication cannot publish, the script stops without changing Wiki content.

## Maintenance rule

When documentation paths or product entry points change, update these navigation links in the same pull request. Do not copy full procedures from `docs/` into the Wiki.
