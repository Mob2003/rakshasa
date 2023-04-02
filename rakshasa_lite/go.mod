module rakshasa_lite

go 1.16

replace cert => ../cert

require (
	cert v0.0.0-00010101000000-000000000000 // indirect
	github.com/creack/pty v1.1.18
	github.com/farmerx/gorsa v0.0.0-20161211100049-3ae06f674f40
	github.com/google/uuid v1.3.0
	gopkg.in/yaml.v3 v3.0.1
)
