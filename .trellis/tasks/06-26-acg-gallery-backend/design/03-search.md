# 设计 · 搜索（bleve CJK + 拼音）

> 依赖地基 §6.1 一致性。

## 6. 搜索（infra/search/bleve）

索引文档 id=image.id：

| 字段 | 分析器 | 来源 |
|---|---|---|
| filename | cjk | image.filename |
| tags | cjk | 该图标签名拼接 |
| filename_pinyin | standard | go-pinyin 全拼（空格分隔） |
| filename_first_letter | keyword | go-pinyin 首字母 |
| size | numeric | image.size |
| created_at | datetime | image.created_at |

要点（调研确认）：
- 必须 blank import `_ "github.com/blevesearch/bleve/v2/analysis/lang/cjk"` 注册 `cjk`。
- CJK bigram 分词，单字查询不匹配（设计如此）。
- 拼音无原生支持：索引时 `mozillazg/go-pinyin` 生成全拼（`pinyin.Normal`）+ 首字母（`pinyin.FirstLetter`）；非中文字符静默跳过。
- 排序 `SortBy([]string{"-created_at"})`；分页 `From/Size`。
- 标签变更需同步更新该图 bleve `tags` 字段。
- 查询路由：`filename`->DisjunctionQuery 打 filename/pinyin/first_letter；`tag`->match on tags；created_at/size 排序，标签排序走 DB。
