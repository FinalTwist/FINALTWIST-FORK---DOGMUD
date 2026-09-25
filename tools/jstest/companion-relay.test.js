// companion-relay.test.js: guards for the companion key relay page
// (modules/aicompanion/relayweb/relay.js), the only place a player's own
// model key ever exists.
//
// Run:  node tools/jstest/companion-relay.test.js
//
// Dependency-free and assertion-library-free, matching hotinput.test.js.
// Needs node 20 or later for globalThis.crypto.subtle. Exits non-zero on
// failure.

var path = require('path');

var Relay = require(path.join(
    __dirname, '..', '..', 'modules', 'aicompanion', 'relayweb', 'relay.js'
));

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

    // --- messages are accepted only from the game page ------------------------
    var parent = {};
    var am = Relay.acceptMessage;
    check('another origin refused', am({ origin: 'https://evil.example', source: parent, data: {} }, GAME, parent), false);
    check('the game origin from the parent accepted', am({ origin: GAME, source: parent, data: {} }, GAME, parent), true);
    check('the game origin from another window refused', am({ origin: GAME, source: {}, data: {} }, GAME, parent), false);
    check('a non-object message refused', am({ origin: GAME, source: parent, data: 'x' }, GAME, parent), false);
    check('an array message refused', am({ origin: GAME, source: parent, data: [] }, GAME, parent), false);
    check('no game origin refuses all', am({ origin: '', source: parent, data: {} }, '', parent), false);

    // --- sealed keys ---------------------------------------------------------
    var c = globalThis.crypto;
    var sealed = await Relay.seal(c, 'correct horse', STORED, 'Alice');
    check('the sealed blob does not contain the key', sealed.indexOf(KEY), -1);
    check('the sealed blob does not contain the endpoint', sealed.indexOf('openai'), -1);
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

    // --- the relay's rules, apart from the DOM ------------------------------
    var posts = [];
    var store = memoryStorage();
    var rf = fakeFetch(textReply(200, '{"r":1}'));
    var relay = Relay.createRelay({ post: function (m) { posts.push(m); }, storage: store, cryptoObj: c, fetchFn: rf });

    var noAcct = await relay.setup({ endpoint: STORED.endpoint, key: KEY, model: 'm', remember: false });
    check('setup before hello is refused', noAcct.ok, false);
    relay.handle({ type: 'hello', account: 'Alice' });
    var shortPass = await relay.setup({ endpoint: STORED.endpoint, key: KEY, model: 'm', remember: true, pass: 'short' });
    check('remember needs a passphrase of eight or more', shortPass.ok, false);
    var badEp = await relay.setup({ endpoint: 'https://u:p@x.example/v1', key: KEY, model: 'm', remember: false });
    check('setup refuses a user-info endpoint', badEp.ok, false);
    var spaced = await relay.setup({ endpoint: STORED.endpoint, key: KEY, model: 'gpt 4', remember: false });
    check('setup refuses a model name the server would refuse', spaced.ok, false);
    var ok = await relay.setup({ endpoint: STORED.endpoint, key: KEY, model: 'gpt-4.1-mini', remember: true, pass: 'correct horse' });
    check('setup with remember succeeds', ok.ok, true);
    check('the blob is stored under Alice', typeof store.get(Relay.storageKey('Alice')), 'string');
    var ready = posts.filter(function (p) { return p.type === 'status' && p.ready; });
    check('ready is posted with the model', ready.length > 0 && ready[ready.length - 1].model, 'gpt-4.1-mini');

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
    await relay.handle({ type: 'request', id: 'beef', body: {} });
    check('Alice\'s key never answers for Bob', posts[posts.length - 1].status, 0);
    check('Bob\'s request never reached fetch', rf.calls.length, 1);

    // Back to Alice: the saved key must be unlocked again.
    relay.handle({ type: 'hello', account: 'Alice' });
    check('Alice is locked after a switch', posts[posts.length - 1].locked, true);
    check('Alice is shown unlock', relay.handle({ type: 'setup' }).show, 'unlock');
    var badUnlock = await relay.unlock('wrong horse');
    check('a wrong passphrase fails cleanly', badUnlock.message, 'That passphrase did not open it.');
    await relay.handle({ type: 'request', id: 'cafe', body: {} });
    check('a failed unlock leaves no partial key', posts[posts.length - 1].status, 0);
    var goodUnlock = await relay.unlock('correct horse');
    check('the right passphrase unlocks', goodUnlock.ok, true);
    await relay.handle({ type: 'request', id: 'cafe', body: {} });
    check('after unlock requests are answered', posts[posts.length - 1].status, 200);

    relay.forget();
    check('forget removes the saved blob', store.get(Relay.storageKey('Alice')), null);
    check('forget posts not ready', posts[posts.length - 1].ready, false);

    var leaked = posts.filter(function (p) { return JSON.stringify(p).indexOf(KEY) !== -1 || /endpoint|authorization|"key"/i.test(JSON.stringify(p)); });
    check('no posted message ever carries the key, an endpoint or a header', leaked.length, 0);

    // --- boot: the DOM wiring ----------------------------------------------
    await bootTests();

    console.log('\n' + (checks - failures) + '/' + checks + ' checks passed');
    if (failures > 0) {
        console.log(failures + ' FAILED');
        process.exit(1);
    }
}

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

function fakePage(opts) {
    var ids = ['setup', 'unlock', 'endpoint', 'key', 'model', 'remember', 'pass', 'passlabel',
        'unlockpass', 'status', 'unlockstatus', 'forget', 'close', 'unlockforget', 'unlockclose', 'ollama'];
    var els = {};
    ids.forEach(function (i) { els[i] = fakeElement(i); });
    var listeners = {};
    var posted = [];
    var parent = { postMessage: function (m, target) { posted.push({ m: m, target: target }); } };
    var storage = {};
    var win = {
        isSecureContext: opts.secure,
        parent: parent,
        crypto: globalThis.crypto,
        location: { origin: 'https://keys.example.org' },
        localStorage: {
            getItem: function (k) { return storage[k] === undefined ? null : storage[k]; },
            setItem: function (k, v) { storage[k] = v; },
            removeItem: function (k) { delete storage[k]; }
        },
        fetch: opts.fetch,
        addEventListener: function (t, fn) { listeners[t] = fn; }
    };
    var doc = {
        querySelector: function (sel) {
            return sel === 'meta[name="game-origin"]' ? { getAttribute: function () { return opts.game; } } : null;
        },
        getElementById: function (i) { return els[i]; }
    };
    return { win: win, doc: doc, els: els, listeners: listeners, posted: posted, parent: parent };
}

function tick() { return new Promise(function (r) { setTimeout(r, 0); }); }
async function until(fn) { for (var i = 0; i < 2000 && !fn(); i++) { await tick(); } }

async function bootTests() {
    var insecure = fakePage({ secure: false, game: GAME, fetch: fakeFetch(textReply(200, 'x')) });
    Relay.boot(insecure.win, insecure.doc);
    check('a non-secure context registers no listener', Object.keys(insecure.listeners).length, 0);

    var badOrigin = fakePage({ secure: true, game: 'https://example.org/path', fetch: fakeFetch(textReply(200, 'x')) });
    Relay.boot(badOrigin.win, badOrigin.doc);
    check('a malformed game origin registers no listener', Object.keys(badOrigin.listeners).length, 0);

    var bf = fakeFetch(textReply(200, '{"ok":1}'));
    var p = fakePage({ secure: true, game: GAME, fetch: bf });
    Relay.boot(p.win, p.doc);
    var send = function (data, from) {
        p.listeners.message({ origin: from || GAME, source: p.parent, data: data });
    };

    send({ type: 'hello', account: 'Alice' });
    send({ type: 'setup' });
    check('setup shows the key form', p.els.setup.hidden, false);
    send({ type: 'setup' }, 'https://evil.example');

    p.els.endpoint.value = 'https://api.openai.com/v1';
    p.els.key.value = KEY;
    p.els.model.value = 'gpt-4.1-mini';
    p.els.remember.checked = true;
    p.els.remember.fire('change');
    check('ticking remember shows the passphrase', p.els.pass.hidden, false);
    p.els.pass.value = 'correct horse';
    p.els.setup.fire('submit');
    await until(function () { return p.els.status.textContent !== 'Working...'; });
    check('after submit the key input is cleared', p.els.key.value, '');
    check('after submit the passphrase input is cleared', p.els.pass.value, '');
    check('after submit the form is hidden', p.els.setup.hidden, true);

    send({ type: 'request', id: 'feed', body: { a: 1 } });
    await until(function () { return p.posted.some(function (x) { return x.m.type === 'response'; }); });
    var r = p.posted.filter(function (x) { return x.m.type === 'response'; })[0];
    check('boot answers a request', r && r.m.status, 200);
    check('boot fetched the stored endpoint', bf.calls[0].url, 'https://api.openai.com/v1/chat/completions');

    // A message claiming the game origin but sent from another window.
    var before = p.posted.length;
    p.listeners.message({ origin: GAME, source: {}, data: { type: 'request', id: 'beef', body: {} } });
    await tick();
    check('a message from a window other than the parent is ignored', p.posted.length, before);
    check('it never reached fetch', bf.calls.length, 1);

    check('every postMessage names the game origin, never *',
        p.posted.every(function (x) { return x.target === GAME; }), true);
    check('something was posted', p.posted.length > 0, true);
    var dump = JSON.stringify(p.posted);
    check('no postMessage carries the key', dump.indexOf(KEY), -1);
    var inDom = Object.keys(p.els).some(function (i) {
        var e = p.els[i];
        return String(e.value).indexOf(KEY) !== -1 || String(e.textContent).indexOf(KEY) !== -1 ||
            JSON.stringify(e.attributes).indexOf(KEY) !== -1;
    });
    check('the key is nowhere in the DOM after submit', inDom, false);
    var stored = JSON.stringify(p.win.localStorage.getItem(Relay.storageKey('Alice')));
    check('the key is not in plain text in storage', stored.indexOf(KEY), -1);
}

main().catch(function (e) {
    console.log('FAIL uncaught: ' + (e && e.stack || e));
    process.exit(1);
});
