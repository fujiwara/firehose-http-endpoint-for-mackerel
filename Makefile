.PHONY: clean

firehose-http-endpoint-for-mackerel: go.* *.go
	go build -o $@ main.go
	file $@

deploy: firehose-http-endpoint-for-mackerel
	lambroll deploy --log-level debug
	lambroll logs --follow

clean:
	rm -f firehose-http-endpoint-for-mackerel
