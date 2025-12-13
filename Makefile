APACHE_DIR := apache
WHITELIST_DIR := whitelist

.PHONY: all whitelist ips useragents clean

all: whitelist

whitelist: $(APACHE_DIR)/whitelist

ips useragents: whitelist
	@echo "Use $(APACHE_DIR)/whitelist; config selection now happens at runtime."

$(APACHE_DIR)/whitelist: $(WHITELIST_DIR)/*.go | $(APACHE_DIR)
	cd $(WHITELIST_DIR) && go build -o ../$@

$(APACHE_DIR):
	mkdir -p $@

clean:
	rm -f $(APACHE_DIR)/whitelist
