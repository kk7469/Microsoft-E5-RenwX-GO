# Microsoft 365 E5 RenewX GO

这是 Microsoft 365 E5 RenewX GO 的 Go 重写版，用于通过随机调用 Microsoft Graph API 保持 E5 开发者订阅活跃。

原项目地址： <https://github.com/hongyonghan/Docker_Microsoft365_E5_Renew_X>

本项目地址： <https://github.com/kk7469/microsoft-e5-renwx-go>

账号注册相关请参考原作者博客： <https://blog.csdn.net/qq_33212020/article/details/119747634>

以及更详细的教程： <https://blog.csdn.net/CingSyuan/article/details/155430662>

## 说明

已从 API 目录移除 8 个在 E5 开发者租户上稳定报错的接口，原因如下（不是程序 Bug）：

- `用户头像` `/me/photo`：账号未设置头像时官方固定返回 **404**；租户限制头像访问时返回 **403**。
- `邮件活跃报告` / `OneDrive 使用报告` `/reports/...`：返回 **403**。除了 `Reports.Read.All`，账号还必须被授予 Entra 管理角色（Reports Reader / Global Reader / Exchange Administrator 等），普通 E5 开发账号通常没有。
- `安全告警` `/security/alerts_v2`：返回 **403**。需要 `SecurityAlert.Read.All`，且账号需具备 Security Reader / Global Reader 等角色，租户还需接入 Microsoft 365 Defender。
- `登录日志` `/auditLogs/signIns`：返回 **403**。除了 `AuditLog.Read.All`，账号还必须具备 Global Reader / Security Reader / Security Administrator / Security Operator / Reports Reader 之一。
- `读取聊天` `/me/chats`、`加入的团队` `/me/joinedTeams`、`在线状态` `/me/presence`：返回 **403**。微软官方确认：账号未被分配 **Teams 许可证**时，即便 `Chat.Read` / `Team.ReadBasic.All` / `Presence.Read` 已授权也会 403，分配许可证后即恢复正常。

## 功能

- 管理员密码登录（默认 `123456`。
- 多运行账号托管
- 登录调用（ROPC：账号 + 密码 + 客户端 ID）
- 非登录调用（client credentials：客户端 ID + 客户端机密）
- 32 个 Microsoft Graph API，按模式随机抽取
- 随机调用间隔 600–1500 秒
- 随机邮件内容和 OneDrive 上传内容
- 连续失败自动暂停，到期自动恢复
- 定期特赦全部账号
- SMTP 异常通知与每日 18 点运行报告
- ICP 备案与站点公告
- JSON 文件存储，无需数据库
- 调用日志仅保留最近 200 条于内存，不写入文件，重启后清空
- 账号运行时状态（状态、成功/失败计数、令牌、调度时间）仅存内存，重启后重置为「运行中」

## 启动

```
# 默认端口 1066，管理员密码 123456
go run .

# 或指定环境变量
PORT=1066 ADMIN_PASSWORD=123456 DATA_PATH=data/store.json go run .
```

访问 `http://localhost:1066`

## Docker

```
# 拉库
git clone https://github.com/kk7469/microsoft-e5-renwx-go.git
cd microsoft-e5-renwx-go
# 构建镜像，后面的.不能删除
docker build -t e5renewx-go:latest .
# 运行容器
docker run -d \
  --name e5renewx-go \
  --restart unless-stopped \
  -p 1066:1066 \
  -e TZ=Asia/Shanghai \
  -e ADMIN_PASSWORD=你的密码 \
  -e SESSION_SECRET=自定义随机字符串 \
  -v /opt/microsoft-e5-renwx-go/data:/app/data \
  e5renewx-go:latest
```

## 添加运行账号

1. 在 Entra 注册应用，开启公共客户端，按模式授予 Graph 权限并管理员同意
2. 登录调用填写：E5 账号、客户端 ID、账号密码
3. 非登录调用填写：E5 账号、客户端 ID、客户端机密、租户 ID
