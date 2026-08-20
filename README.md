# Linko

This is a toy URL shortener project, to be used as the starter repo for the Logging and Telemetry course on [Boot.dev](https://www.boot.dev/).

It's intentionally small, a little messy, and realistic enough to practice adding logs, metrics, and traces in Go.

## Local workflow

The repository includes a `justfile` with the commands used most often in the
course. Run `just` to see the full list.

```bash
just run       # Linko with JSON logs written to linko.access.log
just dev       # monitoring stack in the background + Linko
just test      # Go tests
just lesson    # bootdev run for the current lesson
just submit    # bootdev run -s for the current lesson
```

At the metrics stage, these are also useful:

```bash
just metrics
just traffic 3500
just stack-status
just urls
```

Defaults can be overridden per invocation, for example:

```bash
LINKO_PORT=8080 LINKO_DATA_DIR=./tmp/data LINKO_LOG_FILE=./tmp/linko.log just run
just lesson <lesson-uuid>
```
