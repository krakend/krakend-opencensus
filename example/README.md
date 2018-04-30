Example
====

## Dependencies

### Zipkin server

	$ docker run -d -p 9411:9411 openzipkin/zipkin

### Prometheus server

	$ brew install prometheus
	$ prometheus

## Build and Run

	$ go build
	$ ./example -l DEBUG -d -p 8080 -c krakend.json -name service_name -s 9091



Exposed traces: http://192.168.99.100:9411/zipkin/

Exposed metrics: http://127.0.0.1:9090/