# firehose-http-endpoint-for-mackerel

A PoC of Firehose HTTP Endpoint to post metrics to [Mackerel](https://mackerel.io/).

## Requirements

Go 1.14

## How to run on local

```
$ make
$ ./firehose-http-endpoint-for-mackerel
```

firehose-http-endpoint-for-mackerel runs on 0.0.0.0:8080.

## How to run on Amazon API Gateway

[Makefile](Makefile) includes a example target for deployment to a Lambda function `firehose-http-endpoint-for-mackerel` on your AWS account by [Lambroll](https://github.com/fujiwara/lambroll).

```
$ GOARCH=amd64 GOOS=linux make
$ ROLE_ARN=arn:aws:iam::123456789012:role/lambda make deploy
```

Create an Amazon API Gateway which have a "HTTP integration" with that Lambda function.

The IAM role for the lambda function requires only a policy equals to `arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecution`.

## Connect with Kinesis Data Firehose

Setup a Kinesis Data Firehose which have a destination to HTTP endpoint for the API Gateway or your custom endpoint.

- HTTP endpoint URL: https://..../service
- Access key: Your Mackerel API Key
- Content encoding: "Disabled"
- Parameters
  - service: A Mackerel service name to post metrics

## How to post metrics to Firehose

Post service metrics to the Firehose delivery stream. Record formats are allowed as below.

- JSON format `{"name":"metric.name","time":1596382129,"value":27759}`
- Text format same as for Mackerel agent `metric.name\t27759\t1596382129`

# LICENSE

MIT

# Author

@fujiwara
