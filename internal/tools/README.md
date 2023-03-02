# Internal tools

The purpose of this package is:

1. To allow efficient caching of build/testing tools in CI jobs.
2. To pin these tools to specific versions to improve consistency & reliability.

You don't need to do anything to 'enable' this, it's handled in the build file (`magefile.go`).