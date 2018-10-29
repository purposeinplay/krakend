# provablyfair krakend

## Build

	go get github.com/ProvablyFair/krakend
	# ignore the errors
	cd $GOPATH/src/github.com/ProvablyFair/krakend/
	make

The docker image `devopsfaith/krakend:custom` contains your custom KrakenD API Gateway

## Run

Start the server locally from `$GOPATH/src/github.com/ProvablyFair/krakend/` with:

    docker run -p 8080:8080 -v "$PWD:/etc/krakend/" devopsfaith/krakend:custom run -c /etc/krakend/krakend.json


