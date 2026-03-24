# 设计文档

## 项目背景

### 问题
- Python 版本 rdc-dbtool 部署复杂，需要 Python 运行时和依赖包
- 客户集群环境资源受限，Python 运行时占用过多资源
- 版本管理复杂，依赖冲突频繁

### 解决方案
- 使用 Go 重写，编译为单一二进制文件
- 无需运行时依赖，直接运行
- 二进制体积小（10-20MB vs Python 50-200MB）
- 启动速度快（毫秒级 vs 秒级）

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────┐
│           CLI Layer (cmd/)              │
│  ┌──────────┐  ┌──────────┐            │
│  │  root    │  │  query   │            │
│  └──────────┘  └──────────┘            │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│        Business Layer (internal/)       │
│  ┌──────────┐  ┌──────────┐            │
│  │ config   │  │  query   │            │
│  └────┬─────┘  └────┬─────┘            │
│       │             │                   │
│  ┌────▼─────────────▼─────┐            │
│  │         db/             │            │
│  │  connection + scanner   │            │
│  └─────────────────────────┘            │
└─────────────────────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│          Database Drivers               │
│  MySQL | SQL Server | DM | OpenGauss   │
└─────────────────────────────────────────┘
```

### 分层职责

**CLI Layer (cmd/)**
- 命令行参数解析
- 用户交互
- 结果格式化输出

**Business Layer (internal/)**
- 配置管理：加载、验证、环境变量覆盖
- 查询执行：SQL 路由、超时控制
- 数据库抽象：连接管理、结果扫描

**Driver Layer**
- 各数据库驱动封装
- 统一的 database/sql 接口

## 核心模块设计

### 1. 配置管理 (internal/config/)

**设计原则：**
- 环境隔离：每个环境独立配置文件
- 简单直接：无复杂的合并逻辑
- 安全优先：敏感信息支持环境变量覆盖

**配置加载流程：**
```
用户输入 --env prod
    ↓
确定文件路径: config.prod.toml
    ↓
加载 TOML 文件
    ↓
验证配置结构
    ↓
应用环境变量覆盖
    ↓
返回配置对象
```

**环境变量命名规范：**
- 格式：`POMELO_DATASOURCE_{NAME}_{FIELD}`
- 示例：`POMELO_DATASOURCE_RDC_APP_PASSWORD`
- 优势：与 Kubernetes Secret 无缝集成

### 2. 数据库连接 (internal/db/)

**连接池管理：**
- 使用 Go 标准库 `database/sql`
- 连接池自动管理，无需手动释放
- 支持连接超时和查询超时

**连接字符串构建：**
- 根据数据库类型构建不同格式的连接字符串
- MySQL: `user:password@tcp(host:port)/database`
- SQL Server: `sqlserver://user:password@host:port?database=xxx`
- DM: `dm://user:password@host:port/database`
- Vastbase/OpenGauss: `host=x port=x user=x password=x dbname=x [search_path=schema]`
  - `database` 字段作为 `dbname`（数据库/实例名）
  - `schema` 字段（可选）作为 `search_path`（模式名）

**结果集扫描：**
- 动态列扫描：无需预知表结构
- 类型转换：自动处理 NULL、时间等特殊类型
- 内存优化：流式读取大结果集

### 3. 查询执行 (internal/query/)

**SQL 路由策略：**

```
检测 SQL 类型
    ↓
    ├─ SELECT/SHOW/DESC → Query() → 返回结果集
    │
    └─ INSERT/UPDATE/DELETE → Exec() → 返回影响行数
```

**设计决策：**
- 自动路由：用户无需关心 Query vs Exec
- 简单验证：只检测基本 SQL 类型和空语句
- 用户责任：安全性由用户和数据库权限保障

**超时控制：**
- Context 传递：使用 `context.WithTimeout`
- 可配置：通过 `--timeout` 参数调整
- 默认值：30 秒（平衡性能和安全）

### 4. CLI 设计 (cmd/)

**命令结构：**
```
pomelo-db
├── query              主功能：执行查询
│   ├── --datasource   数据源名称（必需）
│   ├── --execute      SQL 语句
│   ├── --file         SQL 文件
│   ├── --format       输出格式
│   └── --timeout      超时时间
│
└── [全局参数]
    ├── --config       配置文件路径
    └── --env          环境名称
```

**输出格式：**
- JSON：机器可读，易于集成
- Table：人类可读，适合终端查看
- 扩展性：易于添加新格式（CSV、YAML 等）

## 技术选型

### 核心依赖

| 组件 | 选型 | 理由 |
|------|------|------|
| **CLI 框架** | spf13/cobra | Go 社区标准，功能完善 |
| **配置解析** | BurntSushi/toml | 简单易读，无需额外依赖 |
| **MySQL 驱动** | go-sql-driver/mysql | 最成熟的 Go MySQL 驱动 |
| **SQL Server 驱动** | denisenkom/go-mssqldb | 官方推荐 |
| **日志** | log/slog | Go 1.21+ 标准库，性能优异 |

### 为什么不用 Viper？

**考虑因素：**
- 项目规模小，配置需求简单
- BurntSushi/toml 足够满足需求
- 减少依赖，降低二进制体积
- 保持代码简洁易维护

**如果需要 Viper：**
- 配置需求变复杂（多格式、远程配置等）
- 需要配置热加载
- 需要更强大的环境变量绑定

### 为什么不实现 MCP Server？

**原因：**
1. Go 生态无官方 MCP SDK
2. 手动实现成本高，维护负担重
3. 主要用于 K8s 集群，通过 ttyd 调用即可
4. POC 阶段聚焦核心功能

**如需 MCP：**
- 使用 Python 版本 rdc-dbtool（有 FastMCP 支持）
- 或等待 Go 官方 SDK 成熟后集成

## 部署方案

### Docker 构建策略

**多阶段构建：**
- Builder 阶段：编译 Go 二进制
- 无运行时阶段：无需最终镜像（直接提取二进制）

**跨架构支持：**
- AMD64：Intel/AMD 服务器
- ARM64：AWS Graviton、Apple Silicon
- 构建参数：`--build-arg TARGETARCH=amd64|arm64`

**镜像源优化：**
- Go Proxy：`goproxy.cn`（国内）
- Alpine Mirror：`mirrors.aliyun.com`
- 可配置：通过 Makefile 变量覆盖

### Kubernetes 部署模式

**ConfigMap + Secret 分离：**

```
ConfigMap (非敏感)
├── 数据库地址
├── 端口
├── 数据库名
└── 用户名

Secret (敏感)
└── 密码
```

**优势：**
- 权限分离：不同角色管理不同资源
- 安全性高：Secret 加密存储
- 易于更新：修改 Secret 无需重建 Pod

**ttyd 集成：**
- 通过 ttyd 暴露 Web 终端
- 用户通过浏览器执行查询
- 无需直接 SSH 到集群

## 设计决策

### 1. 配置合并 → 独立配置

**旧方案：**
- 基础配置 + 环境配置合并
- 需要手动实现合并逻辑

**新方案：**
- 每个环境独立完整配置
- 无需合并，代码更简单

**理由：**
- 符合 Go 社区最佳实践
- 配置更清晰���易于维护
- 减少代码复杂度

### 2. SQL 验证策略

**选择：基础验证**
- 空 SQL 检测
- SQL 类型检测（SELECT vs DML）
- 不验证语法、不检测危险操作

**理由：**
- 简化实现，降低维护成本
- 数据库权限是第一道防线
- 用户责任：谁执行谁负责
- 避免误杀合法操作

### 3. 输出格式

**当前：JSON + Table**
- JSON：默认，机器可读
- Table：可选，人类友好

**未来扩展：**
- CSV：数据导出
- YAML：配置友好
- 易于实现：只需添加格式化函数

### 4. 错误处理

**策略：**
- 尽早失败：配置错误立即返回
- 友好提示：错误信息清晰易懂
- 不隐藏错误：透传数据库错误

**示例：**
```
Error: datasource 'xxx' not found
Available datasources: rdc_app, mysql_example, ...
```

## 性能考虑

### 启动速度
- Go 编译二进制，启动毫秒级
- 无 Python 解释器开销
- 配置文件解析快速

### 内存占用
- 基础占用：5-10MB
- 查询期间：动态增长
- 无常驻进程：执行完即退出

### 二进制体积
- 当前：约 15-20MB
- 优化手段：
  - `-ldflags="-s -w"`：去除符号表
  - `-trimpath`：移除路径信息
  - 可进一步使用 UPX 压缩

## 安全考虑

### 配置文件
- `.gitignore` 排除所有 `config.*.toml`
- 仅提交 `config.toml.example` 模板
- 密码留空，强制环境变量

### 数据库权限
- 推荐只读账户
- 最小权限原则
- 生产环境使用专用账户

### Kubernetes Secret
- 密码通过 Secret 注入
- RBAC 控制 Secret 访问
- 定期轮换密码

## 未来规划

### 短期（1-2 个月）
- [ ] 添加查询结果缓存
- [ ] 支持批量 SQL 执行
- [ ] 添加更多输出格式（CSV、YAML）
- [ ] 完善单元测试覆盖率

### 中期（3-6 个月）
- [ ] 支持事务执行
- [ ] 添加查询历史记录
- [ ] Web UI（可选）
- [ ] 查询结果导出功能

### 长期（6 个月以上）
- [ ] MCP Server 支持（待 Go SDK 成熟）
- [ ] 查询优化建议
- [ ] 慢查询分析
- [ ] 多数据源联合查询

## 参考资料

- [12-Factor App](https://12factor.net/)
- [Kubernetes ConfigMap Best Practices](https://kubernetes.io/docs/concepts/configuration/configmap/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Cobra CLI Guide](https://github.com/spf13/cobra)
