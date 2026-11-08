module github.com/suvarshachennareddy/minicni/plugins

go 1.27.3

require (
	github.com/suvarshachennareddy/minicni/common v0.0.0-00010101000000-000000000000
	github.com/vishvananda/netlink v1.3.1
)

require (
	github.com/vishvananda/netns v0.0.5 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace github.com/suvarshachennareddy/minicni/common => ../common
