CREDS_PATH:=/team/umcloud/secrets/um-cloud/openstack/admin-openrc.sh
PREFIX:=$(shell [ $$(uname -s) = Darwin ] && echo /Volumes/Keybase || echo /keybase)
LOAD_CREDS:=. $(PREFIX)/team/umcloud/secrets/um-cloud/openstack/admin-openrc.sh
ARGS=-e NOBORRAR
BINNAME:=os-cleanup
GOOS:=$(shell go env GOOS)
GOARCH:=$(shell go env GOARCH)

SRC:= $(shell find . -type f -name '*.go' -not -path "./vendor/*")

TARGET:=build/os_cleanup
RUN=./$(TARGET)	server $(ARGS) $(X)

all: build

test:
	go test -v --count=1 -race ./...

build: $(TARGET)

output: out/list.md out/list.json out/list.table out/emails.txt
	@ls -al out/*

out/emails.txt: out/list.json
	jq -r '.[].email' < $(^) | sort -u | xargs | sed 's/ /,/g' > $(@)

out/list.%: $(TARGET)
	@mkdir -p out
	$(LOAD_CREDS) && $(RUN) -a list -o $(*) | tee $(@)

list-tagged: ARGS=--tagged -d0
list-tagged: run-list

#step-01-tag: X=--yes
step-01-tag: run-tag output

#step-02-stop: X=--yes
step-02-stop: run-stop output

#step-03-stop: X=--yes
step-03-delete: run-delete output

# E.g.:
#   make run-list X="-o json"
#   make run-list X="-o md"
run-%: $(TARGET)
	$(LOAD_CREDS) && $(RUN) -a $(*) $(X)

$(TARGET): $(SRC)
	go build -o build ./...

gofumpt:
	gofumpt -w $(SRC)

lint:
	golangci-lint run ./...

clean:
	rm -f $(TARGET) out/*

.PHONY: all build output run-% list-tagged step-01-tag step-02-stop step-03-delete
