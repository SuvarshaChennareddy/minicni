module github.com/suvarshachennareddy/minicni/daemon

go 1.27.3

replace github.com/suvarshachennareddy/minicni/common => ../common

require (
	github.com/stretchr/testify v1.12.1
	github.com/suvarshachennareddy/minicni/common v0.0.0-00010101000000-000000000000
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
