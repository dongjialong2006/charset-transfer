# The import path is where your repository can be found.
# To import subpackages, always prepend the full import path.
# If you change this, run `make clean`. Read more: https://git.io/vM7zV
IMPORT_PATH := charset-transfer

.PHONY: all
all: charset-transfer

.PHONY: charset-transfer
charset-transfer: .GOPATH/.ok
	go install -tags netgo $(IMPORT_PATH)/cmd
	mv bin/cmd bin/charset-transfer
	

update: .GOPATH/.ok
	glide mirror set https://golang.org/x/crypto https://github.com/golang/crypto
	glide mirror set https://golang.org/x/sys https://github.com/golang/sys
	glide up -v


clean:
	rm -rf bin/* .GOPATH

export GOPATH := $(CURDIR)/.GOPATH

unexport GOBIN


.GOPATH/.ok:
	rm -rf $(CURDIR)/.GOPATH
	mkdir -p $(CURDIR)/.GOPATH/src
	ln -sf $(CURDIR) $(CURDIR)/.GOPATH/src/$(IMPORT_PATH)
	mkdir -p $(CURDIR)/bin
	ln -sf $(CURDIR)/bin $(CURDIR)/.GOPATH/bin
	touch $@