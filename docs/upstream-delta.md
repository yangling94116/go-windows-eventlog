# v0.1.2 上游增量清单

## 基线

- 独立库：`v0.1.1`
- 独立库 commit：`2dfca6367ad3351ce3f76d32082098d1dd689f13`
- Elastic 起点：`v9.2.1`，commit `6485e11edf8854d7792dfc2999bf19db37315ea4`
- Elastic 终点：`v9.2.7`，commit `3147cc0802c718c131386d967c5eaf5fe9f94e7f`

## 文件决策

| Elastic 文件 | 决策 | 独立库映射或原因 |
|---|---|---|
| `eventlog/config.go` | 适配 | 将上游 simple query 数据映射到现有 `eventlog.Config` |
| `eventlog/errors_unix.go` | 排除 | 独立库非 Windows stub 不引用本次新增内部错误 |
| `eventlog/errors_windows.go` | 移植 | 增加上游新增的可恢复错误 |
| `eventlog/record_filter.go` | 移植 | 新增 `pkg/eventlog/record_filter.go` |
| `eventlog/record_filter_test.go` | 适配 | 替换 Beats 类型 import |
| `eventlog/runner.go` | 排除 | 独立库不包含 Beats runner、publisher 和 status |
| `eventlog/wineventlog.go` | 适配 | 移植读取、filter 和恢复逻辑到现有公开 API |
| `eventlog/wineventlog_retry_test.go` | 适配 | 增加独立库恢复行为测试 |
| `eventlog/wineventlog_test.go` | 适配 | 只移植与本次行为变化相关的断言 |
| `sys/wineventlog/iterator.go` | 移植 | 移植 Windows iterator 恢复逻辑 |
| `sys/wineventlog/iterator_test.go` | 移植 | 移植对应测试 |
| `sys/wineventlog/query.go` | 保留 | `wineventlog.Query` 是 v0.1.1 公开兼容 API |
| `sys/wineventlog/query_test.go` | 保留 | 继续验证兼容 API 行为 |
| `sys/wineventlog/util_test.go` | 按需适配 | 只接受与上游测试辅助逻辑相关的变化 |

## 约束

- 不复制 v9.2.7 完整目录。
- 不引入 Beats 内部依赖。
- 不纳入第一套候选 review 发现的额外修正。
- 所有变更保持未提交，等待用户 review。

## 验证记录

- Windows amd64 全包编译通过。
- Windows 386 全包编译通过。
- `go vet ./...` 通过。
- 新增 record filter、iterator recovery、render retry 和 record gap 测试通过。
- `wineventlog.Query` 兼容测试通过。
- Heka `heka-plugins/winlog` 使用本地替换后编译通过。
- 基线仓库未包含被忽略的 EVTX fixture，依赖 `testdata/4752.evtx` 的既有测试无法运行；本增量方案不补充 fixture。

## 与重新抽取候选的差异

- 本候选只修改与上游增量直接相关的既有文件，不替换完整目录。
- 本候选保留 `wineventlog.Query`、原有配置校验和原有测试资产状态。
- 本候选不包含重新抽取候选中的 checkpoint 注释清理、MapStr 清理、EVTX fixture 和 `.gitignore` 调整。
- 两套候选都实现普通读取使用 `*` query、Go 层 record filter、iterator recovery、render retry 和 record ID gap recovery。
