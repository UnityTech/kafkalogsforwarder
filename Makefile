DATESTR := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")


all: build upload

default: all


build:
	/usr/local/go/bin/go build -race -ldflags "-X main.builddate=$(DATESTR)" -o kafkalogsforwarder github.com/garo/kafkalogsforwarder/


upload-version: build
	mkdir -p builds/
	cp kafkalogsforwarder builds/kafkalogsforwarder-$(DATESTR)
	shasum builds/kafkalogsforwarder-$(DATESTR) > builds/kafkalogsforwarder-$(DATESTR).shasum
	sed -i.bak 's/builds\///g' builds/kafkalogsforwarder-$(DATESTR).shasum
	s3cmd put builds/kafkalogsforwarder-$(DATESTR) s3://applifier_deployment/bin/kafkalogsforwarder-$(DATESTR) -P
	s3cmd put builds/kafkalogsforwarder.shasum s3://applifier_deployment/bin/kafkalogsforwarder-$(DATESTR).shasum -P

upload-production: build
	mkdir -p builds/
	cp kafkalogsforwarder builds/kafkalogsforwarder
	shasum builds/kafkalogsforwarder > builds/kafkalogsforwarder.shasum
	sed -i.bak 's/builds\///g' builds/kafkalogsforwarder.shasum
	s3cmd put builds/kafkalogsforwarder s3://applifier_deployment/bin/kafkalogsforwarder -P
	s3cmd put builds/kafkalogsforwarder.shasum s3://applifier_deployment/bin/kafkalogsforwarder.shasum -P


upload-testing: build
	mkdir -p builds/
	cp kafkalogsforwarder builds/kafkalogsforwarder-testing
	shasum builds/kafkalogsforwarder-testing > builds/kafkalogsforwarder-testing.shasum
	sed -i.bak 's/builds\///g' builds/kafkalogsforwarder-testing.shasum
	s3cmd put builds/kafkalogsforwarder-testing s3://applifier_deployment/bin/kafkalogsforwarder-testing -P
	s3cmd put builds/kafkalogsforwarder.shasum s3://applifier_deployment/bin/kafkalogsforwarder-testing.shasum -P

test-kafka:
	kafkalogsforwarder_KAFKA_CONNECTION_STRING=kafka-logs-b-1.us-east-1.applifier.info:9092 /usr/local/go/bin/go test -race -timeout 10s -v github.com/garo/kafkalogsforwarder

test:
	/usr/local/go/bin/go test -race -timeout 60s -v github.com/garo/kafkalogsforwarder
