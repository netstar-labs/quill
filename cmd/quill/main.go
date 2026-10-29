// Command quill inspects and manages a quill log directory: no server, one directory at a
// time. Run `quill help` for the verbs; docs/reference.md is generated from the same table.
package main

import "os"

// version is stamped by build/quill at release time.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
