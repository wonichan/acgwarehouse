# ACGWarehouse Frontend

Vue 3 + TypeScript + Vite 前端。页面含图库首页、详情、搜索、热榜、收藏夹、账户中心；接口统一走 `/api/v1`。

## 启动

```powershell
npm install
npm run dev
```

构建与预览：

```powershell
npm run build
npm run preview
```

## 接口

API 封装在 `src/api/`：

- `client.ts` 暴露业务方法与类型
- `transport.ts` 处理 `/api/v1`、JWT token、错误解包

开发代理在 `vite.config.ts`。若后端跑在本机：

```ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

## 路由部署

本项目使用 Vue Router history mode。静态部署须保留 `public/_redirects`：

```text
/api/* /api/:splat 200
/* /index.html 200
```

`/api/*` 在前，前端兜底在后。
