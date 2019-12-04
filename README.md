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

## create tenant 2
curl -X POST -H "Content-Type: application/json" -d "{\"name\":\"tom\"}" localhost:1323/tenant

## get tenant 2
curl localhost:1323/tenant/2

## update tenant 2, change name from tom to jerry
curl -X PUT -H "Content-Type: application/json" -d "{\"id\":2, \"name\":\"jerry\"}" localhost:1323/tenant/2

## upload files to tenant jerry
curl -X POST -F files=@c:\tmp\1.txt -F files=@c:\tmp\2.txt localhost:1323/tenant/jerry/upload

## delete tenant 1
curl -X DELETE localhost:1323/tenant/1

## build docker image
docker build -f Dockerfile . -t machp

## run docker image
docker run -d -p 8080:1323 --name machp-dev machp

## check the echo server is listening on port 8080
curl -X GET localhost:8080/tenant/1
