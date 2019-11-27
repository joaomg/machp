# machp

## go to machp home
cd c:\celfinet\machp

## download, add and remove unused dependencies
go tidy up

## run all tests
go test

## launch server
go run .\server.go

## create tenant creation 
curl -X POST -H "Content-Type: application/json" -d "{\"name\":\"tom\"}" localhost:1323/tenant