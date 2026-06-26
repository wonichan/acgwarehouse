## 阶段 2 · COS 同步 + bleve 基础

- [x] infra/client/cos：NewClient（占位凭证校验）、ListObjects（分页）、ObjectURL。
- [x] infra/search/bleve.go：建/开索引、文档 mapping（filename/tags cjk、pinyin/first_letter、size、created_at）、Index/Delete/Search 封装；blank import cjk。
- [x] pkg/pinyin：全拼 + 首字母生成。
- [x] po/do: image；repository/image：upsert by cos_key、按条件查询 + 排序 + 分页。
- [x] cmd/sync/main.go：拉 COS -> 解析宽高/分类 -> upsert image -> 建 bleve 文档；统计日志。支持 `--reindex` 从 SQLite 全量重建 bleve（一致性兜底）。
- 验证：占位凭证报错；空库 `--reindex` 成功；`go test ./...`、`go build ./...`、`go vet ./...`、`gofmt -s -l .` 通过。真实 COS sync 幂等和 bleve 文档数对账需配置真实凭证后执行。

- [x] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
