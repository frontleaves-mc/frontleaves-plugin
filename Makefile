# 变量定义，方便后续维护
MAIN_FILE = main.go
SWAG_CMD = swag
SWAG_FLAGS = --parseDependency
BUILD_SCRIPT = script/build-docker.sh
SCRIPT_DIR = script

.DEFAULT_GOAL := help

.PHONY: help swag run dev tidy proto proto-init

# 显示帮助信息
help:
	@echo "BambooBase - 可用命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make swag       - 生成 Swagger 文档"
	@echo "  make run        - 运行程序"
	@echo "  make dev        - 生成文档并运行 (推荐)"
	@echo "  make tidy       - 整理依赖"
	@echo "  make proto-init - 初始化 proto 符号链接"
	@echo "  make proto      - 生成 protobuf Go 代码"
	@echo ""
	@echo "示例:"
	@echo "  make dev"
	@echo ""

# 提取出的 Swagger 生成目标
swag:
	$(SWAG_CMD) init --instanceName frontleaves_plugin -g $(MAIN_FILE) --parseDependency

# 提取出的运行目标
run:
	go run $(MAIN_FILE)

tidy:
	go mod tidy

# Proto 符号链接初始化
BASE_GO_MODULE_DIR := $(shell go list -m -f '{{.Dir}}' github.com/bamboo-services/bamboo-base-go/plugins/grpc 2>/dev/null)
XBASE_LINK := proto/link/base.proto

proto-init:
	@mkdir -p $(dir $(XBASE_LINK))
	@if [ -z "$(BASE_GO_MODULE_DIR)" ]; then \
		echo "错误: 找不到 bamboo-base-go gRPC 模块，请先运行 go mod download"; \
		exit 1; \
	fi
	@ln -sf $(BASE_GO_MODULE_DIR)/proto/base.proto $(XBASE_LINK)
	@echo "符号链接已创建: $(XBASE_LINK) -> $(BASE_GO_MODULE_DIR)/proto/base.proto"

# 生成 protobuf Go 代码
proto:
	@cd proto && buf generate
	@rm -f proto/link/base.pb.go
	@echo "protobuf 代码生成完成"

# 组合目标：先生成文档，再运行程序
# 以后你只需要执行 `make dev` 就可以一键起飞了！
dev: swag run
