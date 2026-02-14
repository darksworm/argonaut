TAPES := overview sync resources commands themes

.PHONY: build record record-all clean $(addprefix record-,$(TAPES)) \
	k9s-cluster-up k9s-cluster-down record-k9s

BIN_DIR := bin
VHS_PATH := PATH="$(CURDIR)/$(BIN_DIR):$(PATH)"
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")

# k3d cluster for k9s demo (not included in TAPES — requires a real cluster)
K3D_K9S_CLUSTER := argonaut-k9s-demo
K9S_KUBECONFIG := $(BIN_DIR)/k9s-demo-kubeconfig

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "-X main.appVersion=$(VERSION)" -o $(BIN_DIR)/argonaut-app ../cmd/app
	go build -ldflags "-X main.appVersion=$(VERSION)" -o $(BIN_DIR)/argonaut ../cmd/demo

# Record all demos
record-all: build $(addprefix record-,$(TAPES))

# Record individual demos: make record-overview, make record-sync, etc.
$(addprefix record-,$(TAPES)): record-%: build
	$(VHS_PATH) vhs $*.tape

# Default: record all
record: record-all

# --- k9s demo (requires k3d + k9s installed) ---

k9s-cluster-up:
	@bash setup-k9s-cluster.sh $(K3D_K9S_CLUSTER) $(CURDIR)/$(K9S_KUBECONFIG)

k9s-cluster-down:
	@k3d cluster delete $(K3D_K9S_CLUSTER) 2>/dev/null || true
	@rm -f $(K9S_KUBECONFIG)

record-k9s: build k9s-cluster-up
	KUBECONFIG="$(CURDIR)/$(K9S_KUBECONFIG)" ARGONAUT_DEMO_SCENARIO=k9s $(VHS_PATH) vhs k9s.tape

clean:
	rm -f $(BIN_DIR)/* *.gif *.mp4 *.webm
