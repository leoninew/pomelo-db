# pomelo-db skill

Claude Code 数据库查询技能，封装 pomelo-db CLI 工具。

## 快速开始

不带参数，在对话中描述需求：

```bash
/pomelo-db
```

## 常用命令

```bash
# 列出所有数据源
pomelo-db -l

# 查看数据源配置
pomelo-db -s mydb

# 查询（只读）
pomelo-db -d mydb -e "SELECT * FROM users"

# 表格输出
pomelo-db -d mydb -e "SELECT * FROM users" -o table

# 写操作
pomelo-db -d mydb -e "DELETE FROM users WHERE id=1" -w

# 执行 SQL 文件
pomelo-db -d mydb -f query.sql

# 设置超时（默认 30 秒）
pomelo-db -d mydb -t 60 -e "SELECT * FROM large_table"
```

## 配置数据源

### 方式一：-a 命令（推荐）

```bash
pomelo-db -a mydb=sqlite://./data/app.db
pomelo-db -a prod=mysql://user:pass@host:3306/db
```

配置写入当前目录 `.env`，格式为 `POMELO_DB_<NAME>=<DSN>`。

### 方式二：config.yaml

```yaml
query:
  datasources:
    mydb: "sqlite://./data/app.db"
    prod: "mysql://user:pass@host:3306/db"
```

### DSN 格式

```
<db-type>://<user>:<password>@<host>:<port>/<database>[?key=value&...]
```

| 类型 | 默认端口 | 示例 |
|------|---------|------|
| `mysql` | 3306 | `mysql://user:pass@host:3306/db?charset=utf8mb4` |
| `sqlserver` | 1433 | `sqlserver://user:pass@host:1433/db` |
| `dm` | 5236 | `dm://SYSDBA:pass@host:5236/db` |
| `vastbase` | 5432 | `vastbase://user:pass@host:5432/db?schema=public` |
| `opengauss` | 5432 | `opengauss://user:pass@host:5432/db?schema=public` |
| `sqlite` | - | `sqlite://./data.db` 或 `sqlite:///abs/path.db` |

密码含特殊字符需 URL 编码：`@` → `%40`，`:` → `%3A`，`/` → `%2F`，`#` → `%23`

## 故障排查

操作不允许（配置了 allowed_operators）：
```
Error: only [...] queries are allowed; use --write to allow write operations
```
解决：加 `-w` 参数，或在 `config.yaml` 中清空 `allowed_operators`。

数据源未找到：
```
Error: datasource 'xxx' not found
```
解决：运行 `pomelo-db -l` 确认名称，或用 `-a` 添加。
