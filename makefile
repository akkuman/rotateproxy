build:
	cd cmd/rotateproxy && go build -trimpath -ldflags="-s -w"