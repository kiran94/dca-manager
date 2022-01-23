GO_OUT=main
COVER_OUT=cover.out

build: build_execute_orders build_process_orders

build_execute_orders:
	go build -o $(GO_OUT) cmd/execute_orders/main.go && rm $(GO_OUT)

build_process_orders:
	go build -o $(GO_OUT) cmd/process_orders/main.go && rm $(GO_OUT)

test:
	gotestsum --format testname -- -race -coverprofile=$(COVER_OUT) ./...

coverage: test
	go tool cover -func=$(COVER_OUT)

lint:
	# go vet ./...
	golint -set_exit_status ./...
	staticcheck ./...

fmt:
	gofmt -s -w ./
	terraform -chdir=terraform fmt

debug:
	go build -gcflags=all="-N -l" -o $(GO_OUT) ./main.go
	cgdb $(GO_OUT)

terraform_apply:
	terraform -chdir=terraform apply

terraform_output:
	terraform -chdir=terraform output

pack_execute_orders:
	go get github.com/aws/aws-lambda-go/lambda
	GOOS=linux go build -o main cmd/execute_orders/main.go
	zip function.zip main

pack_process_orders:
	go get github.com/aws/aws-lambda-go/lambda
	GOOS=linux go build -o main cmd/process_orders/main.go
	zip function.zip main

update_all_packages:
	go get -u all

install_tools:
	go install gotest.tools/gotestsum@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/lint/golint@latest
