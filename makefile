GO_OUT=main

build: build_execute_orders build_process_orders

build_execute_orders:
	go build -o $(GO_OUT) lambda/execute_orders/main.go

build_process_orders:
	go build -o $(GO_OUT) lambda/process_orders/main.go

test:
	go test ./pkg/configuration/ ./pkg/orders ./lambda/execute_orders ./lambda/process_orders

lint:
	go vet

debug:
	go build -gcflags=all="-N -l" -o $(GO_OUT) ./main.go
	cgdb $(GO_OUT)

terraform_apply:
	terraform -chdir=terraform apply

terraform_output:
	terraform -chdir=terraform output

pack_execute_orders:
	go get github.com/aws/aws-lambda-go/lambda
	GOOS=linux go build -o main lambda/execute_orders/main.go
	zip function.zip main

pack_process_orders:
	go get github.com/aws/aws-lambda-go/lambda
	GOOS=linux go build -o main lambda/process_orders/main.go
	zip function.zip main

update_all_packages:
	go get -u all
