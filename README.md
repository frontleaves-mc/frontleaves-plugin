# bamboo-base-go-template

基于 `bamboo-base-go` 的业务端脚手架模板。

## 快速开始

1. 复制环境变量模板

```bash
cp .env.example .env
```

2. 安装依赖并运行

```bash
go mod tidy
make run
```

3. 健康检查接口

```bash
curl http://localhost:8080/api/v1/health/ping
```

## 目录结构

```text
.
├── api
├── internal
│   ├── app
│   │   ├── route
│   │   └── startup
│   ├── constant
│   ├── entity
│   ├── handler
│   ├── logic
│   └── repository
├── main.go
└── Makefile
```
