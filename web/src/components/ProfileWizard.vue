<template>
  <div class="wizard-overlay" @click.self="$emit('close')">
    <div class="wizard">

      <!-- Head -->
      <div class="wizard-head">
        <div>
          <div class="wizard-title">{{ profile ? '编辑配置' : '新建配置' }}</div>
          <div class="wizard-sub">{{ stepTitles[step] }}</div>
        </div>
        <button class="btn-icon" style="font-size:16px" @click="$emit('close')">✕</button>
      </div>

      <!-- Step tabs -->
      <div class="wizard-steps">
        <button v-for="(t, i) in stepTitles" :key="i"
          class="wizard-step"
          :class="{ active: step === i, done: i < step }"
          @click="step = i">
          {{ i + 1 }}. {{ t }}
        </button>
      </div>

      <!-- Body -->
      <div class="wizard-body">

        <!-- ═══ Step 1: 名称 ═══════════════════════════════════════════ -->
        <template v-if="step === 0">
          <div class="wfield">
            <div class="wlabel">配置名称 <span style="color:var(--red)">*</span></div>
            <input class="input" v-model="form.name" placeholder="例如：我的配置" />
          </div>
        </template>

        <!-- ═══ Step 2: 出站 ═══════════════════════════════════════════ -->
        <template v-if="step === 1">
          <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap">
            <button class="btn btn-primary btn-sm" @click="openOutboundEdit(null)">+ 添加出站</button>
          </div>

          <div v-if="!form.outbounds.length" class="empty">点击上方「添加出站」创建出站节点</div>

          <!-- draggable outbound list -->
          <div ref="obDragEl">
            <div v-for="(ob, idx) in form.outbounds" :key="ob.id"
              class="ob-card" style="cursor:default">
              <div class="ob-head">
                <span class="drag-handle" style="cursor:move;color:var(--text3)">⠿</span>
                <span v-if="ob.icon" style="margin:0 4px">
                  <img :src="ob.icon" style="width:14px;height:14px;vertical-align:middle" />
                </span>
                <span class="ob-type" :class="ob.type==='selector'?'ob-type-sel':'ob-type-url'">{{ ob.type }}</span>
                <span class="ob-tag">{{ ob.tag || '(未命名)' }}</span>
                <span class="ob-hint">
                  (引用出站:{{ countOutbounds(ob) }} / 引用订阅:{{ countSubs(ob) }})
                </span>
                <div style="margin-left:auto;display:flex;gap:4px">
                  <button class="btn-icon" @click="openSortModal(idx)" title="排序引用">⇅</button>
                  <button class="btn-icon" @click="openOutboundEdit(idx)" title="编辑">✎</button>
                  <button class="btn-icon danger" @click="deleteOutbound(idx)" title="删除">✕</button>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- ═══ Step 3: 路由 ═══════════════════════════════════════════ -->
        <template v-if="step === 2">
          <div class="tabs" style="margin-bottom:14px">
            <button class="tab-btn" :class="{ on: routeTab==='general' }" @click="routeTab='general'">通用</button>
            <button class="tab-btn" :class="{ on: routeTab==='rulesets' }" @click="routeTab='rulesets'">规则集</button>
            <button class="tab-btn" :class="{ on: routeTab==='rules' }" @click="routeTab='rules'">规则</button>
          </div>

          <!-- 通用 -->
          <div v-if="routeTab==='general'">
            <div class="wfield winline">
              <div class="toggle" :class="{ on: form.route.auto_detect_interface }" @click="form.route.auto_detect_interface=!form.route.auto_detect_interface"></div>
              <span class="wlabel" style="margin:0">自动检测出站接口</span>
            </div>
            <div class="wfield" v-if="!form.route.auto_detect_interface">
              <div class="wlabel">默认出站接口</div>
              <input class="input" v-model="form.route.default_interface" placeholder="eth0" />
            </div>
            <div class="wfield">
              <div class="wlabel">解析节点域名的 DNS 服务器</div>
              <select class="select" v-model="form.route.default_domain_resolver">
                <option value="">不设置</option>
                <option v-for="s in form.dns.servers" :key="s.id" :value="s.id">{{ s.tag }} ({{ s.type }})</option>
              </select>
            </div>
            <div class="wfield">
              <div class="wlabel">默认出站 final</div>
              <select class="select" v-model="form.route.final">
                <option v-for="ob in allOutboundOptions" :key="ob.v" :value="ob.v">{{ ob.l }}</option>
              </select>
            </div>
          </div>

          <!-- 规则集 -->
          <div v-if="routeTab==='rulesets'">
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <button class="btn btn-primary btn-sm" @click="openRulesetEdit(null)">+ 添加规则集</button>
            </div>
            <div v-if="!form.route.rule_set.length" class="empty">暂无规则集</div>
            <div ref="rsDragEl">
              <div v-for="(rs, idx) in form.route.rule_set" :key="rs.id" class="rs-wcard">
                <div class="rs-whead">
                  <span class="drag-handle" style="cursor:move;color:var(--text3)">⠿</span>
                  <span style="font-size:10px;padding:1px 6px;border-radius:3px;font-family:var(--mono);font-weight:700;margin:0 4px"
                    :style="rs.type==='remote'?'background:#dbeafe;color:#1e40af':'background:#d1fae5;color:#065f46'">
                    {{ rs.type }}
                  </span>
                  <span class="rs-wtag">{{ rs.tag || '(未命名)' }}</span>
                  <div style="margin-left:auto;display:flex;gap:4px">
                    <button class="btn-icon" @click="openRulesetEdit(idx)">✎</button>
                    <button class="btn-icon danger" @click="form.route.rule_set.splice(idx,1)">✕</button>
                  </div>
                </div>
                <div v-if="rs.type==='remote'" class="whint" style="margin-top:4px;font-family:var(--mono);font-size:10px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                  {{ rs.url }}
                </div>
                <div v-else-if="rs.type==='local'" class="whint" style="margin-top:4px">{{ rs.path }}</div>
              </div>
            </div>
          </div>

          <!-- 规则 -->
          <div v-if="routeTab==='rules'">
            <div style="display:flex;gap:8px;margin-bottom:10px;flex-wrap:wrap">
              <button class="btn btn-primary btn-sm" @click="openRouteRuleEdit(null)">+ 添加规则</button>
            </div>
            <div class="whint" style="margin-bottom:8px">按顺序匹配，拖动任意行调整顺序；将插入点横条拖到目标位置，再点「+ 添加规则」即可插入到该位置</div>

            <!-- Outbound legend -->
            <div style="display:flex;flex-wrap:wrap;gap:4px;margin-bottom:8px">
              <span v-for="ob in allOutboundOptions" :key="ob.v"
                style="font-size:10px;padding:1px 7px;background:var(--surface2);border:1px solid var(--border);border-radius:3px;font-family:var(--mono)">
                <code style="color:var(--accent)">{{ ob.v }}</code> → {{ ob.l }}
              </span>
            </div>

            <div v-if="!form.route.rules.length" class="empty">暂无路由规则</div>
            <div data-drag-list="route-rules" style="border:1px solid var(--border);border-radius:var(--radius);overflow:hidden">
              <div v-for="(rule, idx) in form.route.rules" :key="rule.id"
                data-drag-row
                class="rule-row" :class="{ off: !rule.enable, 'drag-over': isDragTarget('route-rules', idx), 'drag-src': isDragging('route-rules', idx) }"
                @mousedown="onRowMousedown($event, 'route-rules', idx, form.route.rules)"
                @click="rule.type!=='InsertionPoint' && (rule.enable=!rule.enable)">

                <!-- Insertion point -->
                <div v-if="rule.type==='InsertionPoint'"
                  style="width:100%;text-align:center;padding:4px 0;background:var(--accent-bg);color:var(--accent);font-size:11px;font-weight:700;cursor:grab"
                  @click.stop>
                  ↕ 拖动此横条到目标位置，再点「+ 添加规则」插入
                </div>

                <template v-else>
                  <span class="rule-num">{{ idx + 1 }}</span>
                  <span class="rule-dot" :class="{ on: rule.enable }">●</span>
                  <span class="rule-type">{{ rule.type }}</span>
                  <span class="rule-payload">{{ renderRouteRulePayload(rule) }}</span>
                  <span class="rule-act" :class="'ra-' + rule.action">{{ rule.action }}</span>
                  <span v-if="rule.outbound" class="rule-out">
                    → {{ outboundLabel(rule.outbound) }}
                  </span>
                  <div style="margin-left:auto;display:flex;gap:3px;flex-shrink:0" @click.stop>
                    <button class="btn-icon" @click="openRouteRuleEdit(idx)">✎</button>
                    <button class="btn-icon danger" @click="form.route.rules.splice(idx,1)">✕</button>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </template>

        <!-- ═══ Step 4: DNS ════════════════════════════════════════════ -->
        <template v-if="step === 3">
          <div class="tabs" style="margin-bottom:14px">
            <button class="tab-btn" :class="{ on: dnsTab==='general' }" @click="dnsTab='general'">通用</button>
            <button class="tab-btn" :class="{ on: dnsTab==='servers' }" @click="dnsTab='servers'">服务器</button>
            <button class="tab-btn" :class="{ on: dnsTab==='rules' }" @click="dnsTab='rules'">规则</button>
          </div>

          <!-- 通用 -->
          <div v-if="dnsTab==='general'">
            <div class="wfield winline">
              <div class="toggle" :class="{ on: form.dns.disable_cache }" @click="form.dns.disable_cache=!form.dns.disable_cache"></div>
              <span class="wlabel" style="margin:0">禁用 DNS 缓存</span>
            </div>
            <div class="wfield winline" style="margin-top:8px">
              <div class="toggle" :class="{ on: form.dns.disable_expire }" @click="form.dns.disable_expire=!form.dns.disable_expire"></div>
              <span class="wlabel" style="margin:0">禁用 DNS 缓存过期</span>
            </div>
            <div class="wfield winline" style="margin-top:8px">
              <div class="toggle" :class="{ on: form.dns.independent_cache }" @click="form.dns.independent_cache=!form.dns.independent_cache"></div>
              <div>
                <div class="wlabel" style="margin:0">独立缓存</div>
                <div class="whint">每个 DNS 服务器维护独立缓存</div>
              </div>
            </div>
            <div class="wfield" style="margin-top:10px">
              <div class="wlabel">回退 DNS（final）</div>
              <select class="select" v-model="form.dns.final">
                <option v-for="s in form.dns.servers" :key="s.id" :value="s.id">{{ s.tag }} ({{ s.type }})</option>
              </select>
            </div>
            <div class="wfield">
              <div class="wlabel">解析策略</div>
              <select class="select" v-model="form.dns.strategy">
                <option value="default">默认</option>
                <option value="prefer_ipv4">优先 IPv4</option>
                <option value="prefer_ipv6">优先 IPv6</option>
                <option value="ipv4_only">仅 IPv4</option>
                <option value="ipv6_only">仅 IPv6</option>
              </select>
            </div>
            <div class="wfield">
              <div class="wlabel">客户端子网 EDNS（留空不设）</div>
              <input class="input input-mono" v-model="form.dns.client_subnet" placeholder="1.2.3.4/24" />
            </div>
          </div>

          <!-- 服务器 -->
          <div v-if="dnsTab==='servers'">
            <div style="display:flex;gap:8px;margin-bottom:12px">
              <button class="btn btn-primary btn-sm" @click="openDnsServerEdit(null)">+ 添加服务器</button>
            </div>
            <div v-if="!form.dns.servers.length" class="empty">暂无 DNS 服务器</div>
            <div ref="dnsSrvDragEl">
              <div v-for="(srv, idx) in form.dns.servers" :key="srv.id" class="dns-card">
                <div class="dns-head">
                  <span class="drag-handle" style="cursor:move;color:var(--text3)">⠿</span>
                  <span class="dns-tag">{{ srv.tag }}</span>
                  <span style="font-size:9px;padding:1px 5px;border-radius:3px;background:#ede9fe;color:#5b21b6;font-family:var(--mono);font-weight:700;margin-left:4px">{{ srv.type }}</span>
                  <div style="font-size:11px;color:var(--text3);font-family:var(--mono);flex:1;margin-left:8px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                    {{ dnsServerSummary(srv) }}
                  </div>
                  <div style="display:flex;gap:3px">
                    <button class="btn-icon" @click="openDnsServerEdit(idx)">✎</button>
                    <button class="btn-icon danger" @click="form.dns.servers.splice(idx,1)">✕</button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- DNS 规则 -->
          <div v-if="dnsTab==='rules'">
            <div style="display:flex;gap:8px;margin-bottom:10px">
              <button class="btn btn-primary btn-sm" @click="openDnsRuleEdit(null)">+ 添加规则</button>
            </div>
            <div class="whint" style="margin-bottom:8px">按顺序匹配，拖动任意行调整顺序；将插入点横条拖到目标位置，再点「+ 添加规则」即可插入到该位置</div>
            <div v-if="!form.dns.rules.length" class="empty">暂无 DNS 规则</div>
            <div data-drag-list="dns-rules" style="border:1px solid var(--border);border-radius:var(--radius);overflow:hidden">
              <div v-for="(rule, idx) in form.dns.rules" :key="rule.id"
                data-drag-row
                class="rule-row" :class="{ off: !rule.enable, 'drag-over': isDragTarget('dns-rules', idx), 'drag-src': isDragging('dns-rules', idx) }"
                @mousedown="onRowMousedown($event, 'dns-rules', idx, form.dns.rules)"
                @click="rule.type!=='InsertionPoint' && (rule.enable=!rule.enable)">

                <div v-if="rule.type==='InsertionPoint'"
                  style="width:100%;text-align:center;padding:4px 0;background:var(--accent-bg);color:var(--accent);font-size:11px;font-weight:700;cursor:grab"
                  @click.stop>
                  ↕ 拖动此横条到目标位置，再点「+ 添加规则」插入
                </div>

                <template v-else>
                  <span class="rule-num">{{ idx + 1 }}</span>
                  <span class="rule-dot" :class="{ on: rule.enable }">●</span>
                  <span class="rule-type">{{ rule.type }}</span>
                  <span class="rule-payload">{{ renderDnsRulePayload(rule) }}</span>
                  <span class="rule-act ra-route">route</span>
                  <span v-if="rule.server" class="rule-out">→ {{ dnsServerTagById(rule.server) }}</span>
                  <div style="margin-left:auto;display:flex;gap:3px;flex-shrink:0" @click.stop>
                    <button class="btn-icon" @click="openDnsRuleEdit(idx)">✎</button>
                    <button class="btn-icon danger" @click="form.dns.rules.splice(idx,1)">✕</button>
                  </div>
                </template>
              </div>
            </div>
            <div class="whint" style="margin-top:8px">
              💡 启用 FakeIP 规则需同时在「服务器」中配置 fakeip 类型服务器，并在通用设置启用存储 FakeIP 映射。
            </div>
          </div>
        </template>

      </div><!-- /wizard-body -->

      <!-- Footer -->
      <div class="wizard-foot">
        <span class="wizard-progress">步骤 {{ step + 1 }} / {{ stepTitles.length }}</span>
        <div class="wizard-foot-btns">
          <button class="btn btn-ghost" :disabled="step===0" @click="step--">◀ 上一步</button>
          <button v-if="step < stepTitles.length - 1" class="btn btn-primary" @click="nextStep">下一步 ▶</button>
          <button v-else class="btn btn-success" :disabled="saving" @click="save">
            {{ saving ? '保存中…' : '💾 保存' }}
          </button>
        </div>
      </div>

    </div>
  </div>

  <!-- ══ Outbound Edit Modal ════════════════════════════════════════════ -->
  <div v-if="showObModal" class="mask" @click.self="showObModal=false" style="z-index:500">
    <div class="modal" style="max-width:640px;max-height:85vh;overflow-y:auto">
      <div class="modal-head">
        <span>出站</span>
        <button class="btn-icon" @click="showObModal=false">✕</button>
      </div>
      <div class="modal-body">
        <div class="wfield">
          <div class="wlabel">名称</div>
          <input class="input" v-model="obFields.tag" autofocus placeholder="节点选择" />
        </div>
        <div class="wfield">
          <div class="wlabel">类型</div>
          <div class="seg">
            <button v-for="t in OUTBOUND_TYPES" :key="t.v" class="seg-btn" :class="{ on: obFields.type===t.v }" @click="obFields.type=t.v">{{ t.l }}</button>
          </div>
        </div>

        <template v-if="obFields.type==='selector' || obFields.type==='urltest'">
          <div class="wfield winline">
            <div class="toggle" :class="{ on: obFields.hidden }" @click="obFields.hidden=!obFields.hidden"></div>
            <span class="wlabel" style="margin:0">隐藏此出站</span>
          </div>
          <div class="wfield">
            <div class="wlabel">包含（关键词过滤，| 分隔）</div>
            <input class="input input-mono" v-model="obFields.include" placeholder="keywords1|keywords2" />
          </div>
          <div class="wfield">
            <div class="wlabel">排除（关键词过滤，| 分隔）</div>
            <input class="input input-mono" v-model="obFields.exclude" placeholder="美国" />
          </div>
          <div class="wfield">
            <div class="wlabel">图标 URL</div>
            <div style="display:flex;gap:8px;align-items:center">
              <img v-if="obFields.icon" :src="obFields.icon" style="width:20px;height:20px" />
              <input class="input input-mono" v-model="obFields.icon" placeholder="https://" style="flex:1" />
            </div>
          </div>
        </template>

        <template v-if="obFields.type==='urltest'">
          <div class="wfield">
            <div class="wlabel">测速 URL</div>
            <input class="input input-mono" v-model="obFields.url" />
          </div>
          <div class="wfield">
            <div class="wlabel">测速间隔</div>
            <input class="input" v-model="obFields.interval" placeholder="3m" />
          </div>
          <div class="wfield">
            <div class="wlabel">容差（ms）</div>
            <input class="input" type="number" v-model.number="obFields.tolerance" />
          </div>
        </template>

        <template v-if="obFields.type==='direct' || obFields.type==='block'">
          <div class="alert alert-info text-xs">direct / block 类型无额外配置</div>
        </template>

        <!-- Refs panel: only for selector/urltest -->
        <template v-if="obFields.type==='selector' || obFields.type==='urltest'">
          <div class="wsep">引用出站 &amp; 引用订阅</div>

          <!-- Built-in group -->
          <div class="ob-ref-group">
            <div class="ob-ref-group-head" @click="toggleGroup('builtin')">
              <span style="font-weight:700;flex:1">内置</span>
              <span class="text-xs text-muted">{{ BUILTIN_OUTBOUNDS.length + form.outbounds.length }}</span>
              <span style="margin-left:8px">{{ expandedGroups.has('builtin') ? '∨' : '›' }}</span>
            </div>
            <div v-show="expandedGroups.has('builtin')" class="ob-ref-grid">
              <button v-for="ob in allProxies" :key="ob.id"
                class="ob-ref-item" :class="{ on: isInRefs(ob.id) }"
                @click="toggleRef(ob.id, ob.tag)">
                <span style="font-size:11px;font-weight:700">{{ ob.tag }}</span>
                <span class="text-xs text-muted">{{ ob.type }}</span>
              </button>
            </div>
          </div>

          <!-- Nodes group -->
          <div class="ob-ref-group" style="margin-top:8px">
            <div class="ob-ref-group-head" @click="toggleGroup('nodes')">
              <span style="font-weight:700;flex:1">导入节点</span>
              <span class="text-xs text-muted">{{ nodesStore.nodes.length }}</span>
              <span style="margin-left:8px">{{ expandedGroups.has('nodes') ? '∨' : '›' }}</span>
            </div>
            <div v-show="expandedGroups.has('nodes')" class="ob-ref-grid">
              <button v-if="!nodesStore.nodes.length"
                class="ob-ref-item" style="opacity:.5;cursor:default">无节点</button>
              <button v-for="n in nodesStore.nodes" :key="n.id"
                class="ob-ref-item" :class="{ on: isInRefs(n.id) }"
                @click="toggleRef(n.id, n.name || n.server)">
                <span style="font-size:11px;font-weight:700">{{ n.name || n.server }}</span>
                <span class="text-xs text-muted">{{ n.type }}</span>
              </button>
            </div>
          </div>

          <!-- Subscription group -->
          <div class="ob-ref-group" style="margin-top:8px">
            <div class="ob-ref-group-head" @click="toggleGroup('subscription')">
              <span style="font-weight:700;flex:1">订阅</span>
              <span class="text-xs text-muted">{{ subsStore.subs.length }}</span>
              <span style="margin-left:8px">{{ expandedGroups.has('subscription') ? '∨' : '›' }}</span>
            </div>
            <div v-show="expandedGroups.has('subscription')" class="ob-ref-grid">
              <button v-if="!subsStore.subs.length"
                class="ob-ref-item" style="opacity:.5;cursor:default">无订阅</button>
              <button v-for="sub in subsStore.subs" :key="sub.id"
                class="ob-ref-item" :class="{ on: isInRefs(sub.id) }"
                @click="toggleRef(sub.id, sub.name, 'Subscription')">
                <span style="font-size:11px;font-weight:700">{{ sub.name }}</span>
                <span class="text-xs text-muted">Subscribe</span>
              </button>
            </div>
          </div>
        </template>
      </div>
      <div class="modal-foot">
        <button class="btn btn-ghost" @click="showObModal=false">取消</button>
        <button class="btn btn-primary" @click="saveOutbound">保存</button>
      </div>
    </div>
  </div>

  <!-- ══ Outbound Sort Modal ═══════════════════════════════════════════ -->
  <div v-if="showSortModal" class="mask" @click.self="showSortModal=false" style="z-index:500">
    <div class="modal" style="max-width:500px">
      <div class="modal-head">
        <span>排序引用 — {{ obFields.tag }}</span>
        <button class="btn-icon" @click="showSortModal=false">✕</button>
      </div>
      <div class="modal-body">
        <div class="whint" style="margin-bottom:8px">拖动调整引用顺序</div>
        <div ref="sortDragEl" style="display:flex;flex-wrap:wrap;gap:6px;min-height:40px;border:1px dashed var(--border2);border-radius:var(--radius);padding:8px">
          <div v-for="ref in obFields.outbounds" :key="ref.id"
            style="display:inline-flex;align-items:center;gap:4px;padding:4px 10px;background:var(--surface2);border:1px solid var(--border2);border-radius:var(--radius);cursor:move;font-size:12px">
            <span style="color:var(--text3)">⠿</span>
            {{ ref.tag }}
          </div>
        </div>
      </div>
      <div class="modal-foot">
        <button class="btn btn-ghost" @click="showSortModal=false">取消</button>
        <button class="btn btn-primary" @click="saveSortModal">确定</button>
      </div>
    </div>
  </div>

  <!-- ══ Ruleset Edit Modal ════════════════════════════════════════════ -->
  <div v-if="showRsModal" class="mask" @click.self="showRsModal=false" style="z-index:500">
    <div class="modal" style="max-width:640px;max-height:85vh;overflow-y:auto">
      <div class="modal-head">
        <span>规则集</span>
        <button class="btn-icon" @click="showRsModal=false">✕</button>
      </div>
      <div class="modal-body">
        <div class="wfield">
          <div class="wlabel">名称（Tag）</div>
          <input class="input" v-model="rsFields.tag" placeholder="GeoSite-CN" />
        </div>
        <div class="wfield">
          <div class="wlabel">类型</div>
          <div class="seg">
            <button v-for="t in RULESET_TYPES" :key="t.v" class="seg-btn" :class="{ on: rsFields.type===t.v }" @click="rsFields.type=t.v">{{ t.l }}</button>
          </div>
        </div>

        <template v-if="rsFields.type==='local'">
          <div class="wsep">从本地规则集选择</div>
          <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:8px">
            <div v-if="!rsStore.local.length" class="whint" style="grid-column:1/-1">暂无本地规则集，请先在「规则集」页面下载</div>
            <button v-for="lrs in rsStore.local" :key="lrs.file"
              class="ob-ref-item" :class="{ on: rsFields.path===lrs.file }"
              @click="rsFields.path=lrs.file; rsFields.tag=lrs.file.replace('.srs',''); rsFields.format='binary'">
              <span style="font-size:11px;font-weight:700;word-break:break-all">{{ lrs.file }}</span>
              <span class="text-xs text-muted">{{ fmtSize(lrs.size) }}</span>
            </button>
          </div>
          <div class="wfield" style="margin-top:10px">
            <div class="wlabel">或手动填写路径</div>
            <input class="input input-mono" v-model="rsFields.path" placeholder="/etc/sing-box/rules/xxx.srs" />
          </div>
          <div class="wfield">
            <div class="wlabel">格式</div>
            <div class="seg">
              <button class="seg-btn" :class="{ on: rsFields.format==='binary' }" @click="rsFields.format='binary'">binary</button>
              <button class="seg-btn" :class="{ on: rsFields.format==='source' }" @click="rsFields.format='source'">source</button>
            </div>
          </div>
        </template>

        <template v-if="rsFields.type==='remote'">
          <div class="wfield">
            <div class="wlabel">格式</div>
            <div class="seg">
              <button class="seg-btn" :class="{ on: rsFields.format==='binary' }" @click="rsFields.format='binary'">binary</button>
              <button class="seg-btn" :class="{ on: rsFields.format==='source' }" @click="rsFields.format='source'">source</button>
            </div>
          </div>
          <div class="wfield">
            <div class="wlabel">URL</div>
            <input class="input input-mono" v-model="rsFields.url" placeholder="https://..." />
          </div>
          <div class="wfield">
            <div class="wlabel">下载节点</div>
            <input class="input" v-model="rsFields.download_detour" placeholder="direct" />
          </div>
          <div class="wfield">
            <div class="wlabel">更新间隔（留空不自动更新）</div>
            <input class="input" v-model="rsFields.update_interval" placeholder="1d" />
          </div>
        </template>

        <template v-if="rsFields.type==='inline'">
          <div class="wfield">
            <div class="wlabel">规则内容（JSON）</div>
            <textarea class="textarea" v-model="rsFields.rules" rows="6" placeholder='{"domain":["example.com"]}'></textarea>
          </div>
        </template>
      </div>
      <div class="modal-foot">
        <button class="btn btn-ghost" @click="showRsModal=false">取消</button>
        <button class="btn btn-primary" @click="saveRuleset">保存</button>
      </div>
    </div>
  </div>

  <!-- ══ Route Rule Edit Modal ═════════════════════════════════════════ -->
  <div v-if="showRouteRuleModal" class="mask" @click.self="showRouteRuleModal=false" style="z-index:500">
    <div class="modal" style="max-width:640px;max-height:88vh;overflow-y:auto">
      <div class="modal-head">
        <span>规则</span>
        <button class="btn-icon" @click="showRouteRuleModal=false">✕</button>
      </div>
      <div class="modal-body">
        <div class="wfield">
          <div class="wlabel">规则类型</div>
          <select class="select" v-model="rrFields.type">
            <option v-for="t in RULE_TYPES" :key="t.v" :value="t.v">{{ t.l }}</option>
          </select>
        </div>
        <div class="wfield">
          <div class="wlabel">规则动作</div>
          <div class="seg" style="flex-wrap:wrap">
            <button v-for="a in ROUTE_RULE_ACTIONS" :key="a.v" class="seg-btn" :class="{ on: rrFields.action===a.v }" @click="rrFields.action=a.v">{{ a.l }}</button>
          </div>
        </div>
        <!-- payload -->
        <div v-if="rrFields.type === 'action'" class="alert alert-info text-xs">
          action 类型无条件匹配，直接执行动作（如全局 sniff）
        </div>
        <div v-else-if="rrFields.type === 'network+port'" class="wfield">
          <div class="wlabel">格式：network:port，如 udp:443</div>
          <input class="input input-mono" v-model="rrFields.payload" placeholder="udp:443" />
        </div>
        <div v-else-if="rrFields.type !== 'rule_set'" class="wfield">
          <div class="wlabel">规则内容（payload）</div>
          <div v-if="rrFields.type==='clash_mode'" class="seg">
            <button class="seg-btn" :class="{ on: rrFields.payload==='direct' }" @click="rrFields.payload='direct'">direct</button>
            <button class="seg-btn" :class="{ on: rrFields.payload==='global' }" @click="rrFields.payload='global'">global</button>
          </div>
          <input v-else class="input input-mono" v-model="rrFields.payload" />
        </div>
        <div class="wfield winline">
          <div class="toggle" :class="{ on: rrFields.invert }" @click="rrFields.invert=!rrFields.invert"></div>
          <span class="wlabel" style="margin:0">反向匹配（invert）</span>
        </div>

        <!-- Action params -->
        <div v-if="rrFields.action==='route'" class="wfield">
          <div class="wlabel">出站标签</div>
          <select class="select" v-model="rrFields.outbound">
            <option value="">— 选择出站 —</option>
            <option v-for="ob in allOutboundOptions" :key="ob.v" :value="ob.v">{{ ob.l }}</option>
          </select>
        </div>
        <div v-if="rrFields.action==='sniff'" class="wfield">
          <div class="wlabel">嗅探协议（留空=全部）</div>
          <div style="display:flex;flex-wrap:wrap;gap:6px">
            <label v-for="s in SNIFFERS" :key="s" class="winline" style="cursor:pointer;background:var(--surface2);padding:3px 8px;border-radius:4px;border:1px solid var(--border)">
              <input type="checkbox" :value="s" v-model="rrFields.sniffer" style="accent-color:var(--accent)" />
              <span style="font-size:12px;font-family:var(--mono)">{{ s }}</span>
            </label>
          </div>
        </div>

        <!-- ruleset picker -->
        <template v-if="rrFields.type==='rule_set'">
          <div class="wsep">选择规则集（可多选）</div>
          <div v-if="!form.route.rule_set.length" class="whint">请先在「规则集」Tab 添加规则集</div>
          <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:6px">
            <button v-for="rs in form.route.rule_set" :key="rs.id"
              class="ob-ref-item" :class="{ on: rrRulesetSelected(rs.id) }"
              @click="toggleRulesetInRule(rs.id, rrFields)">
              <span style="font-size:11px;font-weight:700">{{ rs.tag }}</span>
              <span class="text-xs text-muted">{{ rs.type }} {{ rs.format }}</span>
            </button>
          </div>
        </template>
      </div>
      <div class="modal-foot">
        <button class="btn btn-ghost" @click="showRouteRuleModal=false">取消</button>
        <button class="btn btn-primary" @click="saveRouteRule">保存</button>
      </div>
    </div>
  </div>

  <!-- ══ DNS Server Edit Modal ═════════════════════════════════════════ -->
  <div v-if="showDnsSrvModal" class="mask" @click.self="showDnsSrvModal=false" style="z-index:500">
    <div class="modal" style="max-width:500px">
      <div class="modal-head">
        <span>服务器</span>
        <button class="btn-icon" @click="showDnsSrvModal=false">✕</button>
      </div>
      <div class="modal-body">
        <div class="wfield">
          <div class="wlabel">类型</div>
          <div class="seg" style="flex-wrap:wrap">
            <button v-for="t in DNS_SERVER_TYPES" :key="t" class="seg-btn" :class="{ on: dnsSrvFields.type===t }" @click="dnsSrvFields.type=t" style="font-family:var(--mono);font-size:11px">{{ t }}</button>
          </div>
        </div>
        <div class="wfield">
          <div class="wlabel">名称（Tag）</div>
          <input class="input" v-model="dnsSrvFields.tag" />
        </div>
        <template v-if="['tcp','udp','tls','https','quic','h3'].includes(dnsSrvFields.type)">
          <div class="wfield">
            <div class="wlabel">服务器地址</div>
            <div style="display:flex;gap:8px">
              <input class="input input-mono" v-model="dnsSrvFields.server" placeholder="8.8.8.8" style="flex:3" />
              <input class="input input-mono" v-model="dnsSrvFields.server_port" placeholder="53" style="flex:1" />
            </div>
          </div>
          <div v-if="['https','h3'].includes(dnsSrvFields.type)" class="wfield">
            <div class="wlabel">路径</div>
            <input class="input input-mono" v-model="dnsSrvFields.path" placeholder="/dns-query" />
          </div>
          <div class="wfield">
            <div class="wlabel">解析服务器自身域名的 DNS（domain_resolver）</div>
            <select class="select" v-model="dnsSrvFields.domain_resolver">
              <option value="">不设置</option>
              <option v-for="s in form.dns.servers.filter(x=>x.id!==dnsSrvFields.id)" :key="s.id" :value="s.id">{{ s.tag }} ({{ s.type }})</option>
            </select>
          </div>
          <div class="wfield">
            <div class="wlabel">出站节点（detour）</div>
            <select class="select" v-model="dnsSrvFields.detour">
              <option value="">不设置</option>
              <option v-for="ob in allProxies" :key="ob.id" :value="ob.id">{{ ob.tag }}</option>
            </select>
          </div>
        </template>
        <template v-if="dnsSrvFields.type==='fakeip'">
          <div class="wfield">
            <div class="wlabel">FakeIP IPv4 段</div>
            <input class="input input-mono" v-model="dnsSrvFields.inet4_range" placeholder="198.18.0.0/15" />
          </div>
          <div class="wfield">
            <div class="wlabel">FakeIP IPv6 段</div>
            <input class="input input-mono" v-model="dnsSrvFields.inet6_range" placeholder="fc00::/18" />
          </div>
        </template>
        <div v-if="dnsSrvFields.type==='local'" class="alert alert-info text-xs">使用系统本地 DNS，无需额外配置。</div>
      </div>
      <div class="modal-foot">
        <button class="btn btn-ghost" @click="showDnsSrvModal=false">取消</button>
        <button class="btn btn-primary" @click="saveDnsServer">保存</button>
      </div>
    </div>
  </div>

  <!-- ══ DNS Rule Edit Modal ════════════════════════════════════════════ -->
  <div v-if="showDnsRuleModal" class="mask" @click.self="showDnsRuleModal=false" style="z-index:500">
    <div class="modal" style="max-width:640px;max-height:85vh;overflow-y:auto">
      <div class="modal-head">
        <span>DNS 规则</span>
        <button class="btn-icon" @click="showDnsRuleModal=false">✕</button>
      </div>
      <div class="modal-body">
        <div class="wfield">
          <div class="wlabel">规则类型</div>
          <select class="select" v-model="drFields.type">
            <option v-for="t in RULE_TYPES" :key="t.v" :value="t.v">{{ t.l }}</option>
          </select>
        </div>
        <div class="wfield">
          <div class="wlabel">规则动作</div>
          <div class="seg">
            <button v-for="a in DNS_RULE_ACTIONS" :key="a.v" class="seg-btn" :class="{ on: drFields.action===a.v }" @click="drFields.action=a.v">{{ a.l }}</button>
          </div>
        </div>
        <div v-if="drFields.type!=='rule_set'" class="wfield">
          <div class="wlabel">规则内容</div>
          <div v-if="drFields.type==='clash_mode'" class="seg">
            <button class="seg-btn" :class="{ on: drFields.payload==='direct' }" @click="drFields.payload='direct'">direct</button>
            <button class="seg-btn" :class="{ on: drFields.payload==='global' }" @click="drFields.payload='global'">global</button>
          </div>
          <input v-else class="input input-mono" v-model="drFields.payload" />
        </div>
        <div class="wfield winline">
          <div class="toggle" :class="{ on: drFields.invert }" @click="drFields.invert=!drFields.invert"></div>
          <span class="wlabel" style="margin:0">反向匹配</span>
        </div>
        <div v-if="drFields.action==='route'" class="wfield">
          <div class="wlabel">目标 DNS 服务器</div>
          <select class="select" v-model="drFields.server">
            <option value="">— 选择服务器 —</option>
            <option v-for="s in form.dns.servers" :key="s.id" :value="s.id">{{ s.tag }} ({{ s.type }})</option>
          </select>
        </div>
        <template v-if="drFields.type==='rule_set'">
          <div class="wsep">选择规则集（可多选）</div>
          <div v-if="!form.route.rule_set.length" class="whint">请先在路由设置→规则集 添加规则集</div>
          <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:6px">
            <button v-for="rs in form.route.rule_set" :key="rs.id"
              class="ob-ref-item" :class="{ on: drRulesetSelected(rs.id) }"
              @click="toggleRulesetInRule(rs.id, drFields)">
              <span style="font-size:11px;font-weight:700">{{ rs.tag }}</span>
              <span class="text-xs text-muted">{{ rs.type }}</span>
            </button>
          </div>
        </template>
      </div>
      <div class="modal-foot">
        <button class="btn btn-ghost" @click="showDnsRuleModal=false">取消</button>
        <button class="btn btn-primary" @click="saveDnsRule">保存</button>
      </div>
    </div>
  </div>

</template>

<script setup>
import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { useRulesetsStore, useProfilesStore, useNodesStore, useSubsStore } from '../stores.js'

const props = defineProps({
  profile: { type: Object, default: null },
  subs:    { type: Array,  default: () => [] },
})
const emit  = defineEmits(['close', 'saved'])
const saveErrors = ref([])
const savingValidationMsg = ref('')

const profilesStore = useProfilesStore()
const rsStore    = useRulesetsStore()
const nodesStore = useNodesStore()
const subsStore  = useSubsStore()
const saving     = ref(false)
const step       = ref(0)
const routeTab   = ref('general')
const dnsTab     = ref('general')

// ── Constants ──────────────────────────────────────────────────────────────
const stepTitles   = ['名称/订阅','出站设置','路由设置','DNS 设置']
const LOG_LEVELS   = [
  {v:'trace',l:'跟踪'},{v:'debug',l:'调试'},{v:'info',l:'信息'},
  {v:'warn',l:'警告'},{v:'error',l:'错误'},{v:'fatal',l:'致命'},{v:'panic',l:'恐慌'},
]
const CLASH_MODES  = [{v:'rule',l:'规则'},{v:'global',l:'全局'},{v:'direct',l:'直连'}]
const TUN_STACKS   = ['system','gvisor','mixed']
const INBOUND_TYPES = [
  {v:'mixed',l:'Mixed (HTTP+SOCKS)'},{v:'http',l:'HTTP'},
  {v:'socks',l:'SOCKS'},{v:'redirect',l:'Redirect'},
  {v:'tproxy',l:'TProxy'},{v:'tun',l:'TUN'},
]
const OUTBOUND_TYPES = [
  {v:'direct',l:'直连'},{v:'block',l:'阻断'},
  {v:'selector',l:'手动选择'},{v:'urltest',l:'自动选择'},
]
const RULESET_TYPES = [{v:'local',l:'本地'},{v:'remote',l:'远程'},{v:'inline',l:'内联'}]
const RULE_TYPES = [
  {v:'action',l:'action(无条件)'},{v:'inbound',l:'inbound'},{v:'network',l:'network'},{v:'protocol',l:'protocol'},
  {v:'network+port',l:'network+port(复合)'},
  {v:'domain',l:'domain'},{v:'domain_suffix',l:'domain_suffix'},{v:'domain_keyword',l:'domain_keyword'},
  {v:'domain_regex',l:'domain_regex'},{v:'source_ip_cidr',l:'source_ip_cidr'},
  {v:'ip_cidr',l:'ip_cidr'},{v:'ip_is_private',l:'ip_is_private'},
  {v:'source_port',l:'source_port'},{v:'port',l:'port'},
  {v:'port_range',l:'port_range'},{v:'process_name',l:'process_name'},
  {v:'process_path',l:'process_path'},{v:'clash_mode',l:'clash_mode'},
  {v:'rule_set',l:'rule_set'},{v:'inline',l:'inline'},
]
const ROUTE_RULE_ACTIONS = [
  {v:'route',l:'路由'},{v:'route-options',l:'路由选项'},
  {v:'reject',l:'拒绝'},{v:'hijack-dns',l:'劫持DNS'},
  {v:'sniff',l:'协议嗅探'},{v:'resolve',l:'解析DNS'},
]
const DNS_RULE_ACTIONS = [
  {v:'route',l:'路由'},{v:'route-options',l:'路由选项'},{v:'reject',l:'拒绝'},
]
const DNS_SERVER_TYPES = ['udp','tcp','tls','https','quic','h3','fakeip','local']
const SNIFFERS = ['http','tls','quic','stun','dns','bittorrent','dtls']
const BUILTIN_OUTBOUNDS = [
  {id:'direct',tag:'direct',type:'Built-In'},
  {id:'block',tag:'block',type:'Built-In'},
]

// ── ID constants ───────────────────────────────────────────────────────────
const OB_IDS = {
  SELECT:'outbound-select', URLTEST:'outbound-urltest',
  DIRECT:'outbound-direct', BLOCK:'outbound-block',
  FALLBACK:'outbound-fallback', GLOBAL:'outbound-global',
}
const DNS_IDS = {
  FAKEIP:'Fake-IP', LOCAL:'Local-DNS', LOCAL_RESOLVER:'Local-DNS-Resolver',
  REMOTE:'Remote-DNS', REMOTE_RESOLVER:'Remote-DNS-Resolver',
}

// ── Form ───────────────────────────────────────────────────────────────────
function uid() { return Math.random().toString(36).slice(2,10) }
function genSecret() {
  const a = new Uint8Array(32); crypto.getRandomValues(a)
  return Array.from(a, b => b.toString(16).padStart(2,'0')).join('')
}

// Default outbounds (matching GUI4SB DefaultOutbounds)
function defaultOutbounds() {
  return [
    { id: OB_IDS.SELECT,   tag:'节点选择', type:'selector', outbounds:[{id:OB_IDS.URLTEST,tag:'自动选择',type:'Built-in'}], hidden:false, include:'', exclude:'', icon:'', url:'https://www.gstatic.com/generate_204', interval:'3m', tolerance:150, interrupt_exist_connections:true },
    { id: OB_IDS.URLTEST,  tag:'自动选择', type:'urltest',  outbounds:[], hidden:false, include:'', exclude:'', icon:'', url:'https://www.gstatic.com/generate_204', interval:'3m', tolerance:150, interrupt_exist_connections:true },
    { id: OB_IDS.DIRECT,   tag:'全局直连', type:'selector', outbounds:[{id:'direct',tag:'direct',type:'Built-in'},{id:'block',tag:'block',type:'Built-in'}], hidden:false, include:'', exclude:'', icon:'', url:'', interval:'3m', tolerance:150, interrupt_exist_connections:true },
    { id: OB_IDS.BLOCK,    tag:'全局拦截', type:'selector', outbounds:[{id:'block',tag:'block',type:'Built-in'},{id:'direct',tag:'direct',type:'Built-in'}], hidden:false, include:'', exclude:'', icon:'', url:'', interval:'3m', tolerance:150, interrupt_exist_connections:true },
    { id: OB_IDS.FALLBACK, tag:'漏网之鱼', type:'selector', outbounds:[{id:OB_IDS.SELECT,tag:'节点选择',type:'Built-in'},{id:OB_IDS.DIRECT,tag:'全局直连',type:'Built-in'}], hidden:false, include:'', exclude:'', icon:'', url:'', interval:'3m', tolerance:150, interrupt_exist_connections:true },
    { id: OB_IDS.GLOBAL,   tag:'GLOBAL',   type:'selector', outbounds:[{id:OB_IDS.SELECT,tag:'节点选择',type:'Built-in'},{id:OB_IDS.URLTEST,tag:'自动选择',type:'Built-in'},{id:OB_IDS.DIRECT,tag:'全局直连',type:'Built-in'},{id:OB_IDS.BLOCK,tag:'全局拦截',type:'Built-in'},{id:OB_IDS.FALLBACK,tag:'漏网之鱼',type:'Built-in'}], hidden:false, include:'', exclude:'', icon:'', url:'', interval:'3m', tolerance:150, interrupt_exist_connections:true },
  ]
}

function defaultInbounds() {
  return [
    { id:'mixed-in', type:'mixed', tag:'mixed-in', enable:true, listen:'127.0.0.1', listen_port:20122, usersText:'', tcp_fast_open:false, tcp_multi_path:false, udp_fragment:false },
    { id:'tun-in',   type:'tun',   tag:'tun-in',   enable:false, interface_name:'', addressText:'172.18.0.1/30, fdfe:dcba:9876::1/126', mtu:0, auto_route:true, strict_route:true, endpoint_independent_nat:false, stack:'mixed', route_address_text:'', route_exclude_address_text:'' },
  ]
}

function defaultRouteRules() {
  const IP = 'InsertionPoint'
  return [
    { id:uid(), type:'action',     enable:true,  payload:'',                action:'sniff',      outbound:'', invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'inbound',    enable:true,  payload:'dns-in',          action:'hijack-dns', outbound:'', invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'clash_mode', enable:true,  payload:'direct',          action:'route',      outbound:OB_IDS.DIRECT,   invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'clash_mode', enable:true,  payload:'global',          action:'route',      outbound:OB_IDS.GLOBAL,   invert:false, sniffer:[], strategy:'default', server:'' },
    { id:IP,    type:IP,           enable:true,  payload:'',                action:'route',      outbound:'', invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'network+port', enable:true, payload:'udp:443',        action:'reject',     outbound:'', invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'Category-Ads',    action:'route',      outbound:OB_IDS.BLOCK,    invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoSite-Private', action:'route',      outbound:OB_IDS.DIRECT,   invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoSite-CN',      action:'route',      outbound:OB_IDS.DIRECT,   invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoIP-Private',   action:'route',      outbound:OB_IDS.DIRECT,   invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoIP-CN',        action:'route',      outbound:OB_IDS.DIRECT,   invert:false, sniffer:[], strategy:'default', server:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoLocation-!CN', action:'route',      outbound:OB_IDS.SELECT,   invert:false, sniffer:[], strategy:'default', server:'' },
  ]
}

function defaultDnsServers() {
  return [
    { id:DNS_IDS.FAKEIP, tag:DNS_IDS.FAKEIP, type:'fakeip', detour:'', domain_resolver:'', server:'', server_port:'', path:'', inet4_range:'198.18.0.0/15', inet6_range:'fc00::/18' },
    { id:DNS_IDS.LOCAL, tag:DNS_IDS.LOCAL, type:'https', detour:'', domain_resolver:DNS_IDS.LOCAL_RESOLVER, server:'223.5.5.5', server_port:'443', path:'/dns-query', inet4_range:'', inet6_range:'' },
    { id:DNS_IDS.LOCAL_RESOLVER, tag:DNS_IDS.LOCAL_RESOLVER, type:'udp', detour:'', domain_resolver:'', server:'223.5.5.5', server_port:'53', path:'', inet4_range:'', inet6_range:'' },
    { id:DNS_IDS.REMOTE, tag:DNS_IDS.REMOTE, type:'tls', detour:OB_IDS.SELECT, domain_resolver:DNS_IDS.REMOTE_RESOLVER, server:'8.8.8.8', server_port:'853', path:'', inet4_range:'', inet6_range:'' },
    { id:DNS_IDS.REMOTE_RESOLVER, tag:DNS_IDS.REMOTE_RESOLVER, type:'udp', detour:OB_IDS.SELECT, domain_resolver:'', server:'8.8.8.8', server_port:'53', path:'', inet4_range:'', inet6_range:'' },
  ]
}

function defaultDnsRules() {
  const IP = 'InsertionPoint'
  return [
    { id:uid(), type:'clash_mode', enable:true,  payload:'direct',          action:'route', server:DNS_IDS.LOCAL,   invert:false, strategy:'default', disable_cache:false, client_subnet:'' },
    { id:uid(), type:'clash_mode', enable:true,  payload:'global',          action:'route', server:DNS_IDS.REMOTE,  invert:false, strategy:'default', disable_cache:false, client_subnet:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoSite-CN',      action:'route', server:DNS_IDS.LOCAL,   invert:false, strategy:'default', disable_cache:false, client_subnet:'' },
    { id:IP,    type:IP,           enable:true,  payload:'',                action:'route', server:'',              invert:false, strategy:'default', disable_cache:false, client_subnet:'' },
    { id:'__fakeip__', type:'rule_set', enable:false, payload:'__fakeip__', action:'route', server:DNS_IDS.FAKEIP,  invert:false, strategy:'default', disable_cache:false, client_subnet:'' },
    { id:uid(), type:'rule_set',   enable:true,  payload:'GeoLocation-!CN', action:'route', server:DNS_IDS.REMOTE,  invert:false, strategy:'default', disable_cache:false, client_subnet:'' },
  ]
}

function defaultRulesets() {
  const base = 'https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo'
  return [
    { id:'Category-Ads',    type:'remote', tag:'Category-Ads',    format:'binary', url:`${base}/geosite/category-ads-all.srs`, download_detour:OB_IDS.DIRECT, update_interval:'', path:'', rules:'' },
    { id:'GeoIP-Private',   type:'remote', tag:'GeoIP-Private',   format:'binary', url:`${base}/geoip/private.srs`,            download_detour:OB_IDS.DIRECT, update_interval:'', path:'', rules:'' },
    { id:'GeoSite-Private', type:'remote', tag:'GeoSite-Private', format:'binary', url:`${base}/geosite/private.srs`,          download_detour:OB_IDS.DIRECT, update_interval:'', path:'', rules:'' },
    { id:'GeoIP-CN',        type:'remote', tag:'GeoIP-CN',        format:'binary', url:`${base}/geoip/cn.srs`,                 download_detour:OB_IDS.DIRECT, update_interval:'', path:'', rules:'' },
    { id:'GeoSite-CN',      type:'remote', tag:'GeoSite-CN',      format:'binary', url:`${base}/geosite/cn.srs`,               download_detour:OB_IDS.DIRECT, update_interval:'', path:'', rules:'' },
    { id:'GeoLocation-!CN', type:'remote', tag:'GeoLocation-!CN', format:'binary', url:`${base}/geosite/geolocation-!cn.srs`,  download_detour:OB_IDS.DIRECT, update_interval:'', path:'', rules:'' },
  ]
}

function buildForm() {
  return {
    name: '',
    inbounds: [],
    outbounds: defaultOutbounds(),
    route: {
      find_process:false, auto_detect_interface:true, default_interface:'',
      final:OB_IDS.FALLBACK, default_domain_resolver:DNS_IDS.LOCAL,
      rule_set: defaultRulesets(),
      rules:    defaultRouteRules(),
    },
    dns: {
      disable_cache:false, disable_expire:false, independent_cache:false,
      client_subnet:'', strategy:'default', final:DNS_IDS.REMOTE,
      servers: defaultDnsServers(),
      rules:   defaultDnsRules(),
    },
  }
}

const form = reactive(buildForm())

onMounted(async () => {
  await rsStore.scan()
  nodesStore.load()
  subsStore.load()
  if (!props.profile) return
  form.name           = props.profile.name           || ''
  const wc = props.profile.wizardConfig
  if (!wc) return
  const src = typeof wc === 'string' ? JSON.parse(wc) : wc
  if (src.outbounds?.length) form.outbounds = JSON.parse(JSON.stringify(src.outbounds))
  if (src.route) {
    const {rule_set, rules, ...rest} = src.route
    Object.assign(form.route, rest)
    if (rule_set?.length) form.route.rule_set = JSON.parse(JSON.stringify(rule_set))
    if (rules?.length)    form.route.rules    = JSON.parse(JSON.stringify(rules))
  }
  if (src.dns) {
    const {servers, rules, ...rest} = src.dns
    Object.assign(form.dns, rest)
    if (servers?.length) form.dns.servers = JSON.parse(JSON.stringify(servers))
    if (rules?.length) {
      const loadedDnsRules = JSON.parse(JSON.stringify(rules))
      if (!loadedDnsRules.find(r => r.type === 'InsertionPoint')) {
        const IP = 'InsertionPoint'
        loadedDnsRules.unshift({ id:IP, type:IP, enable:true, payload:'', action:'route', server:'', invert:false, strategy:'default', disable_cache:false, client_subnet:'' })
      }
      form.dns.rules = loadedDnsRules
    }
  }
})

// ── Drag-and-drop ──────────────────────────────────────────────────────────
// Pure Vue reactive approach: no DOM insertion, no re-bind on list change.
// Each row gets @mousedown="onRowMousedown(e, listId, idx, list)".
// The template renders a blue highlight border on the dragState.to row.

const ibDragEl        = ref(null)
const obDragEl        = ref(null)
const rsDragEl        = ref(null)
const routeRuleDragEl = ref(null)
const dnsRuleDragEl   = ref(null)
const dnsSrvDragEl    = ref(null)
const sortDragEl      = ref(null)

const dragState = reactive({ listId: null, from: -1, to: -1 })
function isDragging(listId, idx) { return dragState.listId === listId && dragState.from === idx }
function isDragTarget(listId, idx) { return dragState.listId === listId && dragState.to === idx && dragState.from !== idx }

function onRowMousedown(e, listId, fromIdx, list) {
  if (e.target.closest('button,input,select,textarea,a')) return
  e.preventDefault()
  dragState.listId = listId
  dragState.from   = fromIdx
  dragState.to     = fromIdx

  function onMove(ev) {
    const rows = document.querySelectorAll('[data-drag-list="' + listId + '"] > [data-drag-row]')
    let closest = fromIdx, minDist = Infinity
    rows.forEach((row, i) => {
      const r = row.getBoundingClientRect()
      const dist = Math.abs(ev.clientY - (r.top + r.height / 2))
      if (dist < minDist) { minDist = dist; closest = i }
    })
    dragState.to = closest
  }

  function onUp() {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    const { from, to } = dragState
    if (from !== to && from >= 0 && to >= 0) {
      const moved = list.splice(from, 1)[0]
      list.splice(to, 0, moved)
    }
    dragState.listId = null; dragState.from = -1; dragState.to = -1
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// kept for other lists that still use the old pattern (inbounds, outbounds etc)
function useSortable(containerRef, listData) {
  const getList = () => Array.isArray(listData) ? listData : listData.value
  let dragging = null, over = null
  function rowsOf(c) { return [...c.children] }
  function onMousedown(e) {
    const container = containerRef.value; if (!container) return
    if (e.target.closest('button,input,select,textarea,a')) return
    const rows = rowsOf(container)
    const row  = rows.find(el => el.contains(e.target)); if (!row) return
    e.preventDefault()
    dragging = rows.indexOf(row)
    over = dragging
    row.style.opacity = '.45'
    const ghost = row
    function onMove(ev) {
      rows.forEach((r2, i) => {
        const br = r2.getBoundingClientRect()
        if (ev.clientY >= br.top && ev.clientY <= br.bottom) over = i
      })
    }
    function onUp() {
      document.removeEventListener('mousemove', onMove)
      document.removeEventListener('mouseup', onUp)
      ghost.style.opacity = ''
      if (dragging !== over) {
        const list = getList()
        const moved = list.splice(dragging, 1)[0]
        list.splice(over, 0, moved)
      }
      dragging = null; over = null
    }
    document.addEventListener('mousemove', onMove)
    document.addEventListener('mouseup', onUp)
  }
  watch(containerRef, (el, oldEl) => {
    if (oldEl) oldEl.removeEventListener('mousedown', onMousedown)
    if (el) el.addEventListener('mousedown', onMousedown)
  }, { immediate: true })
}

useSortable(ibDragEl,   form.inbounds)
useSortable(obDragEl,   form.outbounds)
useSortable(rsDragEl,   form.route.rule_set)
useSortable(dnsSrvDragEl, form.dns.servers)

// route.rules and dns.rules use onRowMousedown directly in the template
// (these lists are inside v-if tabs and need no containerRef watch)

// ── Inbounds ───────────────────────────────────────────────────────────────
function addInbound(type) {
  const defaults = {
    mixed:    { listen:'127.0.0.1', listen_port:20122, usersText:'', tcp_fast_open:false, tcp_multi_path:false, udp_fragment:false },
    http:     { listen:'127.0.0.1', listen_port:20121, usersText:'', tcp_fast_open:false, tcp_multi_path:false, udp_fragment:false },
    socks:    { listen:'127.0.0.1', listen_port:20120, usersText:'', tcp_fast_open:false, tcp_multi_path:false, udp_fragment:false },
    redirect: { listen:'127.0.0.1', listen_port:7892,  usersText:'', tcp_fast_open:false, tcp_multi_path:false, udp_fragment:false, route_address:'' },
    tproxy:   { listen:'127.0.0.1', listen_port:7893,  usersText:'', tcp_fast_open:false, tcp_multi_path:false, udp_fragment:false, route_address:'' },
    tun:      { interface_name:'', addressText:'172.18.0.1/30, fdfe:dcba:9876::1/126', mtu:0, auto_route:true, strict_route:true, endpoint_independent_nat:false, stack:'mixed', route_address_text:'', route_exclude_address_text:'' },
  }
  form.inbounds.push({ id:uid(), type, tag: type+'-in', enable:true, ...defaults[type] })
}

// ── Outbounds ──────────────────────────────────────────────────────────────
const showObModal   = ref(false)
const showSortModal = ref(false)
let   editObIdx     = -1
const obFields      = reactive({ id:'', tag:'', type:'selector', hidden:false, include:'', exclude:'', icon:'', url:'https://www.gstatic.com/generate_204', interval:'3m', tolerance:150, interrupt_exist_connections:true, outbounds:[] })
const expandedGroups = reactive(new Set(['builtin','subscription']))

function openOutboundEdit(idx) {
  editObIdx = idx
  if (idx === null) {
    Object.assign(obFields, { id:uid(), tag:'', type:'selector', hidden:false, include:'', exclude:'', icon:'', url:'https://www.gstatic.com/generate_204', interval:'3m', tolerance:150, interrupt_exist_connections:true, outbounds:[] })
  } else {
    Object.assign(obFields, JSON.parse(JSON.stringify(form.outbounds[idx])))
  }
  showObModal.value = true
}
function saveOutbound() {
  const ob = JSON.parse(JSON.stringify(obFields))
  const tag = ob.tag.trim()
  if (!tag) { alert('名称不能为空'); return }
  const duplicate = form.outbounds.some(o => o.id !== ob.id && o.tag === tag)
  if (duplicate) { alert(`出站名称「${tag}」已存在，请使用其他名称`); return }
  ob.tag = tag
  if (editObIdx === null) { form.outbounds.push(ob) }
  else { form.outbounds[editObIdx] = ob }
  showObModal.value = false
}
function deleteOutbound(idx) {
  if (!confirm(`删除出站「${form.outbounds[idx].tag}」？`)) return
  form.outbounds.splice(idx, 1)
}
function countOutbounds(ob) {
  if (!['selector','urltest'].includes(ob.type)) return 0
  return (ob.outbounds||[]).filter(r => r.type !== 'Subscription').length
}
function countSubs(ob) {
  if (!['selector','urltest'].includes(ob.type)) return 0
  return (ob.outbounds||[]).filter(r => r.type === 'Subscription').length
}

const allProxies = computed(() => [
  ...BUILTIN_OUTBOUNDS,
  ...form.outbounds.filter(ob => ob.id !== obFields.id).map(ob => ({ id:ob.id, tag:ob.tag, type:ob.type })),
])
function isInRefs(id) { return obFields.outbounds.some(r => r.id === id) }
function toggleRef(id, tag, type='Built-in') {
  const idx = obFields.outbounds.findIndex(r => r.id === id)
  if (idx >= 0) obFields.outbounds.splice(idx, 1)
  else obFields.outbounds.push({ id, tag, type })
}
function toggleGroup(k) {
  if (expandedGroups.has(k)) expandedGroups.delete(k)
  else expandedGroups.add(k)
}

// Sort modal
function openSortModal(idx) {
  editObIdx = idx
  Object.assign(obFields, JSON.parse(JSON.stringify(form.outbounds[idx])))
  showSortModal.value = true
  nextTick(() => useSortable(sortDragEl, obFields.outbounds))
}
function saveSortModal() {
  form.outbounds[editObIdx].outbounds = [...obFields.outbounds]
  showSortModal.value = false
}

// ── Ruleset ────────────────────────────────────────────────────────────────
const showRsModal = ref(false)
let   editRsIdx   = -1
const rsFields    = reactive({ id:'', type:'remote', tag:'', format:'binary', url:'', download_detour:'direct', update_interval:'', path:'', rules:'' })

function openRulesetEdit(idx) {
  editRsIdx = idx
  if (idx === null) {
    Object.assign(rsFields, { id:uid(), type:'remote', tag:'', format:'binary', url:'', download_detour:'direct', update_interval:'', path:'', rules:'' })
  } else {
    Object.assign(rsFields, JSON.parse(JSON.stringify(form.route.rule_set[idx])))
  }
  showRsModal.value = true
}
function saveRuleset() {
  const rs = { ...rsFields }
  if (!rs.id) rs.id = uid()
  const tag = rs.tag.trim()
  if (!tag) { alert('规则集名称不能为空'); return }
  const duplicate = form.route.rule_set.some(r => r.id !== rs.id && r.tag === tag)
  if (duplicate) { alert(`规则集名称「${tag}」已存在，请使用其他名称`); return }
  rs.tag = tag
  if (editRsIdx === null) form.route.rule_set.push(rs)
  else form.route.rule_set[editRsIdx] = rs
  showRsModal.value = false
}
function fmtSize(b) {
  if (!b) return '—'
  if (b < 1024) return b + 'B'
  if (b < 1048576) return (b/1024).toFixed(1)+'KB'
  return (b/1048576).toFixed(2)+'MB'
}

// ── Route rules ────────────────────────────────────────────────────────────
const showRouteRuleModal = ref(false)
let   editRrIdx = -1
const rrFields  = reactive({ id:'', type:'rule_set', action:'route', payload:'', invert:false, outbound:'', sniffer:[], strategy:'default', server:'' })

function openRouteRuleEdit(idx) {
  editRrIdx = idx
  if (idx === null) {
    Object.assign(rrFields, { id:uid(), type:'rule_set', action:'route', payload:'', invert:false, outbound:'', sniffer:[], strategy:'default', server:'' })
  } else {
    Object.assign(rrFields, JSON.parse(JSON.stringify(form.route.rules[idx])))
  }
  showRouteRuleModal.value = true
}
function saveRouteRule() {
  const r = { ...rrFields, sniffer: [...rrFields.sniffer] }
  if (!r.id) r.id = uid()
  if (editRrIdx === null) {
    // insert after InsertionPoint if exists
    const ipIdx = form.route.rules.findIndex(x => x.type === 'InsertionPoint')
    if (ipIdx >= 0) form.route.rules.splice(ipIdx + 1, 0, r)
    else form.route.rules.unshift(r)
  } else {
    form.route.rules[editRrIdx] = r
  }
  showRouteRuleModal.value = false
}
function addInsertionPoint(target) {
  const list = target === 'route' ? form.route.rules : form.dns.rules
  if (list.find(r => r.type === 'InsertionPoint')) return
  const IP = { id:'InsertionPoint', type:'InsertionPoint', enable:true, payload:'', action:'route', outbound:'', invert:false, sniffer:[], strategy:'default', server:'' }
  list.unshift(IP)
}

function rrRulesetSelected(id) {
  return rrFields.payload.split(',').includes(id)
}
function drRulesetSelected(id) {
  return drFields.payload.split(',').includes(id)
}
function toggleRulesetInRule(id, fields) {
  const ids = fields.payload.split(',').filter(Boolean)
  const i = ids.indexOf(id)
  if (i >= 0) { ids.splice(i, 1) } else { ids.push(id) }
  fields.payload = ids.join(',')
}

// ── DNS Server ─────────────────────────────────────────────────────────────
const showDnsSrvModal = ref(false)
let   editDnsSrvIdx   = -1
const dnsSrvFields    = reactive({ id:'', tag:'', type:'udp', server:'', server_port:'', path:'', domain_resolver:'', detour:'', inet4_range:'', inet6_range:'' })

function openDnsServerEdit(idx) {
  editDnsSrvIdx = idx
  if (idx === null) {
    Object.assign(dnsSrvFields, { id:uid(), tag:'', type:'udp', server:'', server_port:'53', path:'', domain_resolver:'', detour:'', inet4_range:'', inet6_range:'' })
  } else {
    Object.assign(dnsSrvFields, JSON.parse(JSON.stringify(form.dns.servers[idx])))
  }
  showDnsSrvModal.value = true
}
function saveDnsServer() {
  const s = { ...dnsSrvFields }
  if (!s.id) s.id = uid()
  const tag = s.tag.trim()
  if (!tag) { alert('DNS 服务器名称不能为空'); return }
  const duplicate = form.dns.servers.some(srv => srv.id !== s.id && srv.tag === tag)
  if (duplicate) { alert(`DNS 服务器名称「${tag}」已存在，请使用其他名称`); return }
  s.tag = tag
  if (editDnsSrvIdx === null) form.dns.servers.push(s)
  else form.dns.servers[editDnsSrvIdx] = s
  showDnsSrvModal.value = false
}

// ── DNS Rules ──────────────────────────────────────────────────────────────
const showDnsRuleModal = ref(false)
let   editDrIdx  = -1
const drFields   = reactive({ id:'', type:'rule_set', action:'route', payload:'', invert:false, server:'', strategy:'default', disable_cache:false, client_subnet:'' })

function openDnsRuleEdit(idx) {
  editDrIdx = idx
  if (idx === null) {
    Object.assign(drFields, { id:uid(), type:'rule_set', action:'route', payload:'', invert:false, server:'', strategy:'default', disable_cache:false, client_subnet:'' })
  } else {
    Object.assign(drFields, JSON.parse(JSON.stringify(form.dns.rules[idx])))
  }
  showDnsRuleModal.value = true
}
function saveDnsRule() {
  const r = { ...drFields }
  if (!r.id) r.id = uid()
  if (editDrIdx === null) {
    const ipIdx = form.dns.rules.findIndex(x => x.type === 'InsertionPoint')
    if (ipIdx >= 0) form.dns.rules.splice(ipIdx + 1, 0, r)
    else form.dns.rules.push(r)
  } else {
    form.dns.rules[editDrIdx] = r
  }
  showDnsRuleModal.value = false
}

// ── Labels & helpers ───────────────────────────────────────────────────────
const allOutboundOptions = computed(() => [
  ...form.outbounds.map(ob => ({ v:ob.id, l:ob.tag })),
  { v:'direct', l:'direct' }, { v:'block', l:'block' },
])

function outboundLabel(id) {
  const ob = form.outbounds.find(o => o.id === id)
  if (ob) return ob.tag
  return id
}
function dnsServerTagById(id) {
  const s = form.dns.servers.find(s => s.id === id)
  return s ? s.tag : id
}
function dnsServerSummary(srv) {
  if (srv.type === 'fakeip') return `${srv.inet4_range} / ${srv.inet6_range}`
  if (srv.type === 'local') return '系统本地 DNS'
  const port = srv.server_port ? ':' + srv.server_port : ''
  const path = srv.path || ''
  return `${srv.server}${port}${path}`
}
function renderRouteRulePayload(rule) {
  if (rule.type === 'action') return '(无条件匹配)'
  if (rule.type === 'network+port') return rule.payload || 'udp:443'
  if (rule.type === 'rule_set') {
    return rule.payload.split(',').map(id => {
      const rs = form.route.rule_set.find(r => r.id === id)
      return rs ? rs.tag : id
    }).join(', ')
  }
  return rule.payload || '—'
}
function renderDnsRulePayload(rule) {
  if (rule.payload === '__fakeip__') return 'FakeIP (logical+and)'
  return renderRouteRulePayload(rule)
}

// ── Navigation ─────────────────────────────────────────────────────────────
function nextStep() {
  if (step.value === 0 && !form.name.trim()) { alert('请填写配置名称'); return }
  step.value++
}

// ── Save ───────────────────────────────────────────────────────────────────
async function save() {
  if (!form.name.trim()) { step.value = 0; return }
  saving.value = true; saveErrors.value = []
  try {
    const wizardConfig = JSON.parse(JSON.stringify({
      outbounds: form.outbounds,
      route:     form.route,
      dns:       form.dns,
    }))

    // Validate before saving
    try {
      const { api } = await import('../api.js')
      const vRes = await api('POST', '/profiles/validate', { wizardConfig })
      if (!vRes.ok && vRes.errors?.length) {
        saveErrors.value = vRes.errors
        // Ask user if they want to save anyway
        const proceed = confirm(
          '⚠ 配置验证发现以下错误：

' +
          vRes.errors.map(e => `• [${e.location}] ${e.message}`).join('
') +
          '

是否仍然保存？（建议修正后再保存）'
        )
        if (!proceed) { saving.value = false; return }
      }
    } catch {}

    let prof
    if (props.profile) {
      prof = await profilesStore.updateMeta(props.profile.id, {
        name: form.name.trim(),
        wizardConfig,
      })
    } else {
      prof = await profilesStore.add(form.name.trim(), wizardConfig)
    }
    emit('saved', prof)
  } catch (e) {
    alert('保存失败: ' + e.message)
  } finally { saving.value = false }
}
</script>

<style scoped>
.ob-ref-group { border: 1px solid var(--border); border-radius: var(--radius); overflow:hidden; }
.ob-ref-group-head {
  display:flex; align-items:center; padding:8px 12px; cursor:pointer;
  background:var(--surface2); font-size:13px; transition:background .12s;
}
.ob-ref-group-head:hover { background:var(--bg); }
.ob-ref-grid { display:grid; grid-template-columns:repeat(4,1fr); gap:6px; padding:8px; }
.ob-ref-item {
  display:flex; flex-direction:column; align-items:flex-start; gap:1px;
  padding:6px 8px; border:1.5px solid var(--border); border-radius:var(--radius);
  background:var(--surface2); cursor:pointer; transition:all .12s; text-align:left;
}
.ob-ref-item:hover { border-color:var(--border2); background:var(--surface); }
.ob-ref-item.on { border-color:var(--accent); background:var(--accent-bg); }
</style>
