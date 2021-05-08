module github.com/pdbogen/vator

go 1.15

require (
	github.com/cbroglie/mustache v1.0.1
	github.com/jrmycanady/nokiahealth v0.0.0-20180822201906-bc0ce2b8e4bc
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.8.1
	go.etcd.io/bbolt v1.3.5
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/sys v0.0.0-20210507161434-a76c4d0a0096 // indirect
)

replace github.com/jrmycanady/nokiahealth => github.com/pdbogen/nokiahealth v0.0.0-20190519180533-60d33df731d5
