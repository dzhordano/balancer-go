run:
	go build -o .bin/balancer ./cmd/app/main.go
	.bin/balancer

test1.1:
	go run ./cmd/app/main.go -c=configs/test.1.yaml &
	go-wrk -c 10 -d 15 -M GET http://localhost:8080/resource1

test1.2:
	go run ./cmd/app/main.go -c=configs/test.1.yaml &
	go-wrk -c 10 -d 30 -M GET http://localhost:8080/resource1

test2:
	go run ./cmd/app/main.go -c=configs/test.2.yaml &
	go-wrk -c 10 -d 30 -M GET http://localhost:8080/resource1