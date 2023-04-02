module rakshasa

go 1.16

replace github.com/abiosoft/readline => ./readline

replace cert => ./cert

require (
	cert v0.0.0-00010101000000-000000000000
	github.com/abiosoft/readline v0.0.0-20180607040430-155bce2042db
	github.com/creack/pty v1.1.18
	github.com/farmerx/gorsa v0.0.0-20161211100049-3ae06f674f40 // indirect
	github.com/google/uuid v1.3.0
	github.com/luyu6056/ishell v1.0.1
	github.com/mattn/go-colorable v0.1.12 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/text v0.3.7
	gopkg.in/yaml.v3 v3.0.1
)
