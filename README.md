# machp

## go to machp home
cd c:\celfinet\machp

## download, add and remove unused dependencies
go tidy up

## run all tests
go test

## launch server
go run .\server.go

## create tenant 1
curl -X POST -H "Content-Type: application/json" -d "{\"name\":\"tom\"}" localhost:1323/tenant

## get tenant 1
curl localhost:1323/tenant/1