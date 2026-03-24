# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Docker 多架构构建**: 支持 AMD64 和 ARM64 架构的 Linux 二进制构建
  - 新增 `Dockerfile.builder` 用于跨平台编译
  - 新增 `.build.env` 配置文件，支持自定义 GOPROXY 和 Alpine 镜像源
  - Makefile 新增 `build-linux-amd64`, `build-linux-arm64`, `build-linux` 目标
- **多环境配置支持**: 通过 `--env` 参数支持不同环境配置
  - 环境配置文件格式: `config.{env}.toml`（如 `config.prod.toml`, `config.test.toml`）
  - 每个环境独立完整配置，无合并逻辑
- **PostgreSQL 系数据库 schema 支持**: 新增可选 `schema` 配置字段
  - 用于 Vastbase/OpenGauss 等 PostgreSQL 系数据库指定默认模式（search_path）
  - `database` 字段统一作为数据库名称（dbname），`schema` 字段可选指定模式
  - 向后兼容：不指定 schema 时使用数据库默认模式
- **日志级别控制**: 新增 `--log-level` 参数，支持 debug/info/warn/error 级别
  - 默认日志级别: info
  - 通过结构化日志 (slog) 输出到 stderr
- **完整文档**:
  - 新增 `README.md` - 项目说明、快速开始、使用指南
  - 新增 `docs/design.md` - 设计文档（架构、技术选型、部署方案）
  - 新增 `docs/changelog.md` - 版本变更日志
  - 新增 `config.toml.example` - 配置模板文件

### Changed
- **CLI 简化**: 移除 `query` 子命令，查询功能直接集成到根命令
  - 删除 `cmd/query.go`，功能合并至 `cmd/root.go`
  - 简化命令调用: `pomelo-db -d mydb -e "SELECT 1"` (原 `pomelo-db query -d mydb -e "SELECT 1"`)
- **日志级别调整**:
  - 操作日志从 INFO 降级为 DEBUG（如连接建立、查询执行）
  - 错误日志保持 ERROR 级别
  - 默认 INFO 级别下，成功操作静默输出，仅错误时输出
- **错误输出优化**:
  - 设置 `SilenceUsage=true` 和 `SilenceErrors=true` 避免重复错误输出
  - 移除 `main.go` 中的冗余 `fmt.Fprintf` 错误输出
  - 移除数据库连接层的错误包装，避免 "query failed: query failed:" 重复
- **构建输出统一**: 所有二进制文件统一输出到 `bin/` 目录
  - Windows: `bin/pomelo-db.exe`
  - Linux AMD64: `bin/pomelo-db-linux-amd64`
  - Linux ARM64: `bin/pomelo-db-linux-arm64`
- **Go 版本统一**: go.mod 中 Go 版本从 1.25.3 降级到 1.23，与 Dockerfile 保持一致

### Removed
- 删除 `docs/plan.md` 旧规划文档（已被 `docs/design.md` 替代）

### Fixed
- 修复错误信息重复输出问题（slog.Error + Cobra 默认输出）
- 修复 Docker 构建中缺少 `file` 命令的非关键错误


## [0.2.0] - 2026-01-20

### Added
- **表格输出格式**: 通过 `--format table` 参数支持表格格式输出
- **达梦（DM）数据库支持**: 新增 DM 数据库连接和查询支持
- **Vastbase 数据库支持**: 新增 Vastbase 数据库连接和查询支持（兼容 PostgreSQL）
- **SQL 类型自动路由**: 根据 SQL 类型自动选择执行方法
  - SELECT/SHOW/DESC 等查询语句使用 `Query()` 返回结果集
  - INSERT/UPDATE/DELETE 等语句使用 `Exec()` 返回影响行数
- **表结构查询**: 支持 `DESCRIBE` 和 `DESC` 命令查询表结构

### Changed
- **简化 SQL 验证**: 移除复杂的 SQL 语法验证，仅保留基础检查
  - 不再检测危险操作（DROP/TRUNCATE 等）
  - 安全性由数据库权限控制，用户自行负责
- **时间格式标准化**: 查询结果中的时间字段统一格式化为 RFC3339 格式

### Fixed
- **列顺序稳定性**: 确保查询结果中列的顺序与 SQL 查询中的列顺序一致
  - 使用有序列名列表替代 map 迭代
  - 避免 Go map 的随机顺序导致输出不稳定


## [0.1.0] - 2026-01-15

### Added
- **项目初始化**: 创建 Go 版本 pomelo-db 项目
- **基础架构**:
  - CLI 框架（Cobra）
  - 配置管理（TOML）
  - 数据库连接层（database/sql）
  - 查询执行层
- **MySQL 支持**: MySQL 数据库连接和查询
- **SQL Server 支持**: SQL Server 数据库连接和查询
- **OpenGauss 支持**: OpenGauss 数据库连接和查询
- **基础功能**:
  - 通过 `-e` 执行 SQL 语句
  - 通过 `-f` 执行 SQL 文件
  - 查询超时控制（`--timeout`）
  - JSON 格式输出
- **配置管理**:
  - TOML 配置文件支持
  - 环境变量覆盖配置
  - 多数据源配置


## [0.0.1] - 2026-01-10

### Added
- 项目原型实现 (rdc_dbquery)
- Python 版本基础功能验证


---

## 版本说明

### 版本命名规范
- **Major (X.0.0)**: 重大架构变更或不兼容的 API 变更
- **Minor (0.X.0)**: 新增功能，向后兼容
- **Patch (0.0.X)**: Bug 修复，向后兼容

### 变更类型
- **Added**: 新增功能
- **Changed**: 现有功能的变更
- **Deprecated**: 即将废弃的功能
- **Removed**: 已移除的功能
- **Fixed**: Bug 修复
- **Security**: 安全相关的修复

### 链接
- [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)


[Unreleased]: https://github.com/mingyuan/pomelo-db/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/mingyuan/pomelo-db/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/mingyuan/pomelo-db/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/mingyuan/pomelo-db/releases/tag/v0.0.1
