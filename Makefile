APACHE_DIR := apache
LOOKUP_DIR := lookup

.PHONY: all lookup test ips useragents clean

all: lookup

lookup: $(APACHE_DIR)/lookup

$(APACHE_DIR)/lookup: $(LOOKUP_DIR)/*.go | $(APACHE_DIR)
	cd $(LOOKUP_DIR) && CGO_ENABLED=0 go build -o ../$@

test:
	cd $(LOOKUP_DIR) && go test ./...

$(APACHE_DIR):
	mkdir -p $@

clean:
	rm -f $(APACHE_DIR)/lookup
