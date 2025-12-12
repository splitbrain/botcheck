APACHE_DIR := apache
WHITELIST_DIR := whitelist

.PHONY: all ips useragents clean

all: $(APACHE_DIR)/ips $(APACHE_DIR)/useragents

ips: $(APACHE_DIR)/ips

useragents: $(APACHE_DIR)/useragents

$(APACHE_DIR)/ips $(APACHE_DIR)/useragents: $(WHITELIST_DIR)/*.go | $(APACHE_DIR)
	cd $(WHITELIST_DIR) && go build -o ../$@

$(APACHE_DIR):
	mkdir -p $@

clean:
	rm -f $(APACHE_DIR)/ips $(APACHE_DIR)/useragents
