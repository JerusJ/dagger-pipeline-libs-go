module ci

go 1.20

replace (
	github.com/jerusj/dagger-pipeline-libs-go/v2 => ../
)

require (
	dagger.io/dagger v0.8.6
	golang.org/x/sync v0.3.0
)

require (
	github.com/99designs/gqlgen v0.17.36 // indirect
	github.com/Khan/genqlient v0.6.0 // indirect
	github.com/adrg/xdg v0.4.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/jerusj/dagger-pipeline-libs-go/v2 v2.0.0-20230918211105-4dcd2170c890 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.8 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)