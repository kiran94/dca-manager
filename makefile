GO_OUT=main

build:
	go build -o main main.go

debug:
	go build -gcflags=all="-N -l" -o $(GO_OUT) ./main.go
	cgdb $(GO_OUT)

terraform_apply:
	terraform -chdir=terraform apply
