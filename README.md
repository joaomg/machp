# machp

## go to machp home
cd c:\celfinet\machp

## download, add and remove unused dependencies
go tidy up

## create machp user
mysql -hlocalhost -P3306 -uroot -ppandora -e"source 0_machp_user.sql;"

## use machp user to drop/create the machp schema
mysql -hlocalhost -P3306 -umachp -pmachp123 machp_dev -e"source 1_machp_schema.sql;"

## run all tests
go test

## calculate test coverage
go test --coverprofile=cover.out

## launch server
go run .\server.go

## create tenant 1
curl -X POST -H "Content-Type: application/json" -d "{\"name\":\"tom\"}" localhost:1323/tenant

## get tenant 1
curl localhost:1323/tenant/1

## update tenant 1, change name from tom to jerry
curl -X PUT -H "Content-Type: application/json" -d "{\"id\":1, \"name\":\"jerry\"}" localhost:1323/tenant/1

## delete tenant 1
curl -X DELETE localhost:1323/tenant/1

