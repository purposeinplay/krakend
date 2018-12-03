KRAKENDCE_REPO=https://github.com/devopsfaith/krakend-ce
KRAKENDCE_VERSION=0.7.0

all: build clean

build:
		git clone -b "${KRAKENDCE_VERSION}" ${KRAKENDCE_REPO}
		cp backend_selector.go krakend-ce/.
		cp handler_factory.go krakend-ce/.
		cat gopkg.toml.partial >> krakend-ce/Gopkg.toml
		cd krakend-ce && make docker_build_alpine && VERSION=${KRAKENDCE_VERSION}-custom make -e krakend_docker

clean:
		rm -rf krakend-ce