# Amazon 详情页采集助手

## 功能
- 仅支持 `Amazon 详情页`：`amazon.com / amazon.ca / amazon.com.mx`
- 页面右下角悬浮按钮一键采集
- 插件弹窗可查看当前页识别结果
- 通过系统 `API Token` 直连后端 `/amazonCollector/extension/upsertDetail`

## 安装
1. 打开 Chrome `扩展程序`
2. 开启 `开发者模式`
3. 选择 `加载已解压的扩展程序`
4. 选择当前目录 `extensions/amazon-collector`

## 配置
1. 打开插件 `选项`
2. 填写 `后台 API Base URL`
3. 填写系统签发的 `API Token`
4. 点击 `测试连接`

## 使用
1. 打开 Amazon 详情页
2. 点击右下角 `采集到系统`
3. 回到后台 `Amazon 工具 / 采集商品列表` 查看结果

## 说明
- 插件固定扩展 ID，对应 Origin 形如 `chrome-extension://<extension-id>`
- 如果你的部署对扩展请求做了额外来源限制，请把该 Origin 加入后端白名单
