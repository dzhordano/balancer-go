## Usage

- specify config in configs/[**config.yaml** for default] (everything your need to specify is in **internal/config/config.go** or see **configs/config.yaml** for example)
- make run (if **make** is installed)
- go run ./cmd/app/main.go (-c=configs/test1.yaml for specific config)

### Dependencies

- [golang](https://golang.org/)
- [go-wrk](https://github.com/dzhordano/go-wrk)****
- [make](https://www.gnu.org/software/make/) [optional]

#### ToFix

concurrent request handling + caching responses
metrics poorly made
separate servers appending and removing from balancers
