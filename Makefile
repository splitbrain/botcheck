APACHE_DIR := apache
LOOKUP_DIR := lookup

.PHONY: all lookup ips useragents clean

all: lookup

lookup: $(APACHE_DIR)/lookup

$(APACHE_DIR)/lookup: $(LOOKUP_DIR)/*.go | $(APACHE_DIR)
	cd $(LOOKUP_DIR) && go build -o ../$@

$(APACHE_DIR):
	mkdir -p $@

clean:
	rm -f $(APACHE_DIR)/lookup
