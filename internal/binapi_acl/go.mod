module github.com/ifzzh/cmd-nse-template/internal/binapi_acl

go 1.23.8

require (
	github.com/networkservicemesh/govpp v0.0.0-20240328101142-8a444680fbba
	go.fd.io/govpp v0.11.0
)

require github.com/lunixbochs/struc v0.0.0-20200521075829-a4cb8d33dbbe // indirect

// 指向本地已本地化的 acl_types 模块
replace github.com/networkservicemesh/govpp/binapi/acl_types => ../binapi_acl_types
