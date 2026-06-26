# 设计 · 图片同步与 COS

> 依赖地基 §3 数据模型、§6.1 一致性。

## 4. COS 集成（infra/client/cos）

- BaseURL `https://acgwarehouse-1301393037.cos.ap-shanghai.myqcloud.com`，bucket `acgwarehouse-1301393037`，region `ap-shanghai`，prefix `thumbnails/`。
- `cos.AuthorizationTransport{SecretID,SecretKey}` 凭证来自 conf（占位符，env `COS_SECRET_ID`/`COS_SECRET_KEY`）。
- 列举：`client.Bucket.Get(ctx,&cos.BucketGetOptions{Prefix,Marker,MaxKeys:1000})` 循环 `IsTruncated`+`NextMarker`，读 `Key/Size/LastModified`。
- URL：`fmt.Sprintf("%s/%s",baseURL,key)`。
- 凭证缺失/占位符 -> 同步时明确报错日志，不静默失败（AC-R0）。

## 5. 同步任务（cmd/sync，幂等）

1. conf 校验 COS 凭证非占位符，否则报错退出。
2. `ListObjects("thumbnails/")` 分页全量拉取。
3. 对每个对象用 `image.DecodeConfig`（只读文件头）解析 width/height；按 key 规则推断 category（规则未知则留空）。
4. 按 `cos_key` upsert 进 image（GORM `OnConflict{Columns:cos_key, DoUpdates:filename,size,last_modified,width,height,category}`）。
5. 更新 bleve 文档（含拼音字段）。
6. 输出统计：新增/更新/bleve 文档数。重复运行无重复（AC-R0）。
