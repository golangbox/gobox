package main

import (
	"strings"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

type mockRemoteEventCollector struct {
	changes chan stateChange
}

func (c *mockRemoteEventCollector) NewRemoteEvent(sc stateChange) {
	c.changes <- sc
}

func DescribeReaderWatcher(c gospec.Context) {
	c.Specify("a io.Reader watcher", func() {
		c.Specify("generates state changes", func() {
			watcher := readerWatcher{}
			collector := mockRemoteEventCollector{make(chan stateChange)}

			watcher.watchFrom(strings.NewReader("update\n"), &collector)
			change := <-collector.changes

			c.Expect(change.(string), Equals, "update")
			c.Expect(watcher.err, IsNil)
		})
	})
}
