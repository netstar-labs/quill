# Examples

One runnable program per way quill is used. Each takes a directory to work in and cleans
up after itself unless told otherwise.

| Example | Shows | Run |
|---|---|---|
| [log](log/main.go) | Open a log, append, flush, read, take a consumer, commit, close, reopen and resume | `GOWORK=off go run ./examples/log` |
| [topics](topics/main.go) | Three logs (`orders`, `payments`, `audit`) in one process, each with its own consumers: topics share nothing | `GOWORK=off go run ./examples/topics` |
| [pipeline](pipeline/main.go) | Two processes: a writer appends; a reader process starts late, opens a `Reader` and a consumer, catches up and keeps following | `GOWORK=off go run ./examples/pipeline -dir /tmp/pipe -role writer` and, in another terminal, `... -role reader`; `go test ./examples/pipeline` runs both |
