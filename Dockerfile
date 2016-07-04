FROM scratch

ADD ./kafkalogsforwarder /kafkalogsforwarder
ADD https://papertrailapp.com/tools/papertrail-bundle.pem /papertrail-bundle.pem

ENTRYPOINT ["/kafkalogsforwarder"]

#BUILD: docker run --rm -v $PWD:/usr/src/myapp -w /usr/src/myapp golang:latest /bin/bash -c "go get -v ./...; go build -a -ldflags '-s' -tags netgo -installsuffix netgo -v -o $(basename $PWD); ldd $(basename $PWD)"; docker build -t erno/kafkalogsforwarder .
