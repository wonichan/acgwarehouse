## 阶段 2 · COS 同步 + bleve 基础

- [ ] infra/client/cos：NewClient（占位凭证校验）、ListObjects（分页）、ObjectURL。
- [ ] infra/search/bleve.go：建/开索引、文档 mapping（filename/tags cjk、pinyin/first_letter、size、created_at）、Index/Delete/Search 封装；blank import cjk。
- [ ] pkg/pinyin：全拼 + 首字母生成。
- [ ] po/do: image；repository/image：upsert by cos_key、按条件查询 + 排序 + 分页。
- [ ] cmd/sync/main.go：拉 COS -> 解析宽高/分类 -> upsert image -> 建 bleve 文档；统计日志。支持 `--reindex` 从 SQLite 全量重建 bleve（一致性兜底）。
- 验证：占位凭证报错；（配置真实凭证后）sync 幂等、bleve 文档数 == image 数（AC-R0）。

- [ ] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
