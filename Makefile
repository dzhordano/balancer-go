run:
	go build -o .bin/balancer ./cmd/app/main.go
	.bin/balancer

run.cfg.test1:
	go build -o .bin/balancer ./cmd/app/main.go
	.bin/balancer -c=configs/test1.yaml

run.cfg.test2:
	go build -o .bin/balancer ./cmd/app/main.go
	.bin/balancer -c=configs/test2.yaml

test.benchmark:
	make run.cfg.test1 &
	make benchmark
	
test.benchmark2:
	make run.cfg.test2 &
	make benchmark.2

benchmark:
	sleep 1
	go-wrk -c 8 -d 30 http://localhost:8080/resource1

benchmark.2:
	sleep 1
	go-wrk -c 8 -d 30 http://localhost:8080/resource1

