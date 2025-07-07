# cmd/shortener

В данной директории будет содержаться код, который скомпилируется в бинарное приложение

````
go build -buildvcs=false -o shortener.exe
````
Для сборки

````
C:\Users\admin\go\bin\mockgen.exe -source=C:\Users\admin\GolandProjects\URLShortener\internal\database\db.go -destination=C:\Users\admin\GolandProjects\URLShortener\internal\mocks\m
ock_database.go -package=mocks
````
Для моков, потому что mockgen почему-то встал в home

````
openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365 -subj "/CN=localhost"
````
Для генерации сертификатов


````
protoc -I=api/proto --go_out=internal/pkg/proto_gen --go-grpc_out=internal/pkg/proto_gen api/proto/shortener_v2.proto
````
Для генерации proto

````
grpcurl -plaintext -d "{\"user_id\":\"123\",\"url\":\"https://example.com\"}" localhost:3200 shortener.v2.ShortenerService.Shorten
{
  "shortUrl": "eaaarvrs5qv39c9s3zo0zw"
}
````
````
H:\Soft\GRPCCurl>grpcurl -plaintext -d "{\"short_url\":\"abc123\"}" localhost:3200 shortener.v2.ShortenerService.Resolve
ERROR:
  Code: Internal
  Message: db error: URL not found: no rows in result set

H:\Soft\GRPCCurl>grpcurl -plaintext -d "{\"short_url\":\"eaaarvrs5qv39c9s3zo0zw\"}" localhost:3200 shortener.v2.ShortenerService.Resolve
{
  "originalUrl": "https://example.com"
}

````
````
H:\Soft\GRPCCurl>grpcurl -plaintext -d "{\"user_id\":\"123\",\"urls\":[{\"correlation_id\":\"1\",\"original_url\":\"https://yandex.com\"},{\"correlation_id\":\"2\",\"original_url\":\"https://google.com\"}]}" localhost:3200 shortener.v2.ShortenerService.BatchShorten
{
"items": [
{
"correlationId": "1",
"shortUrl": "relult5141ya4dj6rv31va"
},
{
"correlationId": "2",
"shortUrl": "bqrvjsg-jiiz3asuq2pq0q"
}
]
}
````
Курл для GRPC