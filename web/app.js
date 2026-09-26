const app = document.getElementById("app");
const state = {
  user: null,
  page: "home",
  overview: null,
  accounts: [],
  logs: [],
  catalog: [],
  settings: null,
  toast: "",
  form: emptyForm(),
  editing: null,
};

function emptyForm() {
  return {
    name: "",
    upn: "",
    clientId: "",
    secret: "",
    tenant: "",
    mode: "login",
    notifyEmail: "",
    apiList: [],
  };
}

function toast(msg) {
  state.toast = msg;
  render();
  setTimeout(() => { state.toast = ""; render(); }, 2400);
}

async function api(path, opts = {}) {
  const res = await fetch(path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...(opts.headers || {}) },
    ...opts,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

async function boot() {
  try {
    await api("/api/me");
    state.user = { name: "admin" };
    await refreshAll();
  } catch {
    state.user = null;
  }
  render();
}

async function refreshAll() {
  const [overview, accounts, logs, catalog, settings] = await Promise.all([
    api("/api/overview"),
    api("/api/accounts"),
    api("/api/logs"),
    api("/api/catalog"),
    api("/api/settings"),
  ]);
  state.overview = overview;
  state.accounts = accounts;
  state.logs = logs;
  state.catalog = catalog;
  state.settings = settings;
}

function h(tag, attrs = {}, children = []) {
  const el = document.createElement(tag);
  Object.entries(attrs).forEach(([k, v]) => {
    if (k === "class") el.className = v;
    else if (k.startsWith("on") && typeof v === "function") el.addEventListener(k.slice(2).toLowerCase(), v);
    else if (k === "html") el.innerHTML = v;
    else if (k === "checked") el.checked = !!v;
    else if (v !== undefined && v !== null) el.setAttribute(k, v);
  });
  (Array.isArray(children) ? children : [children]).forEach((c) => {
    if (c == null || c === false) return;
    el.append(c.nodeType ? c : document.createTextNode(c));
  });
  return el;
}

function formatTime(v) {
  if (!v || v.startsWith("0001-01-01")) return "-";
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return "-";
  return d.toLocaleString();
}

function statusBadge(st) {
  const map = { running: ["ok", "正在运行"], paused: ["warn", "已暂停"], error: ["err", "错误"] };
  const [cls, text] = map[st] || ["info", st];
  return h("span", { class: `badge ${cls}` }, text);
}

function loginView() {
  const pwd = h("input", { type: "password", placeholder: "管理员密码", id: "pwd" });
  return h("div", { class: "login-wrap" }, [
    h("div", { class: "panel login-card" }, [
      h("h1", {}, "E5 Renew X"),
      h("p", { class: "muted" }, "Go 重写版 · Microsoft Graph 保活续订服务"),
      h("div", { class: "field" }, [h("label", {}, "管理员密码"), pwd]),
      h("button", {
        class: "btn",
        onClick: async () => {
          try {
            await api("/api/login", { method: "POST", body: JSON.stringify({ password: pwd.value }) });
            state.user = { name: "admin" };
            await refreshAll();
            render();
          } catch (e) {
            toast(e.message);
          }
        },
      }, "登录"),
      h("p", { class: "muted" }, "默认密码 123456，可在设置页或环境变量 ADMIN_PASSWORD 修改。"),
    ]),
  ]);
}

function navBtn(id, label) {
  return h("button", {
    class: `nav-btn ${state.page === id ? "active" : ""}`,
    onClick: () => { state.page = id; render(); },
  }, label);
}

function layout(content) {
  return h("div", { class: "layout" }, [
    h("aside", { class: "sidebar" }, [
      h("div", { class: "brand" }, ["E5 Renew X", h("small", {}, "Microsoft 365 Graph Keepalive")]),
      navBtn("home", "主页"),
      navBtn("accounts", "运行账号"),
      navBtn("logs", "调用日志"),
      navBtn("settings", "系统设置"),
      navBtn("about", "关于"),
      h("div", { class: "grow" }),
      h("button", {
        class: "nav-btn",
        onClick: async () => {
          await api("/api/logout", { method: "POST" });
          state.user = null;
          render();
        },
      }, "退出登录"),
    ]),
    h("main", { class: "main" }, [
      content,
      state.overview?.icpText ? h("div", { class: "footer" }, [
        h("a", { href: state.overview.icpLink || "#", target: "_blank" }, state.overview.icpText),
      ]) : null,
    ]),
  ]);
}

function homePage() {
  const o = state.overview || {};
  return h("div", {}, [
    h("div", { class: "topbar" }, [h("h1", {}, "系统概览")]),
    h("div", { class: "cards" }, [
      metric("运行账号", o.accounts || 0),
      metric("正在运行", o.running || 0),
      metric("成功调用", o.success || 0),
      metric("失败调用", o.fail || 0),
    ]),
    h("div", { class: "grid2" }, [
      h("div", { class: "panel" }, [
        h("h3", {}, "服务器配置"),
        p("操作系统", `${o.os || "-"} ${o.kernel || ""}`),
        p("主机名", o.hostname || "-"),
        p("应用版本", o.version || "-"),
        p("Go 版本", o.goVersion || "-"),
      ]),
      h("div", { class: "panel" }, [
        h("h3", {}, "资源占用"),
        p("CPU", `${o.cpuCount || 0} 核`),
        p("内存总容量", `${(o.memTotalGB || 0).toFixed(1)} GB`),
        p("磁盘总容量", `${(o.diskTotalGB || 0).toFixed(1)} GB`),
        p("运行时间", formatUptime(o.uptime || 0)),
      ]),
    ]),
    h("div", { class: "grid2" }, [
      h("div", { class: "panel" }, [
        h("h3", {}, "内存"),
        p("已使用", `${(o.memUsedGB || 0).toFixed(1)} GB / ${(o.memTotalGB || 0).toFixed(1)} GB`),
      ]),
      h("div", { class: "panel" }, [
        h("h3", {}, "磁盘"),
        p("可用空间", `${(o.diskFreeGB || 0).toFixed(1)} GB / ${(o.diskTotalGB || 0).toFixed(1)} GB`),
      ]),
    ]),
  ]);
}

function formatUptime(seconds) {
  if (!seconds) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天${hours}小时`;
  if (hours > 0) return `${hours}小时${mins}分钟`;
  return `${mins}分钟`;
}

function metric(label, value) {
  return h("div", { class: "card" }, [
    h("div", { class: "label" }, label),
    h("div", { class: "value" }, String(value)),
  ]);
}
function p(k, v) {
  return h("p", {}, [`${k}：`, h("b", {}, String(v ?? "-"))]);
}

function accountsPage() {
  return h("div", {}, [
    h("div", { class: "topbar" }, [
      h("h1", {}, "运行账号"),
      h("div", { class: "row" }, [
        h("button", { class: "btn ghost", onClick: () => {
          const form = emptyForm();
          form.apiList = recommendedIDs("login");
          state.form = form;
          state.editing = null;
          state.page = "edit";
          render();
        } }, "添加运行账号"),
        h("button", { class: "btn ghost", onClick: async () => { await api("/api/pardon", { method: "POST" }); await refreshAll(); toast("已特赦全部账号"); render(); } }, "全部恢复"),
      ]),
    ]),
    h("div", { class: "panel" }, [
      h("table", {}, [
        h("thead", {}, h("tr", {}, ["名称", "UPN", "模式", "状态", "下次调用", "成功/失败", "操作"].map((t) => h("th", {}, t)))),
        h("tbody", {}, state.accounts.length ? state.accounts.map(accountRow) : [h("tr", {}, h("td", { colspan: "7" }, "还没有运行账号"))]),
      ]),
    ]),
  ]);
}

function accountRow(a) {
  return h("tr", {}, [
    h("td", {}, a.name),
    h("td", {}, a.upn),
    h("td", {}, a.mode === "app" ? "非登录调用" : "登录调用"),
    h("td", {}, [statusBadge(a.status), a.lastError ? h("div", { class: "muted" }, a.lastError.slice(0, 80)) : null]),
    h("td", {}, formatTime(a.nextRunAt)),
    h("td", {}, `${a.successCount} / ${a.failCount}`),
    h("td", {}, h("div", { class: "row" }, [
      h("button", { class: "btn ghost", onClick: () => runNow(a.id) }, "立即调用"),
      a.status === "running"
        ? h("button", { class: "btn ghost", onClick: () => act(a.id, "pause") }, "暂停")
        : h("button", { class: "btn ghost", onClick: () => act(a.id, "resume") }, "恢复"),
      h("button", { class: "btn ghost", onClick: () => editAccount(a) }, "编辑"),
      h("button", { class: "btn danger", onClick: () => delAccount(a.id) }, "删除"),
    ])),
  ]);
}

async function runNow(id) {
  await api(`/api/accounts/${id}/run`, { method: "POST" });
  toast("已触发立即调用");
  setTimeout(async () => { await refreshAll(); render(); }, 1200);
}
async function act(id, action) {
  await api(`/api/accounts/${id}/${action}`, { method: "POST" });
  await refreshAll();
  render();
}
async function delAccount(id) {
  if (!confirm("确认删除该账号？")) return;
  await api(`/api/accounts/${id}`, { method: "DELETE" });
  await refreshAll();
  render();
}
function editAccount(a) {
  state.editing = a.id;
  state.form = {
    name: a.name || "",
    upn: a.upn || "",
    clientId: a.clientId || "",
    secret: "",
    tenant: a.tenant || "",
    mode: a.mode || "login",
    notifyEmail: a.notifyEmail || "",
    apiList: a.apiList || [],
  };
  state.page = "edit";
  render();
}

function editPage() {
  const f = state.form;
  const catalog = state.catalog.filter((x) => x.modes.includes(f.mode));
  const selected = new Set(f.apiList);
  return h("div", {}, [
    h("div", { class: "topbar" }, [h("h1", {}, state.editing ? "编辑运行账号" : "添加运行账号")]),
    h("div", { class: "panel" }, [
      h("div", { class: "grid2" }, [
        field("显示名称", input("name", f.name)),
        field("MS365 账号 (UPN)", input("upn", f.upn, "user@tenant.onmicrosoft.com")),
        field("应用程序（客户端）ID", input("clientId", f.clientId)),
        field("账号密码 / 客户端机密", input("secret", f.secret, state.editing ? "留空则不修改" : "", "password")),
        field("租户 ID / 域名（可空，自动用账号域名）", input("tenant", f.tenant, "目录 ID 或 xxx.onmicrosoft.com")),
        field("通知邮箱", input("notifyEmail", f.notifyEmail)),
      ]),
      h("div", { class: "field" }, [
        h("label", {}, "调用模式"),
        h("select", {
          onChange: (e) => {
            state.form.mode = e.target.value;
            state.form.apiList = recommendedIDs(e.target.value);
            render();
          },
        }, [
          option("login", "登录调用（账号密码 + 委托权限）", f.mode === "login"),
          option("app", "非登录调用（客户端机密 + 应用程序权限）", f.mode === "app"),
        ]),
      ]),
      h("div", { class: "field" }, [
        h("label", {}, "Graph API 列表（随机抽取调用）"),
        h("div", { class: "row", style: "margin-bottom:8px" }, [
          h("button", { class: "btn ghost", type: "button", onClick: () => { state.form.apiList = recommendedIDs(f.mode); render(); } }, "勾选推荐"),
          h("button", { class: "btn ghost", type: "button", onClick: () => { state.form.apiList = catalog.map((x) => x.id); render(); } }, "全选"),
        ]),
        h("div", { class: "api-list" }, catalog.map((apiItem) => h("label", { class: "api-item" }, [
          h("input", {
            type: "checkbox",
            checked: selected.has(apiItem.id),
            onChange: (e) => {
              const set = new Set(state.form.apiList);
              if (e.target.checked) set.add(apiItem.id);
              else set.delete(apiItem.id);
              state.form.apiList = [...set];
            },
          }),
          `${apiItem.name}  ·  ${apiItem.method} ${apiItem.path}  ·  ${apiItem.permission}${apiItem.recommended ? "  ·  推荐" : ""}`,
        ]))),
      ]),
      h("div", { class: "row" }, [
        h("button", { class: "btn", onClick: saveAccount }, "保存"),
        h("button", { class: "btn ghost", onClick: () => { state.page = "accounts"; render(); } }, "返回"),
      ]),
    ]),
  ]);
}

function catalogFor(mode) {
  return state.catalog.filter((x) => x.modes.includes(mode));
}
function recommendedIDs(mode) {
  const list = catalogFor(mode);
  const rec = list.filter((x) => x.recommended).map((x) => x.id);
  return rec.length ? rec : list.map((x) => x.id);
}
function field(label, el) {
  return h("div", { class: "field" }, [h("label", {}, label), el]);
}
function input(key, value, placeholder = "", type = "text") {
  return h("input", {
    type,
    value,
    placeholder,
    onInput: (e) => { state.form[key] = e.target.value; },
  });
}
function option(value, text, selected) {
  const o = h("option", { value }, text);
  if (selected) o.selected = true;
  return o;
}

async function saveAccount() {
  const body = { ...state.form };
  if (!body.apiList.length) body.apiList = recommendedIDs(body.mode);
  try {
    if (state.editing) {
      await api(`/api/accounts/${state.editing}`, { method: "PUT", body: JSON.stringify(body) });
    } else {
      await api("/api/accounts", { method: "POST", body: JSON.stringify(body) });
    }
    state.page = "accounts";
    await refreshAll();
    toast("已保存");
    render();
  } catch (e) {
    toast(e.message);
  }
}

function logsPage() {
  return h("div", {}, [
    h("div", { class: "topbar" }, [h("h1", {}, "调用日志")]),
    h("div", { class: "panel" }, [
      h("table", {}, [
        h("thead", {}, h("tr", {}, ["时间", "账号", "API", "结果", "耗时", "说明"].map((t) => h("th", {}, t)))),
        h("tbody", {}, state.logs.map((l) => h("tr", {}, [
          h("td", {}, formatTime(l.createdAt)),
          h("td", {}, l.accountId),
          h("td", {}, `${l.apiName || l.apiId}`),
          h("td", {}, h("span", { class: `badge ${l.ok ? "ok" : (l.skipped ? "warn" : "err")}` }, l.ok ? `成功 ${l.status}` : (l.skipped ? `跳过 ${l.status || ""}` : `失败 ${l.status || ""}`))),
          h("td", {}, `${l.durationMs || 0} ms`),
          h("td", {}, (l.message || "").slice(0, 160)),
        ]))),
      ]),
    ]),
  ]);
}

function settingsPage() {
  const s = { ...state.settings };
  const bind = (key, el) => {
    el.addEventListener("input", (e) => { s[key] = e.target.value; });
    return el;
  };
  const admin = bind("adminPassword", h("input", { type: "password", placeholder: "留空不修改" }));
  const notify = bind("notifyEmail", h("input", { value: s.notifyEmail || "" }));
  const smtpHost = bind("smtpHost", h("input", { value: s.smtpHost || "" }));
  const smtpPort = bind("smtpPort", h("input", { value: s.smtpPort || 465 }));
  const smtpUser = bind("smtpUser", h("input", { value: s.smtpUser || "" }));
  const smtpPass = bind("smtpPassword", h("input", { type: "password", placeholder: "留空不修改" }));
  const smtpFrom = bind("smtpFrom", h("input", { value: s.smtpFrom || "" }));
  const minI = bind("minIntervalSec", h("input", { value: s.minIntervalSec }));
  const maxI = bind("maxIntervalSec", h("input", { value: s.maxIntervalSec }));
  const maxA = bind("maxApisPerRound", h("input", { value: s.maxApisPerRound }));
  const failT = bind("failPauseThreshold", h("input", { value: s.failPauseThreshold }));
  const resumeH = bind("autoResumeHours", h("input", { value: s.autoResumeHours }));
  const pardon = bind("pardonIntervalDays", h("input", { value: s.pardonIntervalDays }));
  const hour = bind("dailyReportHour", h("input", { value: s.dailyReportHour }));
  const site = bind("siteName", h("input", { value: s.siteName || "" }));
  const icp = bind("icpText", h("input", { value: s.icpText || "" }));
  const icpLink = bind("icpLink", h("input", { value: s.icpLink || "" }));
  const notice = bind("notice", h("textarea", { rows: "4" }, s.notice || ""));
  return h("div", {}, [
    h("div", { class: "topbar" }, [h("h1", {}, "系统设置")]),
    h("div", { class: "panel" }, [
      h("div", { class: "grid2" }, [
        field("管理员密码", admin),
        field("站点名称", site),
        field("通知邮箱", notify),
        field("SMTP Host", smtpHost),
        field("SMTP Port", smtpPort),
        field("SMTP 用户", smtpUser),
        field("SMTP 密码", smtpPass),
        field("发件人", smtpFrom),
        field("最小间隔(秒)", minI),
        field("最大间隔(秒)", maxI),
        field("每轮最多 API 数", maxA),
        field("连续失败暂停阈值", failT),
        field("自动恢复(小时)", resumeH),
        field("特赦间隔(天)", pardon),
        field("日报小时(0-23)", hour),
        field("ICP 备案号", icp),
        field("ICP 链接", icpLink),
      ]),
      field("公告", notice),
      h("button", {
        class: "btn",
        onClick: async () => {
          const body = {
            ...s,
            smtpPort: Number(s.smtpPort || 465),
            minIntervalSec: Number(s.minIntervalSec),
            maxIntervalSec: Number(s.maxIntervalSec),
            maxApisPerRound: Number(s.maxApisPerRound),
            failPauseThreshold: Number(s.failPauseThreshold),
            autoResumeHours: Number(s.autoResumeHours),
            pardonIntervalDays: Number(s.pardonIntervalDays),
            dailyReportHour: Number(s.dailyReportHour),
          };
          try {
            await api("/api/settings", { method: "PUT", body: JSON.stringify(body) });
            await refreshAll();
            toast("设置已保存");
            render();
          } catch (e) {
            toast(e.message);
          }
        },
      }, "保存设置"),
    ]),
  ]);
}

function aboutPage() {
  return h("div", {}, [
    h("div", { class: "topbar" }, [h("h1", {}, "关于")]),
    h("div", { class: "panel notice" }, [
      h("p", {}, "这是 Microsoft 365 E5 Renew X 的 Go 重写版，用于通过随机调用 Microsoft Graph API 保持 E5 开发者订阅活跃。"),
      h("p", {}, "对齐原项目能力：登录/非登录两种调用、多账号托管、随机 API、1000-2000 秒间隔、邮件通知、错误暂停、定时特赦恢复、ICP 与公告。"),
      h("p", {}, "委托权限（登录调用）建议：User.Read, Mail.Read, Mail.Send, Files.ReadWrite, Calendars.Read, Contacts.Read, Sites.Read.All, Group.Read.All 等。"),
      h("p", {}, "应用程序权限（非登录调用）建议：User.Read.All, Mail.Read, Files.Read.All, Directory.Read.All, Sites.Read.All 等，并授予管理员同意。"),
      h("p", {}, "原项目参考：hongyonghan/Docker_Microsoft365_E5_Renew_X、SundayRX Microsoft 365 E5 Renew X。"),
    ]),
  ]);
}

function render() {
  app.innerHTML = "";
  if (!state.user) {
    app.append(loginView());
  } else {
    const pages = { home: homePage, accounts: accountsPage, logs: logsPage, settings: settingsPage, about: aboutPage, edit: editPage };
    app.append(layout((pages[state.page] || homePage)()));
  }
  if (state.toast) app.append(h("div", { class: "toast" }, state.toast));
}

boot();
setInterval(async () => {
  if (!state.user) return;
  try { await refreshAll(); render(); } catch {}
}, 15000);
