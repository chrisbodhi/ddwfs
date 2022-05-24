# ddwfs

## your data.world datasets as a virtual file system

### Getting started (macOS)

1. Install [MacFUSE](https://osxfuse.github.io/)
1. Get your [data.world API token](https://data.world/settings/advanced) (probably the `READ/WRITE` option)
1. Save your API key as an environment variable, `DW_AUTH_TOKEN`, however you like
    For instance, `export DW_AUTH_TOKEN="e..."` in whatever terminal window you're working in
1. Save your userid as an env var, `DW_USERNAME`, however you like
1. Get going: `go build`
