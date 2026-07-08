# WeChat POST 签名校验

## 背景

WeChat 公众号回调的 GET 验证路径已经校验 `signature/timestamp/nonce`，但实际消息 POST 路径未校验签名。成熟 Agent 服务不能信任公网 POST body，否则攻击者可以伪造 openid/thread_id 触发 Agent、写入记忆或创建提醒。

## 方案

- 抽取共享 `verifyWechatSignature`。
- GET 验证和 POST 消息回调都复用同一签名校验。
- POST 校验失败时直接返回错误，不读取 XML body、不调用 Agent。
- `WECHAT_TOKEN` 缺失时返回服务端配置错误。

## 方案反思

- 当前仍未校验消息加解密模式下的密文签名和 `msg_signature`；如果后续开启安全模式，需要扩展。
- WeChat 回调没有纳入 API key middleware 是合理的，平台回调应使用平台签名机制而不是业务 API key。
