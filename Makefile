.PHONY: clean

firehose-http-endpoint-for-mackerel: go.* *.go
	go build -o $@ main.go
	file $@

bootstrap: go.* *.go
	GOOS=linux GOARCH=amd64 go build -o $@ main.go
	file $@

deploy: bootstrap
	lambroll deploy --log-level debug
	lambroll logs --follow

clean:
	rm -f bootstrap
	rm -f firehose-http-endpoint-for-mackerel
