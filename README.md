# pomelo-db

轻量级数据库查询 CLI，单一二进制，无运行时依赖。支持 MySQL、SQL Server、达梦、OpenGauss/Vastbase、SQLite。

## 安装

```bash
make build   # 当前平台
# 或
make build-linux   # Linux AMD64 + ARM64（需要 Docker）
```

## 添加数据源

用 `-a` 直接添加，配置写入当前目录的 `.env`：

```bash
pomelo-db -a mydb=mysql://user:pass@host:3306/dbname
pomelo-db -a local=sqlite://./data.db
pomelo-db -r mydb   # 删除
pomelo-db -l        # 列出所有
```

DSN 格式为 `<type>://user:pass@host:port/db`，支持的类型：`mysql` `sqlserver` `dm` `opengauss` `vastbase` `sqlite`。

不熟悉 DSN 格式？直接告诉 AI 你的数据库信息，让它帮你生成命令。想了解更多，翻看 [skills](.claude/skills/pomelo-db/README.md)

## 查询

```bash
pomelo-db -d mydb -e "SELECT * FROM users LIMIT 10"
pomelo-db -d mydb -e "SELECT * FROM users" -o table   # 表格输出
pomelo-db -d mydb -f query.sql                        # 从文件读取
pomelo-db -d mydb -e "DELETE FROM t WHERE id=1" -w    # 写操作需加 -w
```

## Claude Code Skill

安装后可在 Claude Code 中直接用自然语言查询数据库：

```bash
make skill
```

## 开发

```bash
make deps        # 安装依赖
make test        # 运行测试
make test cov=1  # 运行测试并生成覆盖率报告（coverage.html）
make lint        # 格式化 + lint（自动修复）
```

## 许可证

MIT License - 查看 [LICENSE](LICENSE) 了解详情。
