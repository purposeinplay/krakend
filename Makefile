KRAKENDCE_REPO=https://github.com/devopsfaith/krakend-ce
IMPORT_FILE=backend_selector.go

all: build clean

build:
		git clone ${KRAKENDCE_REPO}
		cp ${IMPORT_FILE} krakend-ce/
		cd krakend-ce && make docker_build_alpine && VERSION=custom make -e krakend_docker

clean:
		rm -rf krakend-ce