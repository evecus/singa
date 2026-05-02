<template>
  <div class="page">
    <div class="topbar">
      <span class="topbar-title">设置</span>
    </div>
    <div class="page" style="display:flex;flex-direction:column;gap:16px">

      <!-- ══ 账号与验证 ════════════════════════════════════════════════ -->
      <div class="card">
        <div class="card-title">账号与验证</div>
        <div class="field-hint" style="margin-bottom:12px">
          开启后，访问 Web 界面需要输入用户名和密码。
        </div>
        <div style="display:flex;flex-direction:column;gap:10px;margin-bottom:14px">
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px">
            <div class="toggle" :class="{ on: authEnabled }" @click="authEnabled=!authEnabled"></div>
            <span>启用登录验证（默认开启）</span>
          </label>
        </div>
        <template v-if="authEnabled">
          <div class="grid-2 gap-3" style="margin-bottom:10px">
            <div class="field">
              <label class="field-label">用户名</label>
              <input class="input" v-model="authUsername" placeholder="admin" />
            </div>
            <div class="field">
              <label class="field-label">新密码（留空则不修改）</label>
              <input class="input" v-model="authPassword" type="password" placeholder="新密码" autocomplete="new-password" />
            </div>
          </div>
        </template>
        <div class="flex gap-2">
          <button class="btn btn-primary" @click="saveAuth">保存</button>
          <span v-if="authMsg" class="text-xs" :class="authMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ authMsg }}</span>
        </div>
      </div>

      <!-- ══ 配置文件配置 ══════════════════════════════════════════════ -->
      <div class="card">
        <div class="card-title">配置文件设置</div>

        <!-- 入站配置 -->
        <div class="section-sub-title">Inbound 端口配置</div>
        <div class="field-hint" style="margin-bottom:12px">
          各 inbound 监听端口及 TUN 网卡名称。保存后下次启动核心时生效，影响全部配置模式。
        </div>
        <div class="grid-2 gap-3" style="margin-bottom:10px">
          <div class="field">
            <label class="field-label">DNS 端口（dns-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.dnsPort" placeholder="5356" />
          </div>
          <div class="field">
            <label class="field-label">混合代理端口（mixed-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.mixedPort" placeholder="2081" />
          </div>
          <div class="field">
            <label class="field-label">Redirect 端口（redirect-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.redirectPort" placeholder="7892" />
          </div>
          <div class="field">
            <label class="field-label">TProxy 端口（tproxy-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.tproxyPort" placeholder="7893" />
          </div>
          <div class="field">
            <label class="field-label">TUN 网卡名称</label>
            <input class="input input-mono" v-model="ib.tunInterface" placeholder="singa" />
          </div>
          <div class="field">
            <label class="field-label">TUN 地址（每行一个）</label>
            <textarea class="textarea input-mono" v-model="ib.tunAddressText" rows="2"
              placeholder="172.31.0.1/30&#10;fdfe:dcba:9876::1/126"></textarea>
          </div>
        </div>
        <div class="flex gap-2" style="margin-bottom:4px">
          <button class="btn btn-ghost btn-sm" @click="saveSingaSettings">保存</button>
          <span v-if="singaMsg" class="text-xs" :class="singaMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ singaMsg }}</span>
        </div>

        <hr class="section-divider" />

        <!-- Experimental -->
        <div class="section-sub-title">Experimental</div>
        <div class="field-hint" style="margin-bottom:12px">
          覆盖全部配置模式的 experimental 块（cache_file + clash_api）。
        </div>
        <div style="margin-bottom:14px">
          <div class="field-label" style="margin-bottom:6px">缓存文件</div>
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:8px">
            <div class="toggle" :class="{ on: exp.cacheEnabled }" @click="exp.cacheEnabled=!exp.cacheEnabled"></div>
            <span>启用缓存</span>
          </label>
          <div class="field" v-if="exp.cacheEnabled">
            <label class="field-label">缓存路径</label>
            <input class="input input-mono" v-model="exp.cachePath" placeholder="cache.db" />
          </div>
        </div>
        <div style="margin-bottom:10px">
          <div class="field-label" style="margin-bottom:6px">Clash API</div>
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:8px">
            <div class="toggle" :class="{ on: exp.clashAPIEnabled }" @click="exp.clashAPIEnabled=!exp.clashAPIEnabled"></div>
            <span>启用 Clash API</span>
          </label>
          <div v-if="exp.clashAPIEnabled" class="grid-2 gap-3">
            <div class="field">
              <label class="field-label">监听地址</label>
              <input class="input input-mono" v-model="exp.clashAPIListen" placeholder="0.0.0.0:9090" />
            </div>
            <div class="field">
              <label class="field-label">UI 路径</label>
              <input class="input input-mono" v-model="exp.clashAPIUI" placeholder="ui" />
            </div>
            <div class="field">
              <label class="field-label">UI 下载地址</label>
              <input class="input input-mono" v-model="exp.clashAPIUIURL"
                placeholder="https://fastly.jsdelivr.net/gh/..." />
            </div>
            <div class="field">
              <label class="field-label">下载出站（outbound tag）</label>
              <input class="input input-mono" v-model="exp.clashAPIDetour" placeholder="direct" />
            </div>
            <div class="field">
              <label class="field-label">默认模式</label>
              <div class="seg">
                <button v-for="m in clashModes" :key="m.v"
                  class="seg-btn" :class="{ on: exp.clashAPIMode===m.v }"
                  @click="exp.clashAPIMode=m.v">{{ m.l }}</button>
              </div>
            </div>
          </div>
        </div>
        <div class="flex gap-2" style="margin-bottom:4px">
          <button class="btn btn-ghost btn-sm" @click="saveSingaSettings">保存</button>
          <span v-if="singaMsg" class="text-xs" :class="singaMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ singaMsg }}</span>
        </div>

        <hr class="section-divider" />

        <!-- 日志 -->
        <div class="section-sub-title">日志</div>
        <div class="field-hint" style="margin-bottom:12px">
          覆盖全部配置模式的 log 块。
        </div>
        <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:10px">
          <div class="toggle" :class="{ on: !logCfg.disabled }" @click="logCfg.disabled=!logCfg.disabled"></div>
          <span>启用日志</span>
        </label>
        <div v-if="!logCfg.disabled" class="field" style="margin-bottom:10px">
          <label class="field-label">日志等级</label>
          <div class="seg">
            <button v-for="l in logLevels" :key="l"
              class="seg-btn" :class="{ on: logCfg.level===l }"
              @click="logCfg.level=l">{{ l }}</button>
          </div>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" @click="saveSingaSettings">保存</button>
          <span v-if="singaMsg" class="text-xs" :class="singaMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ singaMsg }}</span>
        </div>
      </div>

      <!-- ══ 代理模式 ══════════════════════════════════════════════════ -->
      <div class="card">
        <div class="card-title">代理设置</div>

        <!-- 代理模式细化 -->
        <div class="section-sub-title">代理模式细化</div>
        <div class="field-hint" style="margin-bottom:12px">
          此处设置将覆盖仪表盘快速选择的代理模式，提供更精细的 TCP/UDP 控制。
        </div>
        <div class="grid-2 gap-3">
          <div class="field">
            <label class="field-label">TCP 代理模式</label>
            <div class="seg" style="flex-direction:column;border-radius:var(--radius);overflow:hidden">
              <button v-for="m in tcpModeOpts" :key="m.v"
                class="seg-btn" :class="{ on: tcpMode===m.v }"
                style="border-right:none;border-bottom:1px solid var(--border2);text-align:left;padding:8px 12px"
                @click="tcpMode=m.v">
                <span style="font-weight:700">{{ m.l }}</span>
                <span style="font-size:10px;color:var(--text3);margin-left:6px">{{ m.desc }}</span>
              </button>
            </div>
          </div>
          <div class="field">
            <label class="field-label">UDP 代理模式</label>
            <div class="seg" style="flex-direction:column;border-radius:var(--radius);overflow:hidden">
              <button v-for="m in udpModeOpts" :key="m.v"
                class="seg-btn" :class="{ on: udpMode===m.v }"
                style="border-right:none;border-bottom:1px solid var(--border2);text-align:left;padding:8px 12px"
                @click="udpMode=m.v">
                <span style="font-weight:700">{{ m.l }}</span>
                <span style="font-size:10px;color:var(--text3);margin-left:6px">{{ m.desc }}</span>
              </button>
            </div>
          </div>
        </div>
        <div class="alert alert-info mt-3 text-xs">
          当前组合模式：<strong>{{ resolvedProxyMode }}</strong>
        </div>
        <div style="display:flex;flex-direction:column;gap:8px;margin-top:12px;margin-bottom:12px">
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px">
            <div class="toggle" :class="{ on: lanProxy }" @click="lanProxy=!lanProxy"></div>
            <span>局域网代理</span>
          </label>
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px">
            <div class="toggle" :class="{ on: ipv6 }" @click="ipv6=!ipv6"></div>
            <span>IPv6 支持</span>
          </label>
        </div>
        <div class="flex gap-2" style="margin-bottom:4px">
          <button class="btn btn-ghost btn-sm" @click="saveProxyMode">保存偏好</button>
          <span v-if="proxyModeMsg" class="text-xs text-green" style="align-self:center">{{ proxyModeMsg }}</span>
        </div>

        <hr class="section-divider" />

        <!-- 局域网 IP 过滤 -->
        <div class="section-sub-title">局域网 IP 过滤</div>
        <div class="field-hint" style="margin-bottom:12px">
          仅在开启局域网代理时生效。
        </div>
        <div class="mode-grid" style="margin-bottom:10px">
          <div v-for="m in ipfModes" :key="m.v"
            class="mode-card" :class="{ on: ipfMode===m.v }"
            @click="ipfMode=m.v">
            <div class="mode-card-icon">{{ m.icon }}</div>
            <div class="mode-card-name">{{ m.name }}</div>
            <div class="mode-card-desc">{{ m.desc }}</div>
          </div>
        </div>
        <div class="field" style="margin-bottom:10px">
          <label class="field-label">IP 列表（空格或换行分隔，支持 CIDR）</label>
          <textarea class="textarea" v-model="ipfIPs" rows="3"
            :disabled="ipfMode==='off'"
            placeholder="192.168.1.0/24 10.0.0.100"></textarea>
        </div>
        <div class="flex gap-2" style="margin-bottom:4px">
          <button class="btn btn-primary" @click="saveIPFilter">保存</button>
          <span v-if="ipfMsg" class="text-xs" :class="ipfMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ ipfMsg }}</span>
        </div>

        <hr class="section-divider" />

        <!-- 绕过中国大陆流量 -->
        <div class="section-sub-title">绕过中国大陆流量</div>
        <div class="field-hint" style="margin-bottom:12px">
          启用后，中国大陆 IP 段的流量将通过 nftables 规则直接放行，不经过 sing-box 核心。
        </div>
        <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:12px">
          <div class="toggle" :class="{ on: bypassCN }" @click="bypassCN=!bypassCN"></div>
          <span>启用绕过中国大陆流量</span>
        </label>
        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" @click="saveProxyMode">保存偏好</button>
          <span v-if="proxyModeMsg" class="text-xs text-green" style="align-self:center">{{ proxyModeMsg }}</span>
        </div>
      </div>

      <!-- ══ sing-box 运行设置 ══════════════════════════════════════════ -->
      <div class="card">
        <div class="card-title">sing-box 运行设置</div>

        <!-- 定时重启核心 -->
        <div class="section-sub-title">定时重启核心</div>
        <div class="field-hint" style="margin-bottom:12px">
          按照 Cron 表达式定期重启 sing-box 核心（仅在核心运行时生效）。
        </div>
        <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:12px">
          <div class="toggle" :class="{ on: schedEnabled }" @click="schedEnabled=!schedEnabled"></div>
          <span>启用定时重启</span>
        </label>
        <div v-if="schedEnabled" class="field" style="margin-bottom:12px">
          <label class="field-label">Cron 表达式（5 字段，分 时 日 月 周）</label>
          <input class="input input-mono" v-model="schedCron"
            placeholder="例如: 15 3 * * *（每天凌晨 3:15）" />
          <div class="field-hint">
            示例：<code>15 3 * * *</code>（每天3:15）&nbsp;
            <code>0 */6 * * *</code>（每6小时）&nbsp;
            <code>30 8 * * 1</code>（每周一8:30）
          </div>
          <div v-if="schedCronError" class="alert alert-error mt-2">{{ schedCronError }}</div>
        </div>
        <div class="flex gap-2" style="margin-bottom:4px">
          <button class="btn btn-primary" @click="saveSched">保存</button>
          <span v-if="schedMsg" class="text-xs" :class="schedMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ schedMsg }}</span>
        </div>

        <hr class="section-divider" />

        <!-- sing-box 运行目录 -->
        <div class="section-sub-title">sing-box 运行目录</div>
        <div class="field-hint" style="margin-bottom:12px">
          设置 <code>sing-box run -D &lt;路径&gt;</code> 的工作目录（留空则使用默认目录）。
        </div>
        <div class="field" style="margin-bottom:12px">
          <label class="field-label">工作目录路径</label>
          <input class="input input-mono" v-model="singboxWorkDir"
            placeholder="留空使用默认路径（data/run）" />
        </div>
        <div class="flex gap-2">
          <button class="btn btn-primary" @click="saveWorkDir">保存</button>
          <span v-if="workDirMsg" class="text-xs" :class="workDirMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ workDirMsg }}</span>
        </div>
      </div>

      <!-- ══ sing-box 核心 ══════════════════════════════════════════ -->
      <div class="card">
        <div class="card-title">sing-box 核心管理</div>
        <div class="info-table" style="margin-bottom:12px">
          <span class="info-k">已安装版本</span>
          <span class="info-v">{{ sbInfo.version || '未安装' }}</span>
          <span class="info-k">架构</span>
          <span class="info-v">{{ sbInfo.arch || '—' }}</span>
          <span class="info-k">系统</span>
          <span class="info-v">{{ sbInfo.osName || '—' }} / {{ sbInfo.libc || '—' }}</span>
        </div>

        <div class="grid-2 gap-3" style="margin-bottom:10px">
          <div class="field">
            <label class="field-label">构建版本</label>
            <div class="seg">
              <button class="seg-btn" :class="{ on: sbFlavor==='official' }"
                @click="sbFlavor='official'">官方版</button>
              <button class="seg-btn" :class="{ on: sbFlavor==='ref1nd' }"
                @click="sbFlavor='ref1nd'">reF1nd 版</button>
            </div>
            <div class="field-hint">
              官方版：SagerNet/sing-box<br>
              reF1nd 版：含额外补丁
            </div>
          </div>
          <div class="field">
            <label class="field-label">版本号</label>
            <div class="seg" style="margin-bottom:6px">
              <button class="seg-btn" :class="{ on: sbVerMode==='latest' }"
                @click="sbVerMode='latest'">最新</button>
              <button class="seg-btn" :class="{ on: sbVerMode==='custom' }"
                @click="sbVerMode='custom'">指定</button>
            </div>
            <input v-if="sbVerMode==='custom'" class="input input-mono"
              v-model="sbVerInput" placeholder="例如 1.13.2" />
          </div>
        </div>

        <div class="flex gap-2">
          <button class="btn btn-ghost" @click="checkVersion" :disabled="sbChecking">
            {{ sbChecking ? '检测中…' : '↺ 检测版本' }}
          </button>
          <button class="btn btn-primary" @click="installSb" :disabled="sbInstalling">
            {{ sbInstalling ? '下载中…' : '↓ 下载/更新核心' }}
          </button>
          <button class="btn btn-primary" @click="updateRules" :disabled="updatingRules">
            {{ updatingRules ? '更新中…' : '↻ 更新规则集' }}
          </button>
        </div>
        <div v-if="sbMsg" class="alert mt-2" :class="sbMsgClass">{{ sbMsg }}</div>
        <div v-if="rulesMsg" class="alert mt-2" :class="rulesMsgClass">{{ rulesMsg }}</div>
        <div v-if="rulesResults.length" style="margin-top:8px;display:flex;flex-direction:column;gap:2px">
          <div v-for="r in rulesResults" :key="r.file"
            class="flex gap-2 items-center text-xs py-1"
            style="border-bottom:1px solid var(--border)">
            <span :class="r.error ? 'text-red' : 'text-green'">{{ r.error ? '✕' : '✓' }}</span>
            <span class="monospace flex-1">{{ r.file }}</span>
            <span class="text-muted">{{ r.error || r.mirror }}</span>
          </div>
        </div>
      </div>

      <!-- ══ 关于 ════════════════════════════════════════════════════════ -->
      <div class="card">
        <div class="card-title">关于</div>
        <div class="info-table">
          <span class="info-k">singa 版本</span><span class="info-v">v2.0</span>
          <span class="info-k">项目主页</span>
          <span class="info-v">
            <a href="https://github.com/singa" target="_blank" style="color:var(--accent)">
              github.com/singa
            </a>
          </span>
        </div>
      </div>

    </div>
  </div>
</template>
