GO_OUT=main

build:
	go build -o main main.go

test:
	go test

lint:
	go vet

debug:
	go build -gcflags=all="-N -l" -o $(GO_OUT) ./main.go
	cgdb $(GO_OUT)

terraform_apply:
	terraform -chdir=terraform apply

pack_lambda:
	go get github.com/aws/aws-lambda-go/lambda
	GOOS=linux go build main.go
	zip function.zip main
