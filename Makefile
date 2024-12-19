run:
	go build -o .bin/balancer ./cmd/app/main.go
	.bin/balancer
