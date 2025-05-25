# *Inspora*

## 📖 项目简介

Inspora 是一个基于 Go 语言开发的现代化社交内容平台，提供文章发布、用户互动、内容搜索等功能。

## ✨ 功能特性

### 核心功能
- 📝 **文章管理** - 支持文章的创建、编辑、发布和删除
- 👥 **用户系统** - 完整的用户注册、登录、个人资料管理
- 💬 **评论系统** - 支持文章评论和回复功能
- 🔍 **内容搜索** - 基于 Elasticsearch 的全文搜索
- 📱 **关注系统** - 用户关注/取消关注功能
- 📰 **信息流** - 个性化内容推荐
- 📤 **文件上传** - 支持图片等文件上传
- 💰 **支付系统** - 集成微信支付功能

### 技术特性
- 🚀 **高性能** - 基于 Gin 框架的高性能 HTTP 服务
- 🔄 **异步处理** - 基于 Kafka 的消息队列系统
- ⏰ **定时任务** - 支持 Cron 定时任务调度
- 🔐 **安全认证** - JWT Token 认证机制
- 📊 **数据缓存** - Redis 缓存提升性能
- 🔍 **全文搜索** - Elasticsearch 搜索引擎

## 🛠 技术栈

### 后端技术
- **语言**: Go 1.23.3
- **框架**: Gin
- **数据库**: MySQL 8.0
- **缓存**: Redis
- **搜索引擎**: Elasticsearch 8.10.4
- **消息队列**: Kafka 3.6.0
- **ORM**: GORM
- **依赖注入**: Google Wire

### 第三方服务
- **短信服务**: 腾讯云 SMS
- **对象存储**: 阿里云 OSS
- **支付**: 微信支付 API v3

## 📁 项目结构

```
inspora/
├── main.go                 # 应用程序入口
├── app.go                  # 应用程序配置
├── viper.go               # 配置文件管理
├── wire.go                # 依赖注入配置
├── wire_gen.go            # Wire 生成的代码
├── go.mod                 # Go 模块依赖
├── docker-compose.yaml    # Docker 编排配置
├── .air.toml             # 热重载配置
├── config/               # 配置文件目录
├── internal/             # 内部代码
│   ├── web/             # HTTP 处理层
│   ├── service/         # 业务逻辑层
│   ├── repository/      # 数据访问层
│   ├── domain/          # 领域模型
│   ├── events/          # 事件处理
│   └── job/             # 定时任务
├── ioc/                 # 依赖注入容器
└── script/              # 脚本文件
```

## 🚀 快速开始

### 环境要求
- Go 1.23.3+
- Docker & Docker Compose
- Git

### 安装步骤

1. **克隆项目**
```bash
git clone https://github.com/Fairy-nn/inspora.git
cd inspora
```

2. **启动基础服务**
```bash
# 启动 MySQL、Redis、Kafka、Elasticsearch
docker-compose up -d
```

3. **安装依赖**
```bash
go mod download
```

4. **配置文件**
```bash
# 复制配置文件模板并修改相应配置
cp config/config.example.yaml config/config.yaml
```

5. **运行应用**
```bash
go run .
```

应用将在 `http://localhost:8080` 启动

## 🔧 配置说明

主要配置项包括：

- **数据库配置** - MySQL 连接信息
- **Redis 配置** - 缓存服务配置
- **Kafka 配置** - 消息队列配置
- **Elasticsearch 配置** - 搜索服务配置
- **第三方服务** - 腾讯云、阿里云、微信支付等配置

详细配置请参考 `config/` 目录下的配置文件。

## 🏗 架构设计

项目采用分层架构设计：

- **Web 层** - HTTP 请求处理和路由
- **Service 层** - 业务逻辑处理
- **Repository 层** - 数据访问抽象
- **Domain 层** - 领域模型定义

同时集成了：
- **事件驱动** - 基于 Kafka 的异步事件处理
- **定时任务** - 后台定时任务调度
- **缓存策略** - 多级缓存提升性能

---

⭐ 如果这个项目对你有帮助，请给它一个 Star！
