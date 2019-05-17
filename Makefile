all: install

install: auto_led auto_light

auto_led: auto_led/*.go
	go install ./auto_led

auto_light: auto_light/*.go
	go install ./auto_light

.PHONY: all
