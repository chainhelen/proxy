### http_min.go 
仅使用net库，而没有使用net/http库编写成的,
支持http代理，支持https。

http_min.go 可以运行在unix-like 的操作系统下面
```
go run http_min.go
```
或者使用
```
go build http_min.go
./http_min
```
生成可执行文件http_min.go 
默认监听端口127.0.0.1:6010

-----------
MIT 协议