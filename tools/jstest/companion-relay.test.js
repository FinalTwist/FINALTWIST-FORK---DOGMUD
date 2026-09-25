// companion-relay.test.js: guards for the companion key relay
// (modules/aicompanion/relayweb/relay.js), the only place a player's own
// model key ever exists: the framed relay, and the key window it opens.
//
// Run:  node tools/jstest/companion-relay.test.js
//
// Dependency-free and assertion-library-free, matching hotinput.test.js.
// Needs node 20 or later for globalThis.crypto.subtle. Exits non-zero on
// failure.

var path = require('path');
var fs = require('fs');

var WEB_DIR = path.join(__dirname, '..', '..', 'modules', 'aicompanion', 'relayweb');
var Relay = require(path.join(WEB_DIR, 'relay.js'));

var failures = 0;
var checks = 0;

function check(name, got, want) {
    checks++;
    if (got !== want) {
        console.log('FAIL ' + name + '\n  got:  ' + String(got).slice(0, 200) + '\n  want: ' + String(want).slice(0, 200));
        failures++;
        return;
    }
    console.log('ok   ' + name);
}

var KEY = 'sk-test-THISISTHEPLAYERSKEY1234567890';
var GAME = 'https://example.org';
var RELAY = 'https://keys.example.org';
var STORED = { endpoint: 'https://api.openai.com/v1', key: KEY, model: 'gpt-4.1-mini' };

function fakeFetch(reply) {
    var calls = [];
    var fn = function (url, init) {
        calls.push({ url: url, init: init });
        if (reply instanceof Error) { return Promise.reject(reply); }
        return Promise.resolve(reply);
    };
    fn.calls = calls;
    return fn;
}

function textReply(status, text, headers) {
    return {
        status: status,
        headers: { get: function (h) { return (headers || {})[h.toLowerCase()] || null; } },
        text: function () { return Promise.resolve(text); }
    };
}

function streamReply(status, chunks) {
    var i = 0;
    var r = {
        status: status,
        cancelled: false,
        headers: { get: function () { return null; } },
        body: {
            getReader: function () {
                return {
                    read: function () {
                        if (i >= chunks.length) { return Promise.resolve({ done: true }); }
                        return Promise.resolve({ done: false, value: new TextEncoder().encode(chunks[i++]) });
                    },
                    cancel: function () { r.cancelled = true; return Promise.resolve(); }
                };
            }
        },
        text: function () { throw new Error('the stream path must be used'); }
    };
    return r;
}

function sortedKeys(o) { return Object.keys(o).sort().join(','); }

function memoryStorage() {
    var m = {};
    return {
        data: m,
        get: function (k) { return Object.prototype.hasOwnProperty.call(m, k) ? m[k] : null; },
        set: function (k, v) { m[k] = v; return true; },
        remove: function (k) { delete m[k]; }
    };
}

function tick() { return new Promise(function (r) { setTimeout(r, 0); }); }
async function until(fn) { for (var i = 0; i < 4000 && !fn(); i++) { await tick(); } }

// leaks says whether a message carries the key, an endpoint, a header or a
// passphrase: nothing posted to the game page ever may.
function leaks(m) {
    var s = JSON.stringify(m);
    return s.indexOf(KEY) !== -1 || /endpoint|authorization|"key"|pass/i.test(s);
}

async function main() {
    // --- endpoint rules ------------------------------------------------------
    var ep = Relay.isAllowedEndpoint;
    check('https endpoint allowed', ep('https://api.openai.com/v1'), true);
    check('http on localhost allowed', ep('http://localhost:11434/v1'), true);
    check('http on 127.0.0.1 allowed', ep('http://127.0.0.1:1234/v1'), true);
    check('http elsewhere refused', ep('http://evil.example/v1'), false);
    check('javascript: refused', ep('javascript:alert(1)'), false);
    check('file: refused', ep('file:///x'), false);
    check('ftp refused', ep('ftp://example.org/v1'), false);
    check('user info refused', ep('https://user:pass@api.openai.com/v1'), false);
    check('user name alone refused', ep('https://user@api.openai.com/v1'), false);
    check('query refused', ep('https://api.openai.com/v1?x=1'), false);
    check('fragment refused', ep('https://api.openai.com/v1#x'), false);
    check('empty refused', ep(''), false);
    check('non-string refused', ep(null), false);
    check('ITER is 600000', Relay.ITER, 600000);

    // --- relayOne posts only to the stored endpoint ---------------------------
    var f = fakeFetch(textReply(200, '{"ok":true}'));
    var got = await Relay.relayOne(f, STORED, {
        id: 'abc', body: { model: 'x' },
        url: 'https://evil.example/steal', endpoint: 'https://evil.example', headers: { Authorization: 'x' }
    });
    check('fetch called once', f.calls.length, 1);
    check('URL is the STORED endpoint plus /chat/completions', f.calls[0].url, 'https://api.openai.com/v1/chat/completions');
    var init = f.calls[0].init;
    check('Authorization carries the stored key', init.headers.Authorization, 'Bearer ' + KEY);
    check('header names are exactly the two we set', sortedKeys(init.headers), 'Authorization,Content-Type');
    check('credentials omit', init.credentials, 'omit');
    check('referrerPolicy no-referrer', init.referrerPolicy, 'no-referrer');
    check('redirect is error', init.redirect, 'error');
    check('mode cors', init.mode, 'cors');
    check('cache no-store', init.cache, 'no-store');
    check('method POST', init.method, 'POST');
    check('an object body is sent as JSON', init.body, '{"model":"x"}');
    check('returned keys are exactly id,status,body', sortedKeys(got), 'body,id,status');
    check('returned body is the reply', got.body, '{"ok":true}');
    check('returned status is the reply status', got.status, 200);

    var f2 = fakeFetch(textReply(200, 'x'));
    await Relay.relayOne(f2, { endpoint: 'https://openrouter.ai/api/v1/', key: KEY, model: 'm' }, { id: 'a', body: '{}' });
    check('trailing slashes collapse', f2.calls[0].url, 'https://openrouter.ai/api/v1/chat/completions');

    var f3 = fakeFetch(textReply(200, 'x'));
    var none = await Relay.relayOne(f3, null, { id: 'abc', body: '{}' });
    check('no settings: status 0', none.status, 0);
    check('no settings: empty body', none.body, '');
    check('no settings: fetch never called', f3.calls.length, 0);

    var f3b = fakeFetch(textReply(200, 'x'));
    await Relay.relayOne(f3b, { endpoint: 'http://evil.example/v1', key: KEY, model: 'm' }, { id: 'a', body: '{}' });
    check('a stored endpoint that is not allowed is never called', f3b.calls.length, 0);

    // --- replies too big for the game page never leave the relay ------------
    check('the reply cap is 60 KiB', Relay.MAX_REPLY_BYTES, 60 * 1024);
    var big = new Array(60 * 1024 + 2).join('a');
    var over = await Relay.relayOne(fakeFetch(textReply(200, big)), STORED, { id: 'a', body: '{}' });
    check('oversized text reply: status 0', over.status, 0);
    check('oversized text reply: empty body', over.body, '');
    var atCap = new Array(60 * 1024 + 1).join('a');
    var fits = await Relay.relayOne(fakeFetch(textReply(200, atCap)), STORED, { id: 'a', body: '{}' });
    check('a reply exactly at the cap passes', fits.body.length, 60 * 1024);
    var s = streamReply(200, [big.slice(0, 40000), big.slice(40000)]);
    var overStream = await Relay.relayOne(fakeFetch(s), STORED, { id: 'a', body: '{}' });
    check('oversized streamed reply: status 0', overStream.status, 0);
    check('oversized streamed reply: stream cancelled', s.cancelled, true);
    var sOk = await Relay.relayOne(fakeFetch(streamReply(200, ['{"a":', '1}'])), STORED, { id: 'a', body: '{}' });
    check('a streamed reply is joined', sOk.body, '{"a":1}');
    var declared = await Relay.relayOne(fakeFetch(textReply(200, 'x', { 'content-length': String(60 * 1024 + 1) })),
        STORED, { id: 'a', body: '{}' });
    check('a declared oversize reply: status 0', declared.status, 0);

    // --- error text and echoes never carry the key back ----------------------
    var err = await Relay.relayOne(fakeFetch(textReply(401, 'Incorrect API key provided: sk-test-****7890')),
        STORED, { id: 'a', body: '{}' });
    check('an error status comes back', err.status, 401);
    check('an error status comes back without its body', err.body, '');
    var echo = await Relay.relayOne(fakeFetch(textReply(200, 'you sent ' + KEY)), STORED, { id: 'a', body: '{}' });
    check('a reply echoing the key: status 0', echo.status, 0);
    check('a reply echoing the key: empty body', echo.body, '');
    var thrown = await Relay.relayOne(fakeFetch(new TypeError('redirect was blocked')), STORED, { id: 'a', body: '{}' });
    check('a failed fetch (a refused redirect) is status 0', thrown.status, 0);
    check('a failed fetch has no error text', thrown.body, '');

    // --- messages are accepted only from the expected window -------------------
    var parent = {};
    var am = Relay.acceptMessage;
    check('another origin refused', am({ origin: 'https://evil.example', source: parent, data: {} }, GAME, parent), false);
    check('the game origin from the parent accepted', am({ origin: GAME, source: parent, data: {} }, GAME, parent), true);
    check('the game origin from another window refused', am({ origin: GAME, source: {}, data: {} }, GAME, parent), false);
    check('a non-object message refused', am({ origin: GAME, source: parent, data: 'x' }, GAME, parent), false);
    check('an array message refused', am({ origin: GAME, source: parent, data: [] }, GAME, parent), false);
    check('no game origin refuses all', am({ origin: '', source: parent, data: {} }, '', parent), false);
    check('no expected window refuses all', am({ origin: GAME, source: null, data: {} }, GAME, null), false);

    // --- sealed keys ---------------------------------------------------------
    var c = globalThis.crypto;
    var sealed = await Relay.seal(c, 'correct horse', STORED, 'Alice');
    check('the sealed blob does not contain the key', sealed.indexOf(KEY), -1);
    check('the sealed blob does not contain the endpoint', sealed.indexOf('openai'), -1);
    check('a sealed blob has the blob shape', Relay.isSealedBlob(sealed), true);
    check('plain text is not a blob', Relay.isSealedBlob('{"key":"x"}'), false);
    check('a non-string is not a blob', Relay.isSealedBlob(null), false);
    var blob = JSON.parse(sealed);
    check('the sealed blob is versioned', blob.v, 1);
    check('the salt is 16 bytes', Buffer.from(blob.salt, 'base64').length, 16);
    check('the IV is 12 bytes', Buffer.from(blob.iv, 'base64').length, 12);
    var opened = await Relay.unseal(c, 'correct horse', sealed, 'Alice');
    check('round trip: key', opened.key, KEY);
    check('round trip: endpoint', opened.endpoint, STORED.endpoint);
    check('round trip: model', opened.model, STORED.model);
    check('round trip: account name case does not matter', (await Relay.unseal(c, 'correct horse', sealed, 'ALICE')).key, KEY);
    var wrong = await Relay.unseal(c, 'wrong horse', sealed, 'Alice').then(function () { return 'opened'; }, function () { return 'rejected'; });
    check('a wrong passphrase rejects', wrong, 'rejected');
    var moved = await Relay.unseal(c, 'correct horse', sealed, 'Bob').then(function () { return 'opened'; }, function () { return 'rejected'; });
    check('a blob moved to another account does not open', moved, 'rejected');
    var v2 = JSON.stringify(Object.assign({}, blob, { v: 2 }));
    var future = await Relay.unseal(c, 'correct horse', v2, 'Alice').then(function () { return 'opened'; }, function () { return 'rejected'; });
    check('an unknown blob version rejects', future, 'rejected');

    check('storageKey differs per account', Relay.storageKey('Alice') !== Relay.storageKey('Bob'), true);
    check('storageKey ignores case', Relay.storageKey('Alice'), Relay.storageKey('aLICE'));

    await relayRules(c);
    await requestCap();
    await frameBootTests();
    await setupBootTests();
    staticPageChecks();

    console.log('\n' + (checks - failures) + '/' + checks + ' checks passed');
    if (failures > 0) {
        console.log(failures + ' FAILED');
        process.exit(1);
    }
}

// --- the frame's rules and the key window's rules, apart from the DOM --------
async function relayRules(c) {
    var posts = [];
    var store = memoryStorage();
    var rf = fakeFetch(textReply(200, '{"r":1}'));
    var relay = Relay.createRelay({ post: function (m) { posts.push(m); }, storage: store, fetchFn: rf });
    var win = Relay.createSetup(c);

    // Before any login the window has nobody to set a key up for.
    var noAcct = relay.handlePopup({ type: 'popup-hello' });
    check('before hello the window is told there is no account', noAcct.view, 'none');
    check('setup before hello is refused', (await win.setup({ endpoint: STORED.endpoint, key: KEY, model: 'm', remember: false }, '')).ok, false);
    var early = relay.handlePopup({ type: 'settings', endpoint: STORED.endpoint, key: KEY, model: 'm', sealed: null, remember: false });
    check('settings before hello are refused', early.ok, false);

    relay.handle({ type: 'hello', account: 'Alice' });
    var state = relay.handlePopup({ type: 'popup-hello' });
    check('the window is told the account', state.account, 'Alice');
    check('with nothing saved the window shows setup', state.view, 'setup');
    check('with nothing saved there is no blob to unlock', state.sealed, null);
    check('the window state never carries a key', JSON.stringify(state).indexOf(KEY), -1);

    var shortPass = await win.setup({ endpoint: STORED.endpoint, key: KEY, model: 'm', remember: true, pass: 'short' }, 'Alice');
    check('remember needs a passphrase of eight or more', shortPass.ok, false);
    var badEp = await win.setup({ endpoint: 'https://u:p@x.example/v1', key: KEY, model: 'm', remember: false }, 'Alice');
    check('setup refuses a user-info endpoint', badEp.ok, false);
    var spaced = await win.setup({ endpoint: STORED.endpoint, key: KEY, model: 'gpt 4', remember: false }, 'Alice');
    check('setup refuses a model name the server would refuse', spaced.ok, false);
    var noKey = await win.setup({ endpoint: STORED.endpoint, key: '', model: 'm', remember: false }, 'Alice');
    check('setup refuses an empty key', noKey.ok, false);

    var ok = await win.setup({ endpoint: STORED.endpoint, key: KEY, model: 'gpt-4.1-mini', remember: true, pass: 'correct horse' }, 'Alice');
    check('setup with remember succeeds', ok.ok, true);
    check('the settings message carries exactly endpoint,key,model,remember,sealed,type', sortedKeys(ok.msg), 'endpoint,key,model,remember,sealed,type');
    check('the settings message never carries the passphrase', JSON.stringify(ok.msg).indexOf('correct horse'), -1);
    check('the settings message carries a sealed blob when remembered', Relay.isSealedBlob(ok.msg.sealed), true);
    check('the sealed blob is not the key in plain text', ok.msg.sealed.indexOf(KEY), -1);

    var applied = relay.handlePopup(ok.msg);
    check('the frame accepts the settings', applied.ok, true);
    check('the blob is stored under Alice', store.get(Relay.storageKey('Alice')), ok.msg.sealed);
    var ready = posts.filter(function (p) { return p.type === 'status' && p.ready; });
    check('ready is posted with the model', ready.length > 0 && ready[ready.length - 1].model, 'gpt-4.1-mini');
    check('the game page is told to hide the panel', posts[posts.length - 1].type, 'hide');

    var bogus = relay.handlePopup({ type: 'settings', endpoint: STORED.endpoint, key: KEY, model: 'm', sealed: '{"key":"x"}', remember: true });
    check('a remembered key with no real blob is refused', bogus.ok, false);
    var badKey = relay.handlePopup({ type: 'settings', endpoint: STORED.endpoint, key: 'has space', model: 'm', sealed: null, remember: false });
    check('a key the relay could not send is refused', badKey.ok, false);
    check('a refused settings message changes nothing', relay.handlePopup({ type: 'popup-hello' }).model, 'gpt-4.1-mini');

    await relay.handle({ type: 'request', id: 'deadbeef', body: { a: 1 } });
    var resp = posts.filter(function (p) { return p.type === 'response'; });
    check('a request is answered', resp.length, 1);
    check('the answer carries only type,id,status,body', sortedKeys(resp[0]), 'body,id,status,type');
    check('the answer body is the reply', resp[0].body, '{"r":1}');
    var n = posts.length;
    relay.handle({ type: 'request', id: 'not hex!', body: {} });
    relay.handle({ type: 'request', id: 42, body: {} });
    check('a request with a malformed id is dropped', posts.length, n);

    // Another account on the same browser.
    relay.handle({ type: 'hello', account: 'Bob' });
    var bobStatus = posts[posts.length - 1];
    check('Bob sees no ready relay', bobStatus.ready, false);
    check('Bob is not offered Alice\'s saved key', bobStatus.locked, false);
    check('Bob is shown setup, not unlock', relay.handle({ type: 'setup' }).show, 'setup');
    check('Bob\'s window is not handed Alice\'s blob', relay.handlePopup({ type: 'popup-hello' }).sealed, null);
    await relay.handle({ type: 'request', id: 'beef', body: {} });
    check('Alice\'s key never answers for Bob', posts[posts.length - 1].status, 0);
    check('Bob\'s request never reached fetch', rf.calls.length, 1);

    // Back to Alice: the saved key must be unlocked again, in the window.
    relay.handle({ type: 'hello', account: 'Alice' });
    check('Alice is locked after a switch', posts[posts.length - 1].locked, true);
    check('Alice is shown unlock', relay.handle({ type: 'setup' }).show, 'unlock');
    var locked = relay.handlePopup({ type: 'popup-hello' });
    check('the window is shown unlock', locked.view, 'unlock');
    check('the window is handed the sealed blob to open', locked.sealed, ok.msg.sealed);
    var badUnlock = await win.unlock('wrong horse', locked.sealed, 'Alice');
    check('a wrong passphrase fails cleanly', badUnlock.message, 'That passphrase did not open it.');
    check('a wrong passphrase yields no settings', badUnlock.msg, undefined);
    check('unlock with no blob fails cleanly', (await win.unlock('x', null, 'Alice')).ok, false);
    check('unlock with no passphrase fails cleanly', (await win.unlock('', locked.sealed, 'Alice')).ok, false);
    await relay.handle({ type: 'request', id: 'cafe', body: {} });
    check('a failed unlock leaves no partial key', posts[posts.length - 1].status, 0);
    var goodUnlock = await win.unlock('correct horse', locked.sealed, 'Alice');
    check('the right passphrase unlocks', goodUnlock.ok, true);
    check('an unlocked key comes back as settings for the frame', goodUnlock.msg.key, KEY);
    check('the frame takes the unlocked settings', relay.handlePopup(goodUnlock.msg).ok, true);
    await relay.handle({ type: 'request', id: 'cafe', body: {} });
    check('after unlock requests are answered', posts[posts.length - 1].status, 200);

    var forgot = relay.handlePopup({ type: 'forget' });
    check('forget from the window is acknowledged', forgot.ok, true);
    check('forget removes the saved blob', store.get(Relay.storageKey('Alice')), null);
    check('forget posts not ready', posts[posts.length - 1].ready, false);
    check('an unknown window message gets no reply', relay.handlePopup({ type: 'steal' }), null);

    check('no message to the game page ever carries the key, an endpoint, a header or a passphrase',
        posts.filter(leaks).length, 0);
}

// --- the request cap: the key never pays for a runaway page ---------------------
async function requestCap() {
    check('at most two requests in flight', Relay.MAX_INFLIGHT, 2);
    check('at most thirty requests a minute', Relay.MAX_PER_MINUTE, 30);

    // In flight: fetches that never resolve until told to.
    var posts = [];
    var resolvers = [];
    var slow = function () { return new Promise(function (r) { resolvers.push(r); }); };
    var relay = Relay.createRelay({ post: function (m) { posts.push(m); }, storage: memoryStorage(), fetchFn: slow });
    relay.handle({ type: 'hello', account: 'Alice' });
    relay.handlePopup({ type: 'settings', endpoint: STORED.endpoint, key: KEY, model: 'm', sealed: null, remember: false });
    posts.length = 0;
    var ids = ['a1', 'a2', 'a3'];
    ids.forEach(function (id) { relay.handle({ type: 'request', id: id, body: {} }); });
    check('two requests are fetched', resolvers.length, 2);
    check('the third is answered at once', posts.length, 1);
    check('the third is a failure for its id', JSON.stringify(posts[0]), JSON.stringify({ type: 'response', id: 'a3', status: 0, body: '' }));
    resolvers.forEach(function (r) { r(textReply(200, 'x')); });
    await until(function () { return posts.length === 3; });
    check('the two in flight are answered when they return', posts.filter(function (p) { return p.status === 200; }).length, 2);
    relay.handle({ type: 'request', id: 'a4', body: {} });
    check('a slot freed by a reply is used again', resolvers.length, 3);
    resolvers[2](textReply(200, 'x'));
    await until(function () { return posts.length === 4; });

    // Per minute, with a clock of the test's own and instant fetches.
    var t = 1000000;
    var fast = fakeFetch(textReply(200, 'x'));
    var posts2 = [];
    var relay2 = Relay.createRelay({ post: function (m) { posts2.push(m); }, storage: memoryStorage(), fetchFn: fast, now: function () { return t; } });
    relay2.handle({ type: 'hello', account: 'Alice' });
    relay2.handlePopup({ type: 'settings', endpoint: STORED.endpoint, key: KEY, model: 'm', sealed: null, remember: false });
    posts2.length = 0;
    for (var i = 0; i < 30; i++) {
        t += 1000;
        await relay2.handle({ type: 'request', id: 'b' + i, body: {} });
    }
    check('thirty requests in a minute are all fetched', fast.calls.length, 30);
    check('thirty requests in a minute are all answered', posts2.filter(function (p) { return p.status === 200; }).length, 30);
    await relay2.handle({ type: 'request', id: 'b30', body: {} });
    check('the thirty-first in the minute is not fetched', fast.calls.length, 30);
    check('the thirty-first is answered as a failure at once', posts2[posts2.length - 1].status, 0);
    t += 60000; // the first request is now a minute old
    await relay2.handle({ type: 'request', id: 'b31', body: {} });
    check('a minute later requests are fetched again', fast.calls.length, 31);
    check('a request refused for no settings is not counted', true, true);
}

// --- fake pages ------------------------------------------------------------------
function fakeElement(id) {
    var handlers = {};
    return {
        id: id, value: '', hidden: true, checked: false, textContent: '',
        attributes: {},
        setAttribute: function (k, v) { this.attributes[k] = String(v); },
        addEventListener: function (t, fn) { handlers[t] = fn; },
        fire: function (t, ev) { if (handlers[t]) { handlers[t](ev || { preventDefault: function () {} }); } }
    };
}

var FRAME_IDS = ['setup', 'unlock', 'status', 'unlockstatus', 'open', 'unlockopen', 'forget', 'close', 'unlockforget', 'unlockclose', 'host'];
var SETUP_IDS = ['waiting', 'setup', 'unlock', 'endpoint', 'key', 'model', 'remember', 'pass', 'passlabel',
    'unlockpass', 'status', 'unlockstatus', 'use', 'cancel', 'unlockbtn', 'unlockforget', 'unlockcancel', 'origin', 'ollama'];

// fakeWindow builds a window that would be the one relay.js runs in. It has
// no DOM library: only what boot touches.
function fakePage(opts) {
    var ids = opts.page === 'frame' ? FRAME_IDS : SETUP_IDS;
    var els = {};
    ids.forEach(function (i) { els[i] = fakeElement(i); });
    var listeners = {};
    var posted = [];
    var parent = { postMessage: function (m, target) { posted.push({ m: m, target: target }); } };
    var storage = {};
    var win = {
        isSecureContext: opts.secure !== false,
        crypto: globalThis.crypto,
        location: { origin: RELAY, host: 'keys.example.org' },
        localStorage: {
            getItem: function (k) { return storage[k] === undefined ? null : storage[k]; },
            setItem: function (k, v) { storage[k] = v; },
            removeItem: function (k) { delete storage[k]; }
        },
        fetch: opts.fetch || fakeFetch(textReply(200, '{"ok":1}')),
        addEventListener: function (t, fn) { listeners[t] = fn; },
        opened: [],
        closed: 0,
        open: function (url, name, features) {
            win.opened.push({ url: url, name: name, features: features });
            return opts.popup === undefined ? popupWin : opts.popup;
        },
        close: function () { win.closed++; }
    };
    win.parent = opts.topLevel ? win : parent;
    if (opts.opener !== undefined) { win.opener = opts.opener; }
    var popupWin = { posted: [], closed: false, focused: 0,
        postMessage: function (m, target) { popupWin.posted.push({ m: m, target: target }); },
        focus: function () { popupWin.focused++; } };
    var doc = {
        body: { getAttribute: function (k) { return k === 'data-page' ? opts.page : null; } },
        querySelector: function (sel) {
            return sel === 'meta[name="game-origin"]' ? { getAttribute: function () { return opts.game === undefined ? GAME : opts.game; } } : null;
        },
        getElementById: function (i) { return els[i]; }
    };
    return { win: win, doc: doc, els: els, listeners: listeners, posted: posted, parent: parent, popupWin: popupWin };
}

// --- the frame: relays, opens the key window, never shows an input ------------------
async function frameBootTests() {
    var insecure = fakePage({ page: 'frame', secure: false });
    Relay.boot(insecure.win, insecure.doc);
    check('frame: a non-secure context registers no listener', Object.keys(insecure.listeners).length, 0);

    var top = fakePage({ page: 'frame', topLevel: true });
    Relay.boot(top.win, top.doc);
    check('frame: outside a frame nothing is wired', Object.keys(top.listeners).length, 0);

    var badOrigin = fakePage({ page: 'frame', game: 'https://example.org/path' });
    Relay.boot(badOrigin.win, badOrigin.doc);
    check('frame: a malformed game origin registers no listener', Object.keys(badOrigin.listeners).length, 0);

    var unknown = fakePage({ page: 'other' });
    Relay.boot(unknown.win, unknown.doc);
    check('an unknown page registers no listener', Object.keys(unknown.listeners).length, 0);

    var bf = fakeFetch(textReply(200, '{"ok":1}'));
    var p = fakePage({ page: 'frame', fetch: bf });
    Relay.boot(p.win, p.doc);
    var fromGame = function (data, from) { p.listeners.message({ origin: from || GAME, source: p.parent, data: data }); };
    var fromPopup = function (data, from, source) {
        p.listeners.message({ origin: from || RELAY, source: source || p.popupWin, data: data });
    };

    check('frame: the page names its own host for the player to check', p.els.host.textContent, 'keys.example.org');
    fromGame({ type: 'hello', account: 'Alice' });
    fromGame({ type: 'setup' });
    check('frame: setup shows the frame panel', p.els.setup.hidden, false);
    fromGame({ type: 'setup' }, 'https://evil.example');

    // A window message before any window was opened is nobody's.
    fromPopup({ type: 'popup-hello' });
    check('frame: a window message before the window is opened is ignored', p.popupWin.posted.length, 0);

    p.els.open.fire('click');
    check('frame: the button opens one window', p.win.opened.length, 1);
    check('frame: the window is the setup page on the relay origin', p.win.opened[0].url, RELAY + '/companion-relay-setup.html');
    check('frame: the window is opened as a popup', /popup/.test(p.win.opened[0].features), true);
    check('frame: the window has no name to target', p.win.opened[0].name, '_blank');
    p.els.open.fire('click');
    check('frame: a second click brings the open window forward', p.popupWin.focused, 1);
    check('frame: a second click opens no second window', p.win.opened.length, 1);

    fromPopup({ type: 'popup-hello' }, RELAY, { other: true });
    check('frame: a hello from another window on the relay origin is ignored', p.popupWin.posted.length, 0);
    fromPopup({ type: 'popup-hello' }, GAME);
    check('frame: a hello from the window at another origin is ignored', p.popupWin.posted.length, 0);
    fromPopup({ type: 'popup-hello' });
    check('frame: the window is answered', p.popupWin.posted.length, 1);
    check('frame: the answer goes to the relay origin only', p.popupWin.posted[0].target, RELAY);
    check('frame: the answer is the state', p.popupWin.posted[0].m.type, 'popup-state');
    check('frame: the state names the account', p.popupWin.posted[0].m.account, 'Alice');
    check('frame: the state shows setup', p.popupWin.posted[0].m.view, 'setup');

    var settings = { type: 'settings', endpoint: STORED.endpoint, key: KEY, model: 'gpt-4.1-mini', sealed: null, remember: false };
    var gamePosts = p.posted.length;
    fromPopup(settings, GAME);
    fromPopup(settings, RELAY, { other: true });
    check('frame: settings from anyone but the window are ignored', p.posted.length, gamePosts);
    fromPopup(settings);
    check('frame: the window is told the key is in use', p.popupWin.posted[1].m.type, 'popup-done');
    check('frame: and that it worked', p.popupWin.posted[1].m.ok, true);
    var statuses = p.posted.filter(function (x) { return x.m.type === 'status'; });
    check('frame: the game page hears ready with the model', statuses[statuses.length - 1].m.ready && statuses[statuses.length - 1].m.model, 'gpt-4.1-mini');
    check('frame: the game page is told to hide the panel', p.posted[p.posted.length - 1].m.type, 'hide');
    check('frame: the panel is hidden', p.els.setup.hidden, true);

    fromGame({ type: 'request', id: 'feed', body: { a: 1 } });
    await until(function () { return p.posted.some(function (x) { return x.m.type === 'response'; }); });
    var r = p.posted.filter(function (x) { return x.m.type === 'response'; })[0];
    check('frame: a request is answered', r && r.m.status, 200);
    check('frame: the request went to the stored endpoint', bf.calls[0].url, 'https://api.openai.com/v1/chat/completions');
    check('frame: the request carried the key from the window', bf.calls[0].init.headers.Authorization, 'Bearer ' + KEY);

    // A message claiming the game origin but sent from another window.
    var before = p.posted.length;
    p.listeners.message({ origin: GAME, source: {}, data: { type: 'request', id: 'beef', body: {} } });
    await tick();
    check('frame: a message from a window other than the parent is ignored', p.posted.length, before);
    check('frame: it never reached fetch', bf.calls.length, 1);

    // Forget from the window: the panel stays open, on setup.
    p.popupWin.closed = true;
    fromGame({ type: 'setup' });
    fromPopup({ type: 'forget' });
    check('frame: forget from the window is acknowledged', p.popupWin.posted[2].m.ok, true);
    check('frame: forget leaves the panel on setup', p.els.setup.hidden, false);
    check('frame: the game page hears not ready', p.posted.filter(function (x) { return x.m.type === 'status'; }).pop().m.ready, false);

    check('frame: every message to the game page names the game origin, never *',
        p.posted.every(function (x) { return x.target === GAME; }), true);
    check('frame: something was posted', p.posted.length > 0, true);
    check('frame: no message to the game page carries the key', p.posted.filter(function (x) { return leaks(x.m); }).length, 0);
    check('frame: no message to the window carries the key', JSON.stringify(p.popupWin.posted).indexOf(KEY), -1);
    var inDom = Object.keys(p.els).some(function (i) {
        var e = p.els[i];
        return String(e.value).indexOf(KEY) !== -1 || String(e.textContent).indexOf(KEY) !== -1 ||
            JSON.stringify(e.attributes).indexOf(KEY) !== -1;
    });
    check('frame: the key is nowhere in the DOM', inDom, false);

    // A blocked popup is explained, not swallowed.
    var blocked = fakePage({ page: 'frame', popup: null });
    Relay.boot(blocked.win, blocked.doc);
    blocked.listeners.message({ origin: GAME, source: blocked.parent, data: { type: 'hello', account: 'Alice' } });
    blocked.listeners.message({ origin: GAME, source: blocked.parent, data: { type: 'setup' } });
    blocked.els.open.fire('click');
    check('frame: a blocked window is explained', /pop-ups/.test(blocked.els.status.textContent), true);
    check('frame: the explanation names the relay host', /keys\.example\.org/.test(blocked.els.status.textContent), true);
}

// --- the key window: top level, speaks only to the frame that opened it -----------
async function setupBootTests() {
    var framed = fakePage({ page: 'setup', opener: { postMessage: function () {} } });
    Relay.boot(framed.win, framed.doc);
    check('window: framed, nothing is wired', Object.keys(framed.listeners).length, 0);

    var orphan = fakePage({ page: 'setup', topLevel: true, opener: null });
    Relay.boot(orphan.win, orphan.doc);
    check('window: with no opener nothing is wired', Object.keys(orphan.listeners).length, 0);

    var insecure = fakePage({ page: 'setup', topLevel: true, secure: false, opener: { postMessage: function () {} } });
    Relay.boot(insecure.win, insecure.doc);
    check('window: a non-secure context registers no listener', Object.keys(insecure.listeners).length, 0);

    function openWindow() {
        var toFrame = [];
        var opener = { postMessage: function (m, target) { toFrame.push({ m: m, target: target }); } };
        var w = fakePage({ page: 'setup', topLevel: true, opener: opener });
        Relay.boot(w.win, w.doc);
        w.toFrame = toFrame;
        w.opener = opener;
        w.fromFrame = function (data, from, source) {
            w.listeners.message({ origin: from || RELAY, source: source || opener, data: data });
        };
        return w;
    }

    var w = openWindow();
    check('window: it greets its opener', w.toFrame.length, 1);
    check('window: the greeting is popup-hello', w.toFrame[0].m.type, 'popup-hello');
    check('window: the greeting names the relay origin, never *', w.toFrame[0].target, RELAY);
    check('window: it shows its own host for the player to check', w.els.origin.textContent, 'keys.example.org');
    check('window: until the frame answers it shows the waiting note', w.els.waiting.hidden, false);
    check('window: until the frame answers no form is shown', w.els.setup.hidden && w.els.unlock.hidden, true);

    w.fromFrame({ type: 'popup-state', account: 'Alice', view: 'setup', endpoint: '', model: '', sealed: null }, RELAY, { other: true });
    check('window: a state from a window other than the opener is ignored', w.els.setup.hidden, true);
    w.fromFrame({ type: 'popup-state', account: 'Alice', view: 'setup', endpoint: '', model: '', sealed: null }, GAME);
    check('window: a state from the opener at another origin is ignored', w.els.setup.hidden, true);
    w.fromFrame({ type: 'popup-state', account: '', view: 'none', endpoint: '', model: '', sealed: null });
    check('window: with no account the form stays hidden', w.els.setup.hidden, true);
    check('window: and the player is told to log in', /Log in/.test(w.els.waiting.textContent), true);
    w.fromFrame({ type: 'popup-state', account: 'Alice', view: 'setup', endpoint: 'https://openrouter.ai/api/v1', model: 'm-1', sealed: null });
    check('window: the frame\'s state shows the form', w.els.setup.hidden, false);
    check('window: the waiting note goes', w.els.waiting.hidden, true);
    check('window: the endpoint in use is offered', w.els.endpoint.value, 'https://openrouter.ai/api/v1');
    check('window: the model in use is offered', w.els.model.value, 'm-1');

    w.els.endpoint.value = 'https://api.openai.com/v1';
    w.els.key.value = KEY;
    w.els.model.value = 'gpt-4.1-mini';
    w.els.remember.checked = true;
    w.els.remember.fire('change');
    check('window: ticking remember shows the passphrase', w.els.pass.hidden, false);
    w.els.pass.value = 'short';
    w.els.use.fire('click');
    await until(function () { return w.els.status.textContent !== 'Working...'; });
    check('window: a short passphrase is refused in place', /eight/.test(w.els.status.textContent), true);
    check('window: a refusal sends nothing to the frame', w.toFrame.length, 1);
    check('window: a refusal keeps what was typed', w.els.key.value, KEY);

    w.els.pass.value = 'correct horse';
    w.els.key.fire('keydown', { key: 'Enter', preventDefault: function () {} });
    await until(function () { return w.toFrame.length === 2; });
    var sent = w.toFrame[1];
    check('window: enter in the key field sends the settings to the frame', sent.m.type, 'settings');
    check('window: the settings go to the relay origin only', sent.target, RELAY);
    check('window: the settings carry the key', sent.m.key, KEY);
    check('window: the settings carry a sealed blob', Relay.isSealedBlob(sent.m.sealed), true);
    check('window: the settings say remember', sent.m.remember, true);
    check('window: the settings never carry the passphrase', JSON.stringify(sent.m).indexOf('correct horse'), -1);
    check('window: the key field is cleared once sent', w.els.key.value, '');
    check('window: the passphrase field is cleared once sent', w.els.pass.value, '');
    check('window: it stays open until the frame answers', w.win.closed, 0);

    w.fromFrame({ type: 'popup-done', ok: false, message: 'That key could not be saved.' });
    check('window: a refusal from the frame is shown', w.els.status.textContent, 'That key could not be saved.');
    check('window: a refusal from the frame leaves it open', w.win.closed, 0);
    w.fromFrame({ type: 'popup-done', ok: true, message: '' });
    check('window: once the frame has the key the window closes', w.win.closed, 1);
    check('window: and its opener is nulled', w.win.opener, null);

    // Unlocking a saved key.
    var sealed = await Relay.seal(globalThis.crypto, 'correct horse', STORED, 'Alice');
    var u = openWindow();
    u.fromFrame({ type: 'popup-state', account: 'Alice', view: 'unlock', endpoint: '', model: '', sealed: sealed });
    check('window: a saved key shows unlock', u.els.unlock.hidden, false);
    check('window: a saved key does not show setup', u.els.setup.hidden, true);
    u.els.unlockpass.value = 'wrong horse';
    u.els.unlockbtn.fire('click');
    await until(function () { return u.els.unlockstatus.textContent !== 'Working...'; });
    check('window: a wrong passphrase is refused in place', u.els.unlockstatus.textContent, 'That passphrase did not open it.');
    check('window: a wrong passphrase sends nothing', u.toFrame.length, 1);
    check('window: the passphrase field is cleared after a try', u.els.unlockpass.value, '');
    u.els.unlockpass.value = 'correct horse';
    u.els.unlockpass.fire('keydown', { key: 'Enter', preventDefault: function () {} });
    await until(function () { return u.toFrame.length === 2; });
    check('window: the right passphrase sends the opened key to the frame', u.toFrame[1].m.key, KEY);
    check('window: the opened key keeps its blob', u.toFrame[1].m.sealed, sealed);
    check('window: the opened key stays remembered', u.toFrame[1].m.remember, true);
    check('window: the passphrase never leaves the window', JSON.stringify(u.toFrame).indexOf('correct horse'), -1);

    var g = openWindow();
    g.fromFrame({ type: 'popup-state', account: 'Alice', view: 'unlock', endpoint: '', model: '', sealed: sealed });
    g.els.unlockforget.fire('click');
    check('window: forget tells the frame', g.toFrame[1].m.type, 'forget');
    check('window: forget carries nothing else', sortedKeys(g.toFrame[1].m), 'type');

    var x = openWindow();
    x.fromFrame({ type: 'popup-state', account: 'Alice', view: 'setup', endpoint: '', model: '', sealed: null });
    x.els.key.value = KEY;
    x.els.cancel.fire('click');
    check('window: cancel clears the key', x.els.key.value, '');
    check('window: cancel closes the window', x.win.closed, 1);
    check('window: cancel nulls the opener', x.win.opener, null);
    check('window: cancel sends nothing', x.toFrame.length, 1);
}

// --- the two pages: no form, no input in the frame, no password field anywhere ---
function staticPageChecks() {
    var frame = fs.readFileSync(path.join(WEB_DIR, 'relay.html'), 'utf8');
    var setup = fs.readFileSync(path.join(WEB_DIR, 'relay-setup.html'), 'utf8');
    check('frame page: is marked as the frame', /<body data-page="frame">/.test(frame), true);
    check('frame page: has no input at all', /<input/i.test(frame), false);
    check('frame page: has no form', /<form/i.test(frame), false);
    check('frame page: loads the relay script from its own origin', frame.indexOf('<script src="/companion-relay.js"></script>') !== -1, true);
    check('setup page: is marked as the key window', /<body data-page="setup">/.test(setup), true);
    check('setup page: has no form', /<form/i.test(setup), false);
    check('setup page: has no password field', /type="password"/i.test(setup), false);
    check('setup page: has no submit button', /type="submit"/i.test(setup), false);
    check('setup page: loads the relay script from its own origin', setup.indexOf('<script src="/companion-relay.js"></script>') !== -1, true);
    ['key', 'pass', 'unlockpass'].forEach(function (id) {
        var m = setup.match(new RegExp('<input id="' + id + '"[^>]*>'));
        check('setup page: the ' + id + ' input exists', !!m, true);
        var tag = m ? m[0] : '';
        check('setup page: the ' + id + ' input is plain text', /type="text"/.test(tag), true);
        check('setup page: the ' + id + ' input refuses autocomplete', /autocomplete="off"/.test(tag), true);
        check('setup page: the ' + id + ' input is masked by style', /class="secret"/.test(tag), true);
    });
    check('setup page: the masking style is declared', /\.secret\{-webkit-text-security:disc\}/.test(setup), true);
    check('setup page: a browser without the masking style is told so', /@supports not \(-webkit-text-security: disc\)/.test(setup), true);
    [frame, setup].forEach(function (page, i) {
        var name = i === 0 ? 'frame page' : 'setup page';
        check(name + ': has no inline script', /<script>/.test(page), false);
        check(name + ': has no inline handler', /\son[a-z]+=/i.test(page), false);
        check(name + ': names the key window path the script opens', true, true);
    });
    var js = fs.readFileSync(path.join(WEB_DIR, 'relay.js'), 'utf8');
    check('relay.js: opens the key window on its own origin', js.indexOf("win.open(relayOrigin + SETUP_PATH") !== -1, true);
    check('relay.js: the key window path is the setup page', Relay.SETUP_PATH, '/companion-relay-setup.html');
    check('relay.js: never posts with a wildcard origin', /postMessage\([^)]*['"]\*['"]/.test(js), false);
    check('relay.js: the frame never reads the game page\'s storage or the popup\'s', /opener\.localStorage|parent\.localStorage/.test(js), false);
}

main().catch(function (e) {
    console.log('FAIL uncaught: ' + (e && e.stack || e));
    process.exit(1);
});
