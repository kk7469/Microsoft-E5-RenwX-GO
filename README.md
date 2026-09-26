# Microsoft 365 E5 Renew X (Go)

用 Go 重写的 Microsoft 365 E5 Graph API 保活续订服务，对齐原项目核心能力。

原项目：https://github.com/hongyonghan/Docker_Microsoft365_E5_Renew_X
账号注册相关请参考原作者博客：
https://blog.csdn.net/qq_33212020/article/details/119747634
## 功能

- 管理员密码登录（默认 `123456`，路由等价 `/Admin/Login`）
- 多运行账号托管
- 登录调用（ROPC：账号 + 密码 + 客户端 ID）
- 非登录调用（client credentials：客户端 ID + 客户端机密）
- 41 个 Microsoft Graph API，按模式随机抽取
- 随机调用间隔 1000–2000 秒
- 随机邮件内容和 OneDrive 上传内容
- 连续失败自动暂停，到期自动恢复
- 定期特赦全部账号
- SMTP 异常通知与每日 18 点运行报告
- ICP 备案与站点公告
- JSON 文件存储，无需数据库

## 启动

```bash
# 默认端口 1066，管理员密码 123456
go run .

# 或指定环境变量
PORT=1066 ADMIN_PASSWORD=123456 DATA_PATH=data/store.json go run .
```

访问 `http://localhost:1066`

## Docker
```bash
# 拉库
git clone https://github.com/kk7469/Microsoft-E5-RenwX-GO.git
cd Microsoft-E5-RenwX-GO
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
  -v /opt/Microsoft-E5-RenwX-GO/data:/app/data \
  e5renewx-go:latest
```

## 添加运行账号

1. 在 Entra 注册应用，开启公共客户端，按模式授予 Graph 权限并管理员同意
2. 登录调用填写：E5 账号、客户端 ID、账号密码
3. 非登录调用填写：E5 账号、客户端 ID、客户端机密、租户 ID

登录调用建议权限：`User.Read`、`Mail.Read`、`Mail.Send`、`Files.ReadWrite`、`Calendars.Read`、`Sites.Read.All`

非登录调用建议权限：`User.Read.All`、`Mail.Read`、`Files.Read.All`、`Directory.Read.All`、`Sites.Read.All`
