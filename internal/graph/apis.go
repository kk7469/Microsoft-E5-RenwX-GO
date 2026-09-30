package graph

import "e5renewx/internal/model"

func Catalog() []model.APIDef {
	return []model.APIDef{
		{ID: "me", Name: "读取当前用户", Method: "GET", Path: "/me", Permission: "User.Read", Modes: []string{"login"}, Recommended: true, Description: "获取登录用户资料"},
		{ID: "users", Name: "列出用户", Method: "GET", Path: "/users?$top=5", Permission: "User.Read.All", Modes: []string{"login", "app"}, Recommended: true, Description: "读取组织用户列表"},
		{ID: "org", Name: "读取组织", Method: "GET", Path: "/organization", Permission: "Directory.Read.All", Modes: []string{"login", "app"}, Description: "读取租户组织信息"},
		{ID: "mail", Name: "读取邮件", Method: "GET", Path: "/me/messages?$top=5", Permission: "Mail.Read", Modes: []string{"login"}, Recommended: true, Description: "读取收件箱最近邮件"},
		{ID: "mail-user", Name: "读取指定用户邮件", Method: "GET", Path: "/users/{upn}/messages?$top=5", Permission: "Mail.Read", Modes: []string{"app"}, Recommended: true, Description: "以应用身份读取用户邮件"},
		{ID: "mailfolders", Name: "邮件文件夹", Method: "GET", Path: "/me/mailFolders", Permission: "Mail.Read", Modes: []string{"login"}, Recommended: true, Description: "列出邮件文件夹"},
		{ID: "sendmail", Name: "发送随机邮件", Method: "POST", Path: "/me/sendMail", Permission: "Mail.Send", Modes: []string{"login"}, RandomBody: true, Recommended: true, Description: "向自己发送一封随机内容邮件"},
		{ID: "calendar", Name: "读取日历", Method: "GET", Path: "/me/events?$top=5", Permission: "Calendars.Read", Modes: []string{"login"}, Recommended: true, Description: "读取日历事件"},
		{ID: "calendar-user", Name: "读取用户日历", Method: "GET", Path: "/users/{upn}/events?$top=5", Permission: "Calendars.Read", Modes: []string{"app"}, Recommended: true, Description: "读取指定用户日历"},
		{ID: "contacts", Name: "读取联系人", Method: "GET", Path: "/me/contacts?$top=5", Permission: "Contacts.Read", Modes: []string{"login"}, Recommended: true, Description: "读取联系人"},
		{ID: "drive", Name: "读取 OneDrive", Method: "GET", Path: "/me/drive", Permission: "Files.Read", Modes: []string{"login"}, Recommended: true, Description: "读取网盘根信息"},
		{ID: "drive-root", Name: "列出 OneDrive 根目录", Method: "GET", Path: "/me/drive/root/children?$top=10", Permission: "Files.Read", Modes: []string{"login"}, Recommended: true, Description: "列出网盘文件"},
		{ID: "drive-user", Name: "读取用户 OneDrive", Method: "GET", Path: "/users/{upn}/drive", Permission: "Files.Read.All", Modes: []string{"app"}, Recommended: true, Description: "读取用户网盘"},
		{ID: "drive-upload", Name: "上传随机文件", Method: "PUT", Path: "/me/drive/root:/ReGo/{name}.txt:/content", Permission: "Files.ReadWrite", Modes: []string{"login"}, RandomBody: true, Recommended: true, Description: "向 ReGo 目录上传随机文本"},
		{ID: "sites", Name: "搜索站点", Method: "GET", Path: "/sites?search=*", Permission: "Sites.Read.All", Modes: []string{"login", "app"}, Recommended: true, Description: "搜索 SharePoint 站点"},
		{ID: "groups", Name: "列出组", Method: "GET", Path: "/groups?$top=5", Permission: "Group.Read.All", Modes: []string{"login", "app"}, Description: "读取安全组/Office 组"},
		{ID: "notes", Name: "读取 OneNote", Method: "GET", Path: "/me/onenote/notebooks", Permission: "Notes.Read", Modes: []string{"login"}, Description: "读取笔记本"},
		{ID: "todo", Name: "读取待办列表", Method: "GET", Path: "/me/todo/lists", Permission: "Tasks.Read", Modes: []string{"login"}, Description: "读取 To Do 列表"},
		{ID: "planner", Name: "读取 Planner", Method: "GET", Path: "/me/planner/tasks", Permission: "Tasks.Read", Modes: []string{"login"}, Description: "读取 Planner 任务"},
		{ID: "people", Name: "相关人员", Method: "GET", Path: "/me/people?$top=5", Permission: "People.Read", Modes: []string{"login"}, Description: "读取相关联系人"},
		{ID: "outlook-master", Name: "Outlook 分类", Method: "GET", Path: "/me/outlook/masterCategories", Permission: "MailboxSettings.Read", Modes: []string{"login"}, Description: "读取邮箱分类"},
		{ID: "subscribed-skus", Name: "订阅 SKU", Method: "GET", Path: "/subscribedSkus", Permission: "Organization.Read.All", Modes: []string{"login", "app"}, Description: "读取已订阅许可证"},
		{ID: "domains", Name: "读取域名", Method: "GET", Path: "/domains", Permission: "Domain.Read.All", Modes: []string{"login", "app"}, Description: "读取租户域名"},
		{ID: "applications", Name: "列出应用", Method: "GET", Path: "/applications?$top=5", Permission: "Application.Read.All", Modes: []string{"login", "app"}, Description: "读取应用注册"},
		{ID: "service-principals", Name: "服务主体", Method: "GET", Path: "/servicePrincipals?$top=5", Permission: "Application.Read.All", Modes: []string{"login", "app"}, Description: "读取服务主体"},
		{ID: "directory-roles", Name: "目录角色", Method: "GET", Path: "/directoryRoles", Permission: "RoleManagement.Read.Directory", Modes: []string{"login", "app"}, Description: "读取目录角色"},
		{ID: "devices", Name: "列出设备", Method: "GET", Path: "/devices?$top=5", Permission: "Device.Read.All", Modes: []string{"login", "app"}, Description: "读取设备"},
		{ID: "audit-signins", Name: "登录日志", Method: "GET", Path: "/auditLogs/signIns?$top=5", Permission: "AuditLog.Read.All", Modes: []string{"login", "app"}, Description: "读取登录审计"},
		{ID: "sharepoint-lists", Name: "站点列表", Method: "GET", Path: "/sites/root/lists", Permission: "Sites.Read.All", Modes: []string{"login", "app"}, Recommended: true, Description: "读取根站点列表"},
		{ID: "drive-recent", Name: "最近文件", Method: "GET", Path: "/me/drive/recent", Permission: "Files.Read", Modes: []string{"login"}, Recommended: true, Description: "读取最近使用的文件"},
		{ID: "drive-shared", Name: "与我共享", Method: "GET", Path: "/me/drive/sharedWithMe", Permission: "Files.Read", Modes: []string{"login"}, Recommended: true, Description: "读取共享文件"},
		{ID: "mailbox-settings", Name: "邮箱设置", Method: "GET", Path: "/me/mailboxSettings", Permission: "MailboxSettings.Read", Modes: []string{"login"}, Description: "读取邮箱设置"},
		{ID: "member-of", Name: "所属组", Method: "GET", Path: "/me/memberOf", Permission: "Directory.Read.All", Modes: []string{"login"}, Description: "读取当前用户所属组"},
	}
}

func ByID(id string) (model.APIDef, bool) {
	for _, a := range Catalog() {
		if a.ID == id {
			return a, true
		}
	}
	return model.APIDef{}, false
}

func FilterByMode(mode string) []model.APIDef {
	out := make([]model.APIDef, 0)
	for _, a := range Catalog() {
		for _, m := range a.Modes {
			if m == mode {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

func DefaultIDs(mode string) []string {
	ids := make([]string, 0)
	for _, a := range FilterByMode(mode) {
		if a.Recommended {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		for _, a := range FilterByMode(mode) {
			ids = append(ids, a.ID)
		}
	}
	return ids
}
