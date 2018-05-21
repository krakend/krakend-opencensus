Example
====

## Dependencies

### Zipkin server

	$ docker run -d -p 9411:9411 openzipkin/zipkin

### Prometheus server

	$ brew install prometheus
	$ prometheus

### Jaeger server

	$ docker run -d -e \
	  COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
	  -p 5775:5775/udp \
	  -p 6831:6831/udp \
	  -p 6832:6832/udp \
	  -p 5778:5778 \
	  -p 16686:16686 \
	  -p 14268:14268 \
	  -p 9412:9411 \
	  jaegertracing/all-in-one:latest

## Influxd server

	$ docker run -p 8086:8086 \
	  -e INFLUXDB_DB=krakend \
	  -d --name=influx \
	  influxdb

## Build and Run

	$ go build
	$ gcvis ./example -l DEBUG -d -p 8080 -c krakend.json -name gateway0 -s 9091
	$ gcvis ./example -l DEBUG -d -p 8081 -c krakend_2.json -name gateway1 -s 9092
	$ gcvis ./example -l DEBUG -d -p 8082 -c krakend_3.json -name gateway2 -s 9093



Exposed traces: 

+ zipkin http://192.168.99.100:9411/zipkin/
+ jaeger http://192.168.99.100:16686/search


Exposed metrics: http://127.0.0.1:9090/