'use strict';
(() => {
  const people = {
    a: { name: '小李', full: '李沐川', initials: '李', rank: '黄金 III', score: '1,680', tag: '稳定发挥，偶尔超神' },
    b: { name: '小王', full: '王予安', initials: '王', rank: '黄金 II', score: '1,520', tag: '好球友，也是老对手' },
    c: { name: '小陈', full: '陈嘉树', initials: '陈', rank: '白银 I', score: '1,280', tag: '最近一起打过球' }
  };
  const labels = { pending: '待回应', accepted: '已接受', playing: '进行中', completed: '已完成', canceled: '已取消', rejected: '已拒绝', expired: '已失效' };
  const scenes = [
    ['fresh', '从发起开始', '选一位球友，定个大概时间。右侧手机会收到你发出的邀请。'],
    ['pending', '收到与发出', '收到邀请只需要回应，不提前解释开局。接收方也可以暂不处理。'],
    ['accepted', '已约好 · 双入口', '同一条约球，出现在对局与我的。点击进入后，对手才会看到你在等待。'],
    ['waiting', '一人等待 · 一人选择', '左侧没有比分，也没有比赛。右侧进入才正式开始，放弃则一起结束。'],
    ['playing', '正式比赛与再约', '任意一方滑动结束，无需对方同意；未滑到终点会复位。结束后约球同步完成，真实积分计算不在预览范围内。'],
    ['mutual', '同人互邀匹配', '同一个人、同一种类型，直接合并。时间不同也可以，以正在接受的邀请为准。'],
    ['different', '互邀类型不同', '类型不同不悄悄匹配。查看并处理原邀请后，再决定是否接受。'],
    ['replace', '换人接受邀请', '小李发给小王的邀请还未回应，同时收到小陈的邀请。接受前明确提示关闭旧邀请。'],
    ['expired', '跨午夜与失效', '用演示时间切换观察昨天、今天的文案。到07:00只显示已失效，不加冗长说明。'],
    ['history', '对局详情 · 再约一场', '历史比赛也能再约。自动带上对手和类型，时间重新选择，不直接发送。'],
    ['settings', '仅允许好友约球', '独立模拟开关默认关闭。本场景把A/B设为非好友最近对手，可先开B的开关，再让A发送。']
  ];
  const paths = {
    back: 'M15 5l-7 7 7 7', arrow: 'M5 12h14m-6-6 6 6-6 6', chevron: 'M9 5l7 7-7 7',
    clock: 'M12 7v5l3 2M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0',
    ball: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0',
    mail: 'M4 5h16v14H4zM4 6l8 6 8-6', history: 'M4 10a8 8 0 1 1 1 7M4 4v6h6m2-3v5l3 2',
    user: 'M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0M4 21v-2a8 8 0 0 1 16 0v2',
    home: 'M3 10l9-7 9 7M5 9v12h5v-7h4v7h5V9', chart: 'M4 21V12h4v9M10 21V5h4v16M16 21V9h4v12',
    gear: 'M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8M9 3h6l1 3 3 1 2 5-2 5-3 1-1 3H9l-1-3-3-1-2-5 2-5 3-1z',
    scan: 'M8 3H3v5m13-5h5v5M3 16v5h5m13-5v5h-5M3 12h18',
    qr: 'M3 3h6v6H3zM15 3h6v6h-6zM3 15h6v6H3zm12 0h3v3h-3zm3 3h3v3h-3m-3 3v-3m6-3v-3',
    plus: 'M12 5v14M5 12h14', check: 'M5 12l4 4L19 6', shield: 'M12 3l8 3v6c0 5-8 9-8 9s-8-4-8-9V6zM8 12l3 3 5-6',
    trophy: 'M8 3h8v7a4 4 0 0 1-8 0zm8 2h4v3a4 4 0 0 1-4 4M8 5H4v3a4 4 0 0 0 4 4m4 2v6m-4 1h8',
    info: 'M12 11v6m0-10v1M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0', bell: 'M6 9a6 6 0 0 1 12 0v6l2 3H4l2-3zm4 12h4',
    users: 'M10 7a3 3 0 1 1-6 0 3 3 0 0 1 6 0m4-2a3 3 0 0 1 0 6M2 20v-2a5 5 0 0 1 10 0v2m3-6a5 5 0 0 1 5 5v1'
  };
  const icon = name => `<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="${paths[name] || paths.ball}"/></svg>`;
  const escape = value => String(value ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const DAY = 86400000;
  const BASE = Date.parse('2026-08-25T00:00:00+08:00');
  let now = BASE + 16 * 3600000, nextId = 1, invites = [], scene = 'accepted', abFriends = true;
  let prefs = { a: false, b: false, c: false };
  try { const saved = JSON.parse(localStorage.getItem('cp-quick-preview-preferences')); for (const u of ['a', 'b', 'c']) prefs[u] = saved?.[u] === true; } catch (_) { /* file:// 存储不可用时仍可完整预览。 */ }
  const views = { a: { page: 'home', tab: 'home', draft: null, dialog: null }, b: { page: 'home', tab: 'home', draft: null, dialog: null } };
  const messages = { a: '', b: '' }, toastTimers = {};
  const members = r => [r.from, r.to];
  const opponent = (r, u) => r.from === u ? r.to : r.from;
  const isFriend = (a, b) => abFriends && [a, b].sort().join('') === 'ab';
  const involved = u => invites.filter(r => members(r).includes(u) && r.status !== 'merged');
  const current = u => involved(u).find(r => ['accepted', 'playing'].includes(r.status));
  const outgoing = u => involved(u).find(r => r.from === u && r.status === 'pending');
  const incoming = u => involved(u).filter(r => r.to === u && r.status === 'pending');
  const get = id => invites.find(r => r.id === Number(id));
  const today = () => Math.floor((now + 8 * 3600000) / DAY) * DAY - 8 * 3600000;
  function relative(day) {
    const d = Math.round((day - today()) / DAY);
    if (d === 0) return '今天'; if (d === 1) return '明天'; if (d === 2) return '后天'; if (d === -1) return '昨天';
    const date = new Date(day + 8 * 3600000), year = date.getUTCFullYear();
    return `${year !== new Date(now + 8 * 3600000).getUTCFullYear() ? year + '年' : ''}${date.getUTCMonth() + 1}月${date.getUTCDate()}日`;
  }
  const schedule = r => `${relative(r.day)} ${r.start}点—${r.end}点`;
  const expires = r => r.day + DAY + 7 * 3600000;
  const expiryText = r => r.status === 'expired' ? '' : `${relative(r.day + DAY)}早晨7点失效`;
  const clockText = () => new Date(now + 8 * 3600000).toISOString().slice(11, 16);
  function newInvite(from, to, options = {}) {
    const record = { id: nextId++, from, to, game: '中式八球', mode: '排位', day: BASE, start: 16, end: 17, note: '', status: 'pending', entered: [], matchId: null, score: [0, 0], ...options };
    invites.push(record); return record;
  }
  function defaultDraft(u, to, source) {
    const start = Math.floor((now - today()) / 3600000) + 1;
    return { to: to || (u === 'a' ? 'b' : 'a'), day: today() + (start >= 24 ? DAY : 0), start: start >= 24 ? 0 : start, end: start >= 24 ? 1 : start + 1, game: source?.game || '中式八球', mode: source?.mode || '排位', note: '' };
  }
  function expireRecords() {
    for (const r of invites) if (['pending', 'accepted'].includes(r.status) && now >= expires(r)) {
      r.expiredFrom = r.status; r.status = 'expired'; r.entered = [];
      for (const u of ['a', 'b']) if (views[u].recordId === r.id && views[u].page === 'waiting') { views[u].page = views[u].tab; messages[u] = '本次约球已失效'; }
    }
  }
  function setDemoTime(value) {
    now = Date.parse(value);
    // 仅演示时钟允许倒放，恢复被时钟推进过期的邀请；不恢复用户取消的邀请。
    for (const r of invites) if (r.status === 'expired' && r.expiredFrom && now < expires(r)) { r.status = r.expiredFrom; delete r.expiredFrom; }
    render();
  }
  function toast(u, text) {
    messages[u] = text; clearTimeout(toastTimers[u]);
    toastTimers[u] = setTimeout(() => { messages[u] = ''; render(); }, 4200);
  }
  function modal(u, title, text, confirm = '知道了', action = null, cancel = '取消', danger = false) {
    views[u].dialog = { title, text, confirm, action, cancel, danger };
    render(); document.querySelector(`#view-${u} .modal button`)?.focus();
  }
  function navigate(u, page, recordId) {
    views[u].page = page; if (recordId != null) views[u].recordId = Number(recordId);
    if (page === 'home' || page === 'me') views[u].tab = page;
  }
  function blocker(u) {
    const r = current(u) || outgoing(u);
    if (!r) return false;
    modal(u, current(u) ? '已有未结束的约球' : '邀请还在等待回应', `请先处理与${people[opponent(r, u)].name}的当前约球。`, '查看当前约球', () => navigate(u, views[u].tab), '知道了');
    return true;
  }
  function compose(u, to, source) {
    if (blocker(u)) return;
    views[u].draft = defaultDraft(u, to, source); navigate(u, 'compose'); views[u].error = '';
  }
  function send(u) {
    expireRecords(); if (blocker(u)) return;
    const d = views[u].draft;
    if (!d || d.end <= d.start || d.day + d.end * 3600000 <= now || d.day < today() || d.day > today() + 2 * DAY) { views[u].error = '请选择尚未结束的时间段，结束时间需晚于开始时间。'; return; }
    if (prefs[d.to] && !isFriend(u, d.to)) { views[u].error = '对方仅允许好友约球，可以更换一位球友。'; return; }
    newInvite(u, d.to, d); navigate(u, 'home'); toast(u, '约球已发送'); toast(d.to, `${people[u].name}向你发来约球邀请`);
  }
  function accept(u, id, replaceConfirmed = false) {
    expireRecords(); const r = get(id);
    if (!r || r.to !== u || r.status !== 'pending') { toast(u, '这条邀请已处理，请查看最新状态'); return; }
    if (current(u) || current(r.from)) { modal(u, '已有未结束的约球', current(u) ? '请先处理当前约球，再接受其他邀请。' : '对方已有未结束的约球，本次暂时无法接受。', '查看约球', () => navigate(u, views[u].tab)); return; }
    const own = outgoing(u);
    if (own && own.to === r.from && (own.game !== r.game || own.mode !== r.mode)) { modal(u, '约球类型不同', '你们发起的约球类型不同，不能直接匹配。请先处理已发出的邀请。', '查看已发出的邀请', () => navigate(u, 'home')); return; }
    if (own && own.to !== r.from && !replaceConfirmed) {
      modal(u, `接受${people[r.from].name}的约球？`, `接受后会自动关闭已发出的邀请\n原邀请：你发给${people[own.to].name}的约球`, '确认接受', () => accept(u, id, true), '暂不接受'); return;
    }
    if (own) { own.status = own.to === r.from ? 'merged' : 'canceled'; own.mergedInto = own.status === 'merged' ? r.id : null; toast(own.to, own.status === 'merged' ? '已约好，与对方的邀请已合并' : '对方已取消本次邀请'); }
    r.status = 'accepted'; navigate(u, views[u].tab); toast(u, own?.status === 'merged' ? '已约好，与对方的邀请已合并' : '已接受约球'); toast(r.from, `${people[u].name}接受了约球`);
  }
  function enter(u, id, earlyConfirmed = false) {
    expireRecords(); const r = get(id);
    if (!r || !members(r).includes(u) || !['accepted', 'playing'].includes(r.status)) { toast(u, '本次约球已结束'); return; }
    if (r.status === 'playing') { navigate(u, 'playing', id); return; }
    if (r.entered.includes(u)) { navigate(u, 'waiting', id); return; }
    if (!earlyConfirmed && !r.entered.includes(opponent(r, u)) && now < r.day + r.start * 3600000) { modal(u, '现在进入吗？', `预计${relative(r.day)}${r.start}点开始。\n双方进入后才会正式开局。`, '现在进入', () => enter(u, id, true), '稍后再打'); return; }
    r.entered.push(u); navigate(u, 'waiting', id);
    if (members(r).every(p => r.entered.includes(p))) {
      r.status = 'playing'; r.matchId = `DEMO-${r.id}`;
      for (const p of ['a', 'b']) if (views[p].page === 'waiting' && views[p].recordId === r.id) navigate(p, 'playing', id);
      toast(opponent(r, u), '对方已进入，对局正式开始');
    } else toast(opponent(r, u), `${people[u].name}已进入，等你一起开始`);
  }
  function endInvite(u, id, status) {
    expireRecords(); const r = get(id);
    if (!r || !members(r).includes(u) || !['pending', 'accepted'].includes(r.status)) { toast(u, '状态已变化，请查看当前对局'); return; }
    r.status = status; r.entered = [];
    for (const p of ['a', 'b']) if (views[p].recordId === r.id && views[p].page === 'waiting') navigate(p, views[p].tab);
    toast(u, status === 'rejected' ? '已拒绝本次约球' : '本次约球已结束'); toast(opponent(r, u), status === 'rejected' ? '对方这次不参加' : '对方已放弃，本次约球已结束');
  }
  function leave(u, id) {
    const r = get(id); if (r?.status !== 'accepted') { toast(u, '对局已开始，请查看当前比赛'); return; }
    r.entered = r.entered.filter(p => p !== u); navigate(u, views[u].tab); toast(u, '已退出等待，约球仍保留');
  }
  function finish(u, id) {
    const r = get(id); if (r?.status !== 'playing' || !members(r).includes(u)) return;
    r.status = 'completed';
    for (const p of ['a', 'b']) if (views[p].page === 'playing' && views[p].recordId === r.id) navigate(p, 'detail', id);
    toast(u, '对局与约球均已结束'); toast(opponent(r, u), '对方已结束对局，可以再约一场');
  }
  function loadScene(name) {
    scene = name; now = BASE + 16 * 3600000; invites = []; nextId = 1; abFriends = true;
    for (const u of ['a', 'b']) { clearTimeout(toastTimers[u]); messages[u] = ''; views[u] = { page: 'home', tab: 'home', dialog: null, recordId: null, draft: null, error: '' }; }
    newInvite('a', 'b', { day: BASE - DAY * 2, status: 'completed', matchId: 'DEMO-history', score: [5, 3] });
    let r;
    if (['pending', 'accepted', 'waiting', 'playing', 'expired'].includes(name)) {
      r = newInvite('a', 'b', { status: name === 'pending' ? 'pending' : 'accepted' });
      if (name === 'accepted') { views.b.page = 'me'; views.b.tab = 'me'; }
      if (name === 'waiting') { r.entered = ['a']; navigate('a', 'waiting', r.id); }
      if (name === 'playing') { r.status = 'playing'; r.entered = ['a', 'b']; r.matchId = `DEMO-${r.id}`; r.score = [5, 3]; navigate('a', 'playing', r.id); navigate('b', 'playing', r.id); }
      if (name === 'expired') { now = BASE + DAY + 30 * 60000; navigate('a', 'history'); navigate('b', 'history'); }
    }
    if (name === 'mutual' || name === 'different') { newInvite('a', 'b'); newInvite('b', 'a', { start: 18, end: 19, mode: name === 'different' ? '练习' : '排位' }); navigate('a', 'inbox'); navigate('b', 'inbox'); }
    if (name === 'replace') { newInvite('a', 'b'); newInvite('c', 'a', { note: '老地方，打一场？' }); navigate('a', 'inbox'); }
    if (name === 'history') { navigate('a', 'detail', 1); navigate('b', 'history'); }
    if (name === 'settings') { abFriends = false; prefs.b = false; compose('a', 'b'); navigate('b', 'settings'); }
    render();
  }

  const avatar = (u, size = '') => `<span class="avatar ${u} ${size}" aria-hidden="true">${people[u].initials}</span>`;
  const person = (u, sub = '', size = '') => `<div class="person">${avatar(u, size)}<div class="player-info"><strong>${people[u].name}</strong><small>${sub || people[u].rank + ' · ' + people[u].tag}</small></div></div>`;
  const button = (text, action, id = '', cls = 'primary', extra = '') => `<button type="button" class="${cls}" data-action="${action}" data-id="${id}" ${extra}>${text}</button>`;
  const tags = r => `<div class="type-tags"><span>${escape(r.game)}</span><span>${escape(r.mode)}</span><span>自由局数</span></div>`;
  const timeBlock = r => `<div class="appointment-time">${icon('clock')}<span>${schedule(r)}</span></div>`;
  const textAction = (text, action, id) => button(text, action, id, 'text-btn');
  function card(u, r) {
    const other = opponent(r, u), mine = r.entered.includes(u), theirs = r.entered.includes(other), active = ['pending', 'accepted', 'playing'].includes(r.status);
    let state = labels[r.status], style = r.status, title = `与${people[other].name}的约球`, action = '', footer = '', note = '';
    if (r.status === 'pending') {
      state = r.from === u ? '等待回应' : '待你回应';
      if (r.to === u) { title = `${people[other].name}约你打一场`; action = `<div class="action-row">${button('这次不了', 'reject', r.id, 'secondary')}${button(current(u) ? '已有约球' : '接受约球', 'accept', r.id)}</div>`; }
      else footer = textAction('取消约球', 'cancel', r.id);
    }
    if (r.status === 'accepted') {
      state = mine ? '等待对方进入' : theirs ? '对方已进入' : '已接受'; style = mine || theirs ? 'waiting' : 'accepted';
      if (theirs) title = `${people[other].name}已进入，等你开始`;
      note = mine ? `你已进入，${people[other].name}还未进入` : theirs ? '' : '双方进入后，正式开始';
      action = `<div class="action-row">${button(mine ? '返回等待页面' : '进入对局 ' + icon('arrow'), 'enter', r.id)}</div>`;
      footer = (mine ? textAction('退出等待，保留约球', 'leave', r.id) : '') + textAction(theirs && !mine ? '放弃本次约球' : '取消约球', 'cancel', r.id);
    }
    if (r.status === 'playing') { title = '继续这场好球'; action = `<div class="action-row">${button('继续对局 ' + icon('arrow'), 'enter', r.id)}</div>`; note = `${r.score[0]} : ${r.score[1]} · 正在进行`; }
    if (!active) action = `<div class="action-row">${button(r.matchId ? '查看赛果' : '再约一场', r.matchId ? 'detail' : 'rematch', r.id, 'secondary')}</div>`;
    return `<article class="card"><div class="card-header"><h3>${title}</h3><span class="pill ${style} dot">${state}</span></div>${person(other)}${tags(r)}${timeBlock(r)}${active && r.status !== 'playing' ? `<p class="expire-copy">${expiryText(r)}</p>` : ''}${r.note ? `<p class="message-box">“${escape(r.note)}”</p>` : ''}${note ? `<p class="note">${note}</p>` : ''}${action}${footer ? `<div class="card-footer">${footer}</div>` : ''}</article>`;
  }
  function header(u, title, back = true, right = '') {
    return `<header class="screen-head">${back ? button(icon('back') + '返回', 'back', '', 'back') : `<div><h2>${title}</h2><div class="head-sub">CHASING POINTS</div></div>`}${back ? `<h2 class="center-title">${title}</h2>` : ''}${right || (back ? '<span class="head-spacer"></span>' : button(icon('mail'), 'inbox', '', 'icon-btn', 'aria-label="收到的邀请"'))}</header>`;
  }
  function tabs(u) {
    return `<nav class="tabs" aria-label="${people[u].name}的页面导航">${[['home', 'ball', '对局'], ['history', 'history', '记录'], ['inbox', 'mail', '邀请'], ['me', 'user', '我的']].map(([p, i, label]) => button(icon(i) + label, 'nav', p, views[u].page === p ? 'active' : '', `aria-label="${label}" ${views[u].page === p ? 'aria-current="page"' : ''}`)).join('')}</nav>`;
  }
  function recentRows(u, all = false) {
    const rows = involved(u).filter(r => r.matchId && r.status === 'completed');
    return (all ? rows : rows.slice(-1)).reverse().map(r => `<button type="button" class="record-row" data-action="detail" data-id="${r.id}">${avatar(opponent(r, u), 'small')}<span><strong>与${people[opponent(r, u)].name}的对局</strong><small>${r.game} · ${relative(r.day)}</small></span><span class="record-score">${r.score[0]} <span style="color:var(--muted)">:</span> ${r.score[1]}</span></button>`).join('');
  }
  function home(u, me = false) {
    const r = current(u) || outgoing(u), received = incoming(u), first = received[0];
    const profile = `<div class="profile-hero">${person(u, 'ID · ' + (u === 'a' ? '100028' : '100036'), '')}<div class="profile-rank"><div><strong>${people[u].rank}</strong><p style="margin-top:5px"><small>中式八球 · 排位分 ${people[u].score}</small></p></div><span class="rank-ball">✦</span></div></div>`;
    const emptyHero = `<div class="hero"><p>A GOOD GAME AWAITS</p><h3>下一场，<br>和谁开杆？</h3><small>熟悉的球友，新的交锋。</small><div class="table-art" aria-hidden="true"></div></div>`;
    const quick = `<div class="quick-actions">${button(icon('scan') + '扫一扫', 'unavailable', '', '')}${button(icon('qr') + '出示二维码', 'unavailable', '', '')}</div>`;
    const invitesLink = `<button class="link-row" type="button" data-action="inbox"><span class="left">${icon('mail')}收到的邀请${received.length ? `<span class="count">${received.length}</span>` : ''}</span>${icon('chevron')}</button>`;
    return header(u, me ? '我的' : '对局', false, me ? button(icon('gear'), 'settings', '', 'icon-btn', 'aria-label="设置"') : '') + `<div class="scroll">${me ? profile : !r ? emptyHero : `<p class="page-title">今天，约场好球。</p><p class="page-sub">把时间留给球桌，把开局变简单。</p>`}${r ? card(u, r) : ''}${!r ? `${quick}${button(icon('plus') + '发起约球', 'select', '', 'primary new-invite')}` : ''}${invitesLink}${!r && first ? `<div class="section-head">待你回应</div>${card(u, first)}` : ''}${me ? `<div class="metrics"><div><strong>28</strong><small>累计对局</small></div><div><strong>64<span style="font-size:11px">%</span></strong><small>胜率</small></div><div><strong>5</strong><small>最高连胜</small></div></div>` : ''}<div class="section-head"><span>最近交锋</span>${textAction('全部记录 ' + '›', 'history-tab', 'matches')}</div>${recentRows(u)}<button class="link-row" type="button" data-action="history"><span class="left">${icon('history')}约球记录</span>${icon('chevron')}</button>${me ? `<button class="link-row" type="button" data-action="settings"><span class="left">${icon('gear')}设置</span>${icon('chevron')}</button>` : ''}<p class="quiet-tip">每一次交锋，都值得认真记录。</p></div>${tabs(u)}`;
  }
  function selectView(u) {
    const friends = Object.keys(people).filter(p => p !== u && isFriend(u, p));
    const recent = Object.keys(people).filter(p => p !== u && !isFriend(u, p));
    const group = (title, list) => `<div class="section-label">${title}</div><div style="margin-bottom:25px">${list.map(p => `<button class="contact" type="button" data-action="choose" data-id="${p}">${avatar(p)}<div class="player-info"><strong style="font-size:14px">${people[p].name}</strong><p class="field-hint">${people[p].rank} · ID ${p === 'a' ? '100028' : p === 'b' ? '100036' : '100052'}</p></div>${icon('chevron')}</button>`).join('')}</div>`;
    return header(u, '选择球友') + `<div class="scroll"><h3 class="page-title">和谁打一场？</h3><p class="page-sub">老朋友，新交锋。</p>${friends.length ? group('我的好友', friends) : ''}${group('最近对手', recent)}<p class="quiet-tip">好友和最近对手，都可以约球。</p></div>`;
  }
  function composeView(u) {
    const d = views[u].draft || (views[u].draft = defaultDraft(u));
    const choices = (field, options) => `<div class="chips">${options.map(([val, label]) => button(label, 'pick', val, `chip ${String(d[field]) === String(val) ? 'selected' : ''}`, `data-field="${field}" aria-pressed="${String(d[field]) === String(val)}"`)).join('')}</div>`;
    const hours = (field, from, to) => `<label><span class="sr-only">${field === 'start' ? '开始小时' : '结束小时'}</span><select name="${field}" data-field="${field}">${Array.from({ length: to - from + 1 }, (_, i) => i + from).map(h => `<option value="${h}" ${d[field] === h ? 'selected' : ''}>${h} 点${h === 24 ? '（当晚结束）' : ''}</option>`).join('')}</select></label>`;
    return header(u, '发起约球') + `<div class="scroll"><p class="page-title">约好下一场。</p><p class="page-sub">见面时，直接进入就好。</p><div class="form-person">${person(d.to, people[d.to].rank)}${textAction('更换', 'change-person')}</div><fieldset class="form-field"><legend>打什么</legend>${choices('game', ['中式八球', '斯诺克', '美式九球'].map(x => [x, x]))}</fieldset><fieldset class="form-field"><legend>预计什么时候</legend>${choices('day', [0, 1, 2].map((n, i) => [today() + n * DAY, ['今天', '明天', '后天'][i]]))}<div class="hours">${hours('start', 0, 23)}<span>到</span>${hours('end', 1, 24)}</div><p class="field-hint">预计时间仅作提醒，不限制比赛时长</p></fieldset><fieldset class="form-field"><legend>比赛类型</legend>${choices('mode', [['排位', '排位'], ['练习', '练习']])}<p class="field-hint">${d.mode === '排位' ? '公开对局 · 影响排位积分' : '私密练习 · 不影响排位积分'} · 自由局数</p></fieldset><div class="form-field"><label for="note-${u}">附言 <span style="font-weight:400;color:var(--muted)">（选填）</span></label><input id="note-${u}" name="note" maxlength="50" data-field="note" value="${escape(d.note)}" placeholder="比如：老地方，一起打一场？"></div></div><div class="form-bottom">${views[u].error ? `<p class="form-error" role="status">${escape(views[u].error)}</p>` : ''}<p class="field-hint">${relative(d.day + DAY)}早晨7点失效</p>${button('发送约球 ' + icon('arrow'), 'send')}</div>`;
  }
  function inboxView(u) {
    return header(u, '收到的邀请') + `<div class="scroll"><h3 class="page-title">球友的下一场，<br>想和你一起。</h3><p class="page-sub">有空就回应，也可以稍后再看。</p>${incoming(u).map(r => card(u, r)).join('') || `<div class="empty">${icon('mail')}<h3>暂时没有新邀请</h3><p>你也可以主动约一位球友。</p></div>`}</div>${tabs(u)}`;
  }
  function historyView(u) {
    const mode = views[u].historyTab || 'invites';
    return header(u, mode === 'matches' ? '对局记录' : '约球记录') + `<div class="scroll"><div class="segmented">${button('约球记录', 'history-tab', 'invites', mode === 'invites' ? 'active' : '')}${button('对局记录', 'history-tab', 'matches', mode === 'matches' ? 'active' : '')}</div>${mode === 'matches' ? recentRows(u, true) : involved(u).slice().reverse().map(r => card(u, r)).join('')}<p class="quiet-tip">${mode === 'matches' ? '只记录真实开始的比赛' : '每一次约定，都有迹可循'}</p></div>${tabs(u)}`;
  }
  function waitingView(u, r) {
    if (!r || r.status !== 'accepted') { navigate(u, views[u].tab); return home(u, views[u].tab === 'me'); }
    const other = opponent(r, u);
    return header(u, '等待对手') + `<div class="scroll"><div class="waiting-screen"><div class="waiting-kind">${r.game} <span style="color:#c8bea5;margin:0 7px">/</span> ${r.mode}</div><div class="waiting-orbit"><span class="orbit-dot"></span><div class="waiting-player">${avatar(u, 'large')}<strong>${people[u].name}</strong><small class="ready">已进入</small></div><div class="waiting-player">${avatar(other, 'large')}<strong>${people[other].name}</strong><small>待进入</small></div></div><h3>等待${people[other].name}进入</h3><p class="description">好球不怕等。<br>双方进入后，正式开始。</p><div class="wait-time">${icon('clock')}${schedule(r)}</div><div class="wait-bottom"><div class="hint">返回或锁屏后仍会等待<br>暂时不打，可以退出等待</div>${button('退出等待，保留约球', 'leave', r.id, 'secondary')}${textAction('取消本次约球', 'cancel', r.id)}</div></div></div>`;
  }
  function matchView(u, r, playing) {
    if (!r) return historyView(u);
    const finishSlider = `<div class="finish-slide"><span class="finish-caption" aria-hidden="true">向右滑动结束对局 <span>››</span></span><input type="range" class="finish-slider" id="finish-${u}" data-finish-slider="${r.id}" min="0" max="100" step="1" value="0" aria-label="滑动结束对局" aria-describedby="finish-help-${u}"></div><p class="field-hint" id="finish-help-${u}">滑到终点后松手即可结束，无需对方确认。<span class="sr-only">键盘可用方向键或End键移至终点结束，Escape键复位。</span></p>`;
    return header(u, playing ? '对局进行中' : '对局详情') + `<div class="scroll"><div class="match-banner"><p class="result-eyebrow">${playing ? 'GOOD GAME, GOOD COMPANY' : 'EVERY GAME COUNTS'}</p>${tags(r)}<div class="score-players"><div class="waiting-player">${avatar(r.from)}<strong>${people[r.from].name}</strong></div><span style="color:#b6a77d;font-size:11px">VS</span><div class="waiting-player">${avatar(r.to)}<strong>${people[r.to].name}</strong></div></div><div class="score-number">${r.score[0]} : ${r.score[1]}</div><span class="result-seal">${playing ? '进行中' : '对局已结束'}</span></div><div class="detail-rows"><div class="detail-row"><span>对局时间</span><strong>${schedule(r)}</strong></div><div class="detail-row"><span>比赛类型</span><strong>${r.mode} · 自由局数</strong></div><div class="detail-row"><span>约球状态</span><strong>${playing ? '正在进行' : '已完成'}</strong></div></div>${playing ? `<div class="card"><p style="font-size:12px;line-height:1.8">专注每一杆，享受这场交锋。</p><p class="field-hint">演示结果 ${r.score.join(' : ')} · 真实记分与结算沿用现有流程</p></div>` : `<div class="section-head">本场回顾</div><p class="page-sub">一场好球，也是下一次见面的理由。</p>${button('再约一场 ' + icon('arrow'), 'rematch', r.id)}`}<p class="quiet-tip">${playing ? '此页只模拟结束联动，不进行真实结算' : '模拟赛果 · 本次约球已同步完成'}</p></div>${playing ? `<div class="form-bottom finish-control">${finishSlider}</div>` : ''}`;
  }
  function settingsView(u) {
    return header(u, '设置') + `<div class="scroll"><p class="page-title">按你的习惯来。</p><p class="page-sub">留给球桌的，是舒服的相处。</p><p class="section-label">通用设置</p><div class="settings-group"><div class="setting-row"><strong>主题模式</strong><span>亮色 · 本次预览</span></div><div class="setting-row"><strong>通知管理</strong><span>沿用现有设置</span></div></div><div class="section-head">隐私与约球</div><div class="settings-group"><div class="setting-row"><div><strong>隐藏战绩</strong><p>控制是否在公开场景展示你的战绩</p></div><span>现有功能</span></div><div class="setting-row"><div><strong>仅允许好友约球</strong><p>开启后，非好友无法向你<br>发送约球邀请</p></div><button type="button" class="switch" data-action="toggle-friends" role="switch" aria-checked="${prefs[u]}" aria-label="仅允许好友约球"></button></div></div><p class="field-hint" style="padding:9px 3px">仅影响新收到的邀请，已有约球不受影响。</p><div class="settings-group"><div class="setting-row"><strong>隐私政策</strong><span>沿用现有页面</span></div><div class="setting-row"><strong>用户协议</strong><span>沿用现有页面</span></div></div><p class="quiet-tip">${scene === 'settings' ? '当前演示：小李和小王是非好友最近对手' : '演示设置仅保存在当前浏览器，不修改真实账号'}</p></div>`;
  }
  function render() {
    expireRecords();
    document.querySelectorAll('.device-time').forEach(el => el.textContent = clockText());
    const idx = scenes.findIndex(s => s[0] === scene), meta = scenes[idx];
    document.getElementById('scenarios').innerHTML = scenes.map(([id, title], i) => `<button type="button" class="scene-btn ${scene === id ? 'active' : ''}" data-scene="${id}" aria-pressed="${scene === id}"><span class="num">${String(i + 1).padStart(2, '0')}</span>${title}</button>`).join('');
    document.getElementById('scene-number').textContent = String(idx + 1).padStart(2, '0');
    document.getElementById('scene-title').textContent = meta[1]; document.getElementById('scene-note').textContent = meta[2];
    document.getElementById('demo-clock').value = new Date(now + 8 * 3600000).toISOString().slice(0, 19) + '+08:00';
    document.getElementById('simulation-status').textContent = `模拟比赛 ${invites.filter(r => r.matchId && r.id !== 1).length} 场 · A/B ${abFriends ? '好友' : '最近对手'}`;
    for (const u of ['a', 'b']) {
      const root = document.getElementById(`view-${u}`), oldPage = root.dataset.page;
      const scroll = root.querySelector('.scroll')?.scrollTop || 0;
      const focused = root.contains(document.activeElement) ? document.activeElement : null;
      const focusKey = focused?.name || focused?.id;
      const focusButton = focused?.dataset.action ? `[data-action="${focused.dataset.action}"][data-id="${focused.dataset.id || ''}"]${focused.dataset.field ? `[data-field="${focused.dataset.field}"]` : ''}` : null;
      const { page } = views[u], r = get(views[u].recordId);
      let html;
      if (page === 'home' || page === 'me') html = home(u, page === 'me');
      else if (page === 'select') html = selectView(u);
      else if (page === 'compose') html = composeView(u);
      else if (page === 'inbox') html = inboxView(u);
      else if (page === 'history') html = historyView(u);
      else if (page === 'waiting') html = waitingView(u, r);
      else if (page === 'playing' || page === 'detail') html = matchView(u, r, page === 'playing' && r?.status === 'playing');
      else html = settingsView(u);
      const d = views[u].dialog;
      if (messages[u]) html += `<div class="toast" role="status">${escape(messages[u])}</div>`;
      if (d) html += `<div class="overlay"><section class="modal" role="dialog" aria-modal="true" aria-labelledby="dialog-title-${u}"><div class="modal-symbol">${icon(d.danger ? 'info' : 'ball')}</div><h3 id="dialog-title-${u}">${escape(d.title)}</h3><p>${escape(d.text)}</p><div class="action-row">${d.action ? button(d.cancel, 'dismiss', '', 'secondary') : ''}${button(d.confirm, 'confirm', '', `primary ${d.danger ? 'danger' : ''}`)}</div></section></div>`;
      root.innerHTML = html; root.dataset.page = views[u].page;
      if (oldPage === views[u].page && root.querySelector('.scroll')) root.querySelector('.scroll').scrollTop = scroll;
      if (focusKey) root.querySelector(`[name="${focusKey}"], [id="${focusKey}"]`)?.focus({ preventScroll: true });
      else if (focusButton && oldPage === views[u].page) root.querySelector(focusButton)?.focus({ preventScroll: true });
      if (d && !root.querySelector('.modal').contains(document.activeElement)) root.querySelector('.modal button')?.focus({ preventScroll: true });
    }
  }
  document.addEventListener('click', e => {
    const sceneBtn = e.target.closest('[data-scene]'); if (sceneBtn) { loadScene(sceneBtn.dataset.scene); return; }
    const role = e.target.closest('[data-role]'); if (role) { document.querySelector('.stage').dataset.active = role.dataset.role; document.querySelectorAll('[data-role]').forEach(b => b.setAttribute('aria-pressed', String(b === role))); return; }
    const target = e.target.closest('[data-action]'), device = target?.closest('[data-user]'); if (!device) return;
    const u = device.dataset.user, v = views[u], { action, id } = target.dataset, r = get(id);
    if (action === 'nav') navigate(u, id);
    else if (action === 'back') navigate(u, v.tab);
    else if (action === 'inbox' || action === 'history' || action === 'settings') navigate(u, action);
    else if (action === 'history-tab') { v.historyTab = id; navigate(u, 'history'); }
    else if (action === 'select') { if (!blocker(u)) navigate(u, 'select'); }
    else if (action === 'change-person') { v.changingPerson = true; navigate(u, 'select'); }
    else if (action === 'choose') {
      if (v.changingPerson && v.draft) { v.draft.to = id; v.changingPerson = false; v.error = ''; navigate(u, 'compose'); }
      else compose(u, id);
    }
    else if (action === 'pick') { v.draft[target.dataset.field] = target.dataset.field === 'day' ? Number(id) : id; v.error = ''; }
    else if (action === 'send') send(u);
    else if (action === 'accept') accept(u, id);
    else if (action === 'reject') endInvite(u, id, 'rejected');
    else if (action === 'enter') enter(u, id);
    else if (action === 'leave') leave(u, id);
    else if (action === 'detail') navigate(u, 'detail', id);
    else if (action === 'rematch' && r) compose(u, opponent(r, u), r);
    else if (action === 'cancel' && r) modal(u, '取消本次约球？', `${people[opponent(r, u)].name}会收到提醒，本次约球将结束。\n不会产生比赛和战绩。`, '确认取消', () => endInvite(u, id, 'canceled'), '继续约球', true);
    else if (action === 'dismiss') v.dialog = null;
    else if (action === 'confirm') { const fn = v.dialog?.action; v.dialog = null; fn?.(); }
    else if (action === 'unavailable') toast(u, '扫码属于现有功能，本预览只演示约球');
    else if (action === 'toggle-friends') { prefs[u] = !prefs[u]; try { localStorage.setItem('cp-quick-preview-preferences', JSON.stringify(prefs)); } catch (_) { /* 仅模拟存储。 */ } toast(u, prefs[u] ? '已开启，仅允许好友约球' : '已关闭好友限制'); }
    render();
    if ((action === 'dismiss' || action === 'confirm') && !v.dialog) document.getElementById(`view-${u}`).querySelector('button')?.focus({ preventScroll: true });
  });
  function resetFinishSlider(slider) {
    slider.value = '0'; slider.dataset.inputKind = '';
    slider.closest('.finish-slide').style.setProperty('--progress', '0%');
  }
  document.addEventListener('pointerdown', e => {
    const slider = e.target.closest('[data-finish-slider]'); if (!slider) return;
    const x = e.clientX - slider.getBoundingClientRect().left;
    // 原生range保留拖动/键盘能力；轨道点击不能直接跳到终点结束。
    if (e.button !== 0 || e.isPrimary === false || x < 0 || x > 46) {
      e.preventDefault(); resetFinishSlider(slider); return;
    }
    slider.dataset.inputKind = 'pointer';
  });
  document.addEventListener('pointercancel', e => {
    if (e.target.matches('[data-finish-slider]')) resetFinishSlider(e.target);
  });
  document.addEventListener('focusout', e => {
    if (e.target.matches('[data-finish-slider]')) resetFinishSlider(e.target);
  });
  document.addEventListener('change', e => {
    const slider = e.target.closest('[data-finish-slider]');
    if (slider) {
      if (Number(slider.value) === 100 && ['pointer', 'keyboard'].includes(slider.dataset.inputKind)) {
        finish(slider.closest('[data-user]').dataset.user, slider.dataset.finishSlider); render();
      } else if (slider.dataset.inputKind !== 'keyboard') resetFinishSlider(slider);
      return;
    }
    if (e.target.id === 'demo-clock') { setDemoTime(e.target.value); return; }
    const u = e.target.closest('[data-user]')?.dataset.user, field = e.target.dataset.field;
    if (u && field && views[u].draft) { views[u].draft[field] = field === 'note' ? e.target.value : Number(e.target.value); views[u].error = ''; render(); }
  });
  document.addEventListener('input', e => {
    if (e.target.matches('[data-finish-slider]')) { e.target.closest('.finish-slide').style.setProperty('--progress', `${e.target.value}%`); return; }
    const u = e.target.closest('[data-user]')?.dataset.user;
    if (u && e.target.dataset.field === 'note') views[u].draft.note = e.target.value;
  });
  document.addEventListener('keydown', e => {
    if (e.target.matches('[data-finish-slider]')) {
      if (e.key === 'Escape') { e.preventDefault(); resetFinishSlider(e.target); }
      else if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(e.key)) e.target.dataset.inputKind = 'keyboard';
      return;
    }
    const u = e.target.closest('[data-user]')?.dataset.user; if (!u || !views[u].dialog) return;
    if (e.key === 'Escape') { views[u].dialog = null; render(); document.querySelector(`#view-${u} button`)?.focus(); }
    if (e.key === 'Tab') { const buttons = [...document.querySelectorAll(`#view-${u} .modal button`)]; const i = buttons.indexOf(document.activeElement); e.preventDefault(); buttons[(i + (e.shiftKey ? -1 : 1) + buttons.length) % buttons.length].focus(); }
  });
  document.getElementById('reset-scene').addEventListener('click', () => loadScene(scene));
  if (window.matchMedia?.('(max-width: 600px)').matches) document.querySelector('.controls').open = false;
  // 原型使用固定时钟与内存数据；真实后台校验、网络竞争和结算由应用实现负责。
  window.Preview = { loadScene, render, send, accept, enter, leave, endInvite, finish, compose,
    setTime: setDemoTime,
    state: () => ({ invites, views, prefs, now }), schedule, expiryText };
  loadScene('accepted');
})();
