# Changelog

## v0.5.0 (Unreleased)

### DSN 格式配置支持

- 新增 DSN (Data Source Name) 格式配置，大幅简化数据源配置
- 格式：`<db-type>://<user>:<password>@<host>:<port>/<database>[?schema=<schema>]`
- SQLite 特殊格式：`sqlite:///path/to/file.db`
- 向后兼容：传统结构化配置格式仍然支持
- 配置示例：
  ```yaml
  query:
    datasources:
      mydb: "mysql://root:secret@127.0.0.1:3306/mydb"
      vastbase: "vastbase://user:pass@host:5432/db?schema=public"
      sqlite: "sqlite:///path/to/file.db"
  ```

### Bug 修复

- 修复 SQLite 相对路径解析问题，支持 `sqlite:///./path/to/db` 格式
- 自动将相对路径转换为绝对路径，避免 "unable to open database file" 错误

### 改进

- Makefile: 将 `install-skill` 重命名为 `skill`，与 pomelo-pw 保持一致
- 修复 help 信息中的路径显示，使用 `$(SKILL_DST)` 变量

## v0.4.0

### 统一 CLI 配置覆盖 & 可写模式支持

- 新增 SQLite 数据库支持（纯 Go 驱动 `modernc.org/sqlite`，无需 CGO）
- 新增 `--config`/`-c` 参数，支持 `key=value` 格式覆盖任意配置项，可重复使用
- 移除 `--verbose` 参数，改用 `-c log.level=debug` 替代
- 新增 `query.readonly` 配置项，默认 `true`（只读），设为 `false` 时允许写操作
- JSON 输出字段 `rowcount` 重命名为 `row_affected`
- 表格输出和语句结果增加执行耗时（毫秒）
- SQL 输入自动清理 shell 续行符（`\` + 换行）
- Makefile: `install-skill` 添加 `build` 依赖，支持跨平台二进制后缀
- Skill 文档同步更新

**配置优先级（低→高）：** 内嵌默认值 → 用户 config.yaml → `-c key=value`

**用法示例：**
```bash
pomelo-db -d mydb -e "SELECT 1" -c log.level=debug
pomelo-db -d mydb -e "INSERT INTO t VALUES(1)" -c query.readonly=false
```

## v0.3.0

### Viper 配置管理 & 日志级别可配置化

- 引入 Viper 替代手动 YAML 解析，支持默认配置 + 用户配置合并
- 内嵌 `config.defaults.yaml` 到二进制，无需用户提供即可启动
- 新增 `log.level` 配置项（支持 debug/info/warn/error），取代仅有的 `--verbose` 布尔开关
- `--verbose` flag 保留，作为强制覆盖 log level 为 debug 的快捷方式
- 删除 `config.yaml.example`，默认配置即文档

**配置优先级（低→高）：** 内嵌默认值 → 用户 config.yaml → CLI `--verbose`

## v0.2.0

### 项目重命名

- 项目名称从 `mks-ttyd-query`/`mks-query` 重命名为 `pomelo-db`
- 环境变量前缀从 `MKS_QUERY_` 改为 `POMELO_DB_`
- 配置目录从 `~/.mks-query/` 改为 `~/.pomelo-db/`
- 配置结构 `datasource.<name>` → `query.datasources.<name>`（YAML 格式）
- VastBase/OpenGauss 字段 `instance` → `database`，新增可选 `schema`
- 新增 `--config-dir` 参数，支持与 mks-ttyd 共享配置目录
- 默认只读模式，仅允许 SELECT/SHOW/DESCRIBE
- JSON 输出格式对齐：`{rowcount, data, message}`
- 新增 `--format table` 表格输出、`--file` 从文件读取 SQL
