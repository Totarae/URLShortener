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