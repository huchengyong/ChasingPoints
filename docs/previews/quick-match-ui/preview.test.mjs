import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

// 复用仓库已有 jsdom；预览页面本身不依赖它。
const require = createRequire(new URL('../../../uniapp/package.json', import.meta.url));
const { JSDOM } = require('jsdom');
const html = readFileSync(new URL('./index.html', import.meta.url), 'utf8');
const script = readFileSync(new URL('./preview.js', import.meta.url), 'utf8');
function setup(t, scene) {
  const dom = new JSDOM(html, { url: 'http://preview.local/', runScripts: 'outside-only' });
  dom.window.eval(script);
  t.after(() => dom.window.close());
  const p = dom.window.Preview;
  p.loadScene(scene);
  return { p, doc: dom.window.document, window: dom.window, click: selector => { const el = dom.window.document.querySelector(selector); assert.ok(el, selector); el.click(); } };
}
const live = p => p.state().invites.filter(r => !['completed', 'merged', 'canceled', 'rejected', 'expired'].includes(r.status));

test('双方接受不建比赛；共同进入唯一开局，任一方结束同步释放约球', t => {
  const { p, doc, click } = setup(t, 'pending');
  click('#view-b [data-action="accept"]');
  const r = live(p)[0];
  assert.equal(r.status, 'accepted'); assert.equal(r.matchId, null);
  click('#view-a [data-action="enter"]');
  assert.equal(r.matchId, null);
  assert.match(doc.querySelector('#view-a').textContent, /等待小王进入/);
  assert.equal(doc.querySelector('#view-a .score-number'), null);
  click('#view-b [data-action="enter"]');
  const id = r.matchId;
  p.enter('a', r.id); p.enter('b', r.id);
  assert.equal(r.matchId, id); assert.equal(live(p).length, 1);
  p.finish('c', r.id); assert.equal(r.status, 'playing');
  p.finish('a', r.id); assert.equal(r.status, 'playing'); assert.equal(r.finishRequest.by, 'a');
  p.confirmFinish('b', r.id); assert.equal(r.status, 'completed');
  assert.equal(p.state().views.b.page, 'detail');
  p.finish('b', r.id); assert.equal(live(p).length, 0);
  p.compose('a', 'b', r); assert.equal(p.state().views.a.page, 'compose');
});

function gestures(doc, window, selector) {
  const slider = doc.querySelector(selector);
  assert.ok(slider, selector);
  return {
    slider,
    start: x => slider.dispatchEvent(new window.MouseEvent('pointerdown', { bubbles: true, cancelable: true, button: 0, clientX: x })),
    move: value => { slider.value = String(value); slider.dispatchEvent(new window.Event('input', { bubbles: true })); },
    release: () => slider.dispatchEvent(new window.Event('change', { bubbles: true })),
    cancel: () => slider.dispatchEvent(new window.Event('pointercancel', { bubbles: true }))
  };
}

test('结束与确认滑块：误点、半途松手和触摸取消都不生效，滑到底松手才发起或确认', t => {
  for (const user of ['a', 'b']) {
    const other = user === 'a' ? 'b' : 'a';
    const { p, doc, window } = setup(t, 'playing');
    const r = live(p)[0];
    for (const [selector, done] of [
      [`#view-${user} [data-slider-kind="finish"]`, () => r.finishRequest?.by === user],
      [`#view-${other} .finish-confirm [data-slider-kind="confirm"]`, () => r.status === 'completed']
    ]) {
      const g = gestures(doc, window, selector);
      assert.equal(g.start(200), false); g.move(100); g.release();
      assert.ok(!done()); assert.equal(g.slider.value, '0');
      g.start(10); g.move(60); g.release();
      assert.ok(!done()); assert.equal(g.slider.value, '0');
      g.start(10); g.move(100);
      assert.ok(!done(), '到终点但未松手不能生效');
      g.cancel(); g.release();
      assert.ok(!done()); assert.equal(g.slider.value, '0');
      g.start(10); g.move(100); g.release();
      assert.ok(done());
    }
    assert.equal(r.endWay, 'confirmed');
    assert.equal(p.state().views.a.page, 'detail'); assert.equal(p.state().views.b.page, 'detail');
  }
});

test('结束与确认滑块可用键盘操作，Escape复位，End完成', t => {
  const { p, doc, window } = setup(t, 'playing');
  const r = live(p)[0];
  const keyboard = selector => {
    const slider = doc.querySelector(selector);
    return { slider, key: name => slider.dispatchEvent(new window.KeyboardEvent('keydown', { key: name, bubbles: true })), set: value => { slider.value = String(value); slider.dispatchEvent(new window.Event('change', { bubbles: true })); } };
  };
  const finish = keyboard('#view-b [data-slider-kind="finish"]');
  finish.key('ArrowRight'); finish.set(40);
  assert.equal(r.finishRequest, undefined);
  finish.key('Escape'); assert.equal(finish.slider.value, '0');
  finish.key('End'); finish.set(100);
  assert.equal(r.status, 'playing'); assert.equal(r.finishRequest.by, 'b');
  const confirm = keyboard('#view-a [data-slider-kind="confirm"]');
  confirm.key('End'); confirm.set(100);
  assert.equal(live(p).length, 0); assert.equal(p.state().views.a.page, 'detail');
});

test('普通返回继续等待，退出等待保留约球，放弃让等待方退出且无比赛', t => {
  const { p, click } = setup(t, 'waiting');
  const r = live(p)[0];
  click('#view-a [data-action="back"]');
  assert.ok(r.entered.includes('a'));
  click('#view-a [data-action="enter"]');
  click('#view-a [data-action="leave"]');
  assert.equal(r.entered.length, 0); assert.equal(r.status, 'accepted');
  p.enter('a', r.id); p.render();
  click('#view-b [data-action="cancel"]');
  click('#view-b [data-action="confirm"]');
  assert.equal(r.status, 'canceled'); assert.equal(r.matchId, null);
  assert.equal(p.state().views.a.page, 'home');
});

test('互邀按类型匹配，时间可不同，重复接受不重复记录', t => {
  const { p } = setup(t, 'mutual');
  const incoming = p.state().invites.find(r => r.from === 'b' && r.status === 'pending');
  p.accept('a', incoming.id); p.accept('a', incoming.id);
  assert.equal(live(p).length, 1); assert.equal(live(p)[0].start, 18);
  assert.equal(p.state().invites.filter(r => r.status === 'merged').length, 1);
  p.loadScene('different');
  p.accept('a', p.state().invites.find(r => r.from === 'b' && r.status === 'pending').id);
  assert.equal(live(p).length, 2); assert.ok(live(p).every(r => r.status === 'pending'));
  assert.match(p.state().views.a.dialog.title, /类型不同/);
});

test('换人接受有提醒，取消不改变旧邀请；旧邀请先被接受时不覆盖', t => {
  const { p, click } = setup(t, 'replace');
  const old = p.state().invites.find(r => r.from === 'a' && r.status === 'pending');
  const next = p.state().invites.find(r => r.from === 'c');
  p.accept('a', next.id);
  assert.match(p.state().views.a.dialog.text, /自动关闭/);
  click('#view-a [data-action="dismiss"]'); assert.equal(old.status, 'pending');
  p.accept('a', next.id); p.accept('b', old.id);
  p.accept('a', next.id, true);
  assert.equal(old.status, 'accepted'); assert.equal(next.status, 'pending');
  p.loadScene('replace');
  const n = p.state().invites.find(r => r.from === 'c');
  p.accept('a', n.id); click('#view-a [data-action="confirm"]');
  assert.equal(n.status, 'accepted'); assert.equal(p.state().invites.find(r => r.id === 2).status, 'canceled');
});

test('收到不阻止发送，自己待回应阻止重复发送', t => {
  const { p } = setup(t, 'pending');
  p.compose('b', 'c'); p.send('b');
  assert.equal(p.state().invites.filter(r => r.status === 'pending').length, 2);
  p.send('b'); assert.equal(p.state().invites.filter(r => r.from === 'b').length, 1);
});

test('提前入局只提示第一个人，对方正在等待时直接进入；退出等待后重新判断', t => {
  for (const first of ['a', 'b']) {
    const second = first === 'a' ? 'b' : 'a';
    const { p, doc, click } = setup(t, 'accepted');
    p.setTime('2026-08-25T15:00:00+08:00');
    const r = live(p)[0];
    click(`#view-${first} [data-action="enter"]`);
    assert.ok(doc.querySelector(`#view-${first} [role="dialog"]`));
    assert.equal(r.entered.length, 0); assert.equal(r.matchId, null);
    click(`#view-${first} [data-action="confirm"]`);
    assert.ok(r.entered.includes(first)); assert.equal(r.matchId, null);
    click(`#view-${second} [data-action="enter"]`);
    assert.equal(doc.querySelector(`#view-${second} [role="dialog"]`), null);
    assert.equal(r.status, 'playing'); assert.ok(r.matchId);
    assert.equal(p.state().views.a.page, 'playing'); assert.equal(p.state().views.b.page, 'playing');

    p.loadScene('waiting'); p.setTime('2026-08-25T15:00:00+08:00');
    const waiting = live(p)[0];
    click('#view-a [data-action="leave"]');
    click('#view-b [data-action="enter"]');
    assert.ok(doc.querySelector('#view-b [role="dialog"]'));
    assert.equal(waiting.matchId, null); assert.equal(waiting.entered.length, 0);
  }
});

test('午夜日期、提前提示与07:00失效；比赛不随约球截止而停止', t => {
  const { p, doc, click } = setup(t, 'accepted');
  const r = live(p)[0];
  p.setTime('2026-08-25T15:00:00+08:00'); p.enter('a', r.id);
  assert.match(p.state().views.a.dialog.title, /现在进入/); assert.equal(r.entered.length, 0);
  click('#view-a [data-action="dismiss"]');
  p.setTime('2026-08-26T00:30:00+08:00');
  assert.match(p.schedule(r), /昨天/); assert.equal(p.expiryText(r), '今天早晨7点失效');
  p.setTime('2026-08-26T07:00:00+08:00'); p.enter('a', r.id);
  assert.equal(r.status, 'expired'); assert.equal(r.matchId, null);
  assert.doesNotMatch(doc.body.textContent, /已于.*失效/);
  p.loadScene('playing'); p.setTime('2026-08-26T07:00:00+08:00');
  assert.equal(live(p)[0].status, 'playing');
});

test('仅好友开关默认关闭，按接收方拦截新邀请且不回收已发送邀请', t => {
  const { p, click } = setup(t, 'settings');
  assert.equal(p.state().prefs.b, false);
  click('#view-b [data-action="toggle-friends"]'); p.send('a');
  assert.equal(p.state().invites.length, 1);
  assert.match(p.state().views.a.error, /仅允许好友/);
  click('#view-b [data-action="toggle-friends"]'); p.send('a');
  assert.equal(p.state().invites.length, 2);
  click('#view-b [data-action="toggle-friends"]');
  assert.equal(live(p)[0].status, 'pending');
});

test('再约保留对手类型、更新日期，不直接发出；表单文本安全显示', t => {
  const { p, doc, click } = setup(t, 'history');
  click('#view-a [data-action="rematch"]');
  const d = p.state().views.a.draft;
  assert.equal(d.to, 'b'); assert.equal(d.mode, '排位'); assert.equal(d.day, Date.parse('2026-08-25T00:00:00+08:00'));
  assert.equal(p.state().invites.length, 1);
  d.note = '<img src=x onerror=alert(1)>'; p.send('a'); p.render();
  assert.equal(doc.querySelector('#view-a .message-box img'), null);
  assert.match(doc.querySelector('#view-a .message-box').textContent, /<img/);
});

test('收到待回应邀请时收起头图、发起降为次要，扫码仍在前；无邀请时保持原样', t => {
  const { p, doc } = setup(t, 'pending');
  const b = doc.querySelector('#view-b');
  assert.equal(b.querySelector('.hero'), null);
  assert.ok(b.querySelector('.new-invite').classList.contains('secondary'));
  assert.ok(b.querySelector('[data-action="accept"]').classList.contains('primary'));
  assert.ok(b.querySelector('.quick-actions').compareDocumentPosition(b.querySelector('[data-action="accept"]')) & 4, '扫码入口仍在待回应卡之前');
  assert.equal(b.querySelector('.link-row .count').textContent, '1');
  p.loadScene('fresh');
  assert.ok(doc.querySelector('#view-b .hero'));
  assert.ok(doc.querySelector('#view-b .new-invite').classList.contains('primary'));
});

test('比分与胜负按查看者展示', t => {
  const { p, doc, click } = setup(t, 'fresh');
  assert.match(doc.querySelector('#view-a .record-score').textContent, /胜\s*5 : 3/);
  assert.match(doc.querySelector('#view-b .record-score').textContent, /负\s*3 : 5/);
  click('#view-b [data-action="detail"]');
  const b = doc.querySelector('#view-b');
  assert.equal(b.querySelector('.score-number').textContent, '3 : 5');
  assert.equal(b.querySelector('.score-players strong').textContent, '小王');
  assert.ok(b.querySelector('.result-seal').classList.contains('loss'));
  assert.match(b.querySelector('.result-seal').textContent, /失利/);
  p.loadScene('playing');
  assert.equal(doc.querySelector('#view-a .score-number').textContent, '5 : 3');
  assert.equal(doc.querySelector('#view-b .score-number').textContent, '3 : 5');
  assert.equal(doc.querySelector('#view-b .result-seal').textContent, '进行中');
});

test('底部只有对局与我的；收到的邀请、记录为带返回的二级页，我的页不重复对局页入口', t => {
  const { p, doc, click } = setup(t, 'accepted');
  const a = () => doc.querySelector('#view-a');
  assert.deepEqual([...a().querySelectorAll('.tabs button')].map(el => el.textContent), ['对局', '我的']);
  assert.equal(a().querySelectorAll('[data-action="inbox"]').length, 1);
  click('#view-a [data-action="inbox"]');
  assert.equal(a().querySelector('.tabs'), null); assert.ok(a().querySelector('[data-action="back"]'));
  click('#view-a [data-action="back"]'); assert.equal(p.state().views.a.page, 'home');
  click('#view-a .section-head [data-action="history-tab"]');
  assert.equal(p.state().views.a.historyTab, 'matches'); assert.equal(a().querySelector('.tabs'), null);
  click('#view-a [data-action="back"]');
  click('#view-a .link-row[data-action="history-tab"]');
  assert.equal(p.state().views.a.historyTab, 'invites');
  const b = doc.querySelector('#view-b');
  assert.equal(p.state().views.b.page, 'me');
  assert.ok(b.querySelector('[data-action="enter"]'));
  assert.equal(b.querySelector('.record-row, .link-row, .quick-actions'), null);
});

test('排位结束说明写在滑块旁：对方30秒内确认，期间可撤销，超时按当前比分结束', t => {
  const { p, doc } = setup(t, 'playing');
  assert.match(doc.querySelector('#view-a .finish-note').textContent, /松手后需小王在30秒内确认，期间你可撤销；无人处理则按 5 : 3 自动结束，之后不能再提出异议/);
  assert.match(doc.querySelector('#view-b .finish-note').textContent, /按 3 : 5 自动结束/);
  live(p)[0].mode = '练习'; p.render();
  assert.match(doc.querySelector('#view-a .finish-note').textContent, /^滑到终点后松手即可结束，无需对方确认。/);
  assert.doesNotMatch(doc.querySelector('#view-a .finish-note').textContent, /30秒|撤销|异议/);
});

test('排位发起后比分冻结：发起方看到倒计时和撤销，对方滑动确认立即结束', t => {
  const { p, doc, click } = setup(t, 'playing');
  const r = live(p)[0];
  const a = () => doc.querySelector('#view-a'), b = () => doc.querySelector('#view-b');
  click('#view-b [data-action="back"]');
  p.finish('a', r.id); p.render();
  assert.equal(r.status, 'playing');
  assert.match(a().querySelector('.finish-wait').textContent, /等待小王确认 · 30 秒/);
  assert.match(a().querySelector('.finish-wait').textContent, /按 5 : 3 自动结束，之后不能再提出异议/);
  assert.ok(a().querySelector('[data-action="undo-finish"]'));
  assert.equal(a().querySelector('[data-slider-kind="finish"]'), null);
  assert.equal(a().querySelector('.finish-confirm'), null, '发起方不弹确认框');
  assert.match(a().querySelector('.result-seal').textContent, /结束确认中/);
  assert.match(b().querySelector('.toast').textContent, /小李申请结束对局，请在30秒内处理/, '不在对局页时提示');
  assert.match(b().querySelector('.card .note').textContent, /3 : 5 · 结束确认中/);
  click('#view-b [data-action="enter"]');
  const dialog = b().querySelector('.finish-confirm');
  assert.match(dialog.textContent, /小李申请结束对局/); assert.match(dialog.textContent, /比分 3 : 5。30 秒内未处理/);
  p.finish('b', r.id); assert.equal(r.finishRequest.by, 'a', '确认期间不能重复发起');
  p.confirmFinish('a', r.id); assert.equal(r.status, 'playing', '发起方不能自己确认');
  p.confirmFinish('c', r.id); assert.equal(r.status, 'playing', '非参赛者不能确认');
  p.confirmFinish('b', r.id); p.render();
  assert.equal(r.status, 'completed'); assert.equal(r.endWay, 'confirmed');
  assert.match(a().textContent, /你发起 · 小王确认/); assert.match(b().textContent, /小李发起 · 你确认/);
  assert.match(a().querySelector('.toast').textContent, /小王已确认，对局结束/);
  assert.doesNotMatch(b().textContent, /异议/);
  p.compose('a', 'b', r); assert.equal(p.state().views.a.page, 'compose');
});

test('发起方撤销或对方有异议都恢复对局，比分不变且可再次发起', t => {
  const { p, doc, click } = setup(t, 'playing');
  const r = live(p)[0];
  p.finish('a', r.id); p.render();
  assert.equal(doc.querySelector('#view-b .toast'), null, '对方正在看本场时由弹窗告知，不叠加提示');
  click('#view-a [data-action="undo-finish"]');
  assert.equal(r.status, 'playing'); assert.equal(r.finishRequest, undefined); assert.equal(r.score.join(':'), '5:3');
  assert.match(doc.querySelector('#view-b .toast').textContent, /小李撤销了结束，对局继续/);
  assert.ok(doc.querySelector('#view-a [data-slider-kind="finish"]'));
  assert.equal(doc.querySelector('#view-b .finish-confirm'), null);
  p.finish('b', r.id); p.render();
  click('#view-a .finish-confirm [data-action="undo-finish"]');
  assert.equal(r.status, 'playing'); assert.equal(r.finishRequest, undefined); assert.equal(r.score.join(':'), '5:3');
  assert.match(doc.querySelector('#view-a .toast').textContent, /已提出异议，对局继续/);
  assert.match(doc.querySelector('#view-b .toast').textContent, /小李有异议，对局继续/);
  p.undoFinish('a', r.id); assert.equal(r.status, 'playing');
  p.finish('a', r.id); assert.equal(r.finishRequest.by, 'a', '恢复后可再次发起');
});

test('30秒无人处理按冻结比分自动结束，之后不能再撤销或异议；练习赛滑动即结束', t => {
  const { p, doc } = setup(t, 'playing');
  const r = live(p)[0];
  p.finish('b', r.id); p.render();
  for (let i = 0; i < 29; i++) p.tick();
  assert.equal(r.status, 'playing');
  assert.equal(doc.querySelector('#view-b .finish-wait .countdown').textContent, '1');
  assert.equal(doc.querySelector('#view-a .finish-confirm .countdown').textContent, '1');
  p.tick();
  assert.equal(r.status, 'completed'); assert.equal(r.endWay, 'timeout');
  assert.match(doc.querySelector('#view-a').textContent, /小王发起 · 30秒未处理，自动结束/);
  assert.match(doc.querySelector('#view-a .toast').textContent, /30秒未处理，对局已按 5 : 3 结束/);
  assert.match(doc.querySelector('#view-b .toast').textContent, /30秒未处理，对局已按 3 : 5 结束/);
  p.undoFinish('a', r.id); p.confirmFinish('a', r.id);
  assert.equal(r.status, 'completed'); assert.equal(r.endWay, 'timeout');
  assert.equal(doc.querySelector('#view-a [data-action="undo-finish"]'), null);
  p.loadScene('playing');
  const practice = live(p)[0]; practice.mode = '练习';
  p.finish('b', practice.id); p.render();
  assert.equal(practice.status, 'completed'); assert.equal(practice.endWay, 'direct'); assert.equal(practice.finishRequest, undefined);
  assert.match(doc.querySelector('#view-a .toast').textContent, /^小王结束了对局 · 5 : 3$/);
  assert.doesNotMatch(doc.querySelector('#view-a').textContent, /结束方式/, '练习的手动与达标结束无法区分，赛果页不标注');
});
