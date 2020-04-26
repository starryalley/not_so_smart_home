.PHONY: all

CMD := auto_led auto_light door_monitor sensor_logger

all: build

build:
	for target in $(CMD); do \
		$(BUILD_ENV_FLAGS) go build -v -o bin/$$target ./cmd/$$target; \
	done

test:
	go test ./...

install:
	cp bin/* $(GOBIN)/

clean:
	rm -rf ./bin/*
