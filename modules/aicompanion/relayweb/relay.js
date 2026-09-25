// relay.js: the companion key relay. This origin (RelayOrigin) is the only
// place a player's own model key ever exists. It serves two pages, both run
// by this one script:
//
// The FRAME (/companion-relay.html), framed by the game page. It holds the
// key and relays requests. It talks to the game page only by postMessage:
//
//   game -> frame  {type:'hello', account}    who is logged in
//                  {type:'setup'}             show the frame's key panel
//                  {type:'request', id, body} one chat completions body
//   frame -> game  {type:'status', ready, model?, locked}
//                  {type:'response', id, status, body}
//                  {type:'hide'}
//
// The POPUP (/companion-relay-setup.html), a top-level window the FRAME
// opens on its own origin, so its address bar authenticates the key form
// and no page can draw over it. It is the only place the key is typed. It
// talks only to the frame that opened it (window.opener, same origin), by
// postMessage; storage partitioning cannot come between them, because no
// storage is shared: the popup hands the frame the key, and the frame owns
// the storage.
//
//   popup -> frame {type:'popup-hello'}
//                  {type:'settings', endpoint, key, model, sealed, remember}
//                  {type:'forget'}
//   frame -> popup {type:'popup-state', account, view, endpoint, model, sealed}
//                  {type:'popup-done', ok, message}
//
// The key never appears in any message to the game page, any URL, attribute,
// log or error text. The endpoint a request goes to is the one the player
// stored, never one a message names. Served by modules/aicompanion/relaypage.go
// with a CSP that allows no inline script; tests in
// tools/jstest/companion-relay.test.js.

(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) {
    module.exports = api; // node, for the tests
  } else {
    root.CompanionRelay = api;
    api.boot(root, root.document);
  }
}(typeof self !== 'undefined' ? self : this, function () {
  'use strict';

  var ITER = 600000;
  var SALT_BYTES = 16;
  var IV_BYTES = 12;
  var BLOB_VERSION = 1;
  // The game page's websocket closes on a frame over 64 KiB, so a reply
  // bigger than this never leaves the relay.
  var MAX_REPLY_BYTES = 60 * 1024;
  var MIN_PASS = 8;
  var MAX_MODEL = 100;
  var MAX_ACCOUNT = 64;
  var FETCH_TIMEOUT_MS = 90000;
  // The server allows far fewer calls than this; more means the page or the
  // server is misbehaving, and the key must not pay for it.
  var MAX_INFLIGHT = 2;
  var MAX_PER_MINUTE = 30;
  var MINUTE_MS = 60000;
  var SETUP_PATH = '/companion-relay-setup.html';

  // isAllowedEndpoint accepts https anywhere, or http only on this
  // computer, with no user info, query or fragment.
  function isAllowedEndpoint(u) {
    if (typeof u !== 'string' || u === '') { return false; }
    var p;
    try { p = new URL(u); } catch (e) { return false; }
    if (p.username !== '' || p.password !== '' || p.search !== '' || p.hash !== '') { return false; }
    if (p.protocol === 'https:') { return p.hostname !== ''; }
    return p.protocol === 'http:' && (p.hostname === 'localhost' || p.hostname === '127.0.0.1');
  }

  // endpointURL is the stored endpoint's chat completions address.
  function endpointURL(endpoint) {
    var p = new URL(endpoint);
    return p.origin + p.pathname.replace(/\/+$/, '') + '/chat/completions';
  }

  function isValidModel(m) {
    // A subset of the server's relayModelOK: printable, no spaces.
    return typeof m === 'string' && /^[\x21-\x7e]+$/.test(m) && m.length <= MAX_MODEL;
  }

  function isValidKey(k) {
    return typeof k === 'string' && k.length > 0 && k.length <= 1024 && /^[\x21-\x7e]+$/.test(k);
  }

  function isValidId(id) {
    return typeof id === 'string' && /^[0-9a-f]{1,64}$/i.test(id);
  }

  // isSealedBlob is the shape check on a stored blob: versioned, with the
  // three base64 fields. Opening it is unseal's job.
  function isSealedBlob(s) {
    if (typeof s !== 'string' || s.length > 8192) { return false; }
    var o;
    try { o = JSON.parse(s); } catch (e) { return false; }
    return !!o && o.v === BLOB_VERSION && typeof o.salt === 'string' && typeof o.iv === 'string' && typeof o.ct === 'string';
  }

  function storageKey(account) { return 'companion-key:' + String(account || '').toLowerCase(); }

  function utf8(s) { return new TextEncoder().encode(s); }

  function b64(u8) { var s = ''; for (var i = 0; i < u8.length; i++) { s += String.fromCharCode(u8[i]); } return btoa(s); }
  function unb64(s) { var b = atob(s), u = new Uint8Array(b.length); for (var i = 0; i < b.length; i++) { u[i] = b.charCodeAt(i); } return u; }

  async function deriveKey(subtle, pass, salt) {
    var base = await subtle.importKey('raw', utf8(pass), 'PBKDF2', false, ['deriveKey']);
    return subtle.deriveKey({ name: 'PBKDF2', salt: salt, iterations: ITER, hash: 'SHA-256' },
      base, { name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
  }

  // seal encrypts {endpoint, key, model} under the passphrase. The account's
  // storage key is bound in as additional data, so a blob copied to another
  // account's slot does not open.
  async function seal(cryptoObj, pass, secret, account) {
    var salt = cryptoObj.getRandomValues(new Uint8Array(SALT_BYTES));
    var iv = cryptoObj.getRandomValues(new Uint8Array(IV_BYTES));
    var k = await deriveKey(cryptoObj.subtle, pass, salt);
    var plain = utf8(JSON.stringify({ endpoint: secret.endpoint, key: secret.key, model: secret.model }));
    var ct = await cryptoObj.subtle.encrypt({ name: 'AES-GCM', iv: iv, additionalData: utf8(storageKey(account)) }, k, plain);
    return JSON.stringify({ v: BLOB_VERSION, salt: b64(salt), iv: b64(iv), ct: b64(new Uint8Array(ct)) });
  }

  // unseal opens a sealed blob or throws. It never returns part of one.
  async function unseal(cryptoObj, pass, sealed, account) {
    var o = JSON.parse(sealed);
    if (!o || o.v !== BLOB_VERSION || typeof o.salt !== 'string' || typeof o.iv !== 'string' || typeof o.ct !== 'string') {
      throw new Error('unsupported');
    }
    var salt = unb64(o.salt), iv = unb64(o.iv);
    if (salt.length !== SALT_BYTES || iv.length !== IV_BYTES) { throw new Error('unsupported'); }
    var k = await deriveKey(cryptoObj.subtle, pass, salt);
    var pt = await cryptoObj.subtle.decrypt({ name: 'AES-GCM', iv: iv, additionalData: utf8(storageKey(account)) }, k, unb64(o.ct));
    var s = JSON.parse(new TextDecoder().decode(pt));
    if (!s || !isAllowedEndpoint(s.endpoint) || !isValidKey(s.key) || !isValidModel(s.model)) {
      throw new Error('unsupported');
    }
    return { endpoint: s.endpoint, key: s.key, model: s.model };
  }

  // readCapped reads a reply's text, or returns null once it passes cap
  // bytes, without holding more than that.
  async function readCapped(r, cap) {
    var declared = r.headers && typeof r.headers.get === 'function' ? Number(r.headers.get('content-length')) : 0;
    if (declared > cap) {
      try { if (r.body && r.body.cancel) { await r.body.cancel(); } } catch (e) { /* nothing to free */ }
      return null;
    }
    if (r.body && typeof r.body.getReader === 'function') {
      var reader = r.body.getReader(), dec = new TextDecoder(), total = 0, out = '';
      for (;;) {
        var step = await reader.read();
        if (step.done) { break; }
        total += step.value.length;
        if (total > cap) {
          try { await reader.cancel(); } catch (e) { /* nothing to free */ }
          return null;
        }
        out += dec.decode(step.value, { stream: true });
      }
      return out + dec.decode();
    }
    var text = await r.text();
    return utf8(text).length > cap ? null : text;
  }

  // relayOne posts one request body to the STORED endpoint and returns only
  // {id, status, body}. A message cannot name a URL or a header. An error
  // status comes back without its body: providers echo part of a bad key in
  // their error text, and the server needs only the status.
  async function relayOne(fetchFn, settings, msg) {
    var fail = { id: msg.id, status: 0, body: '' };
    if (!settings || !isValidKey(settings.key) || !isAllowedEndpoint(settings.endpoint)) { return fail; }
    var body = msg.body;
    if (body === undefined || body === null) { return fail; }
    if (typeof body !== 'string') { body = JSON.stringify(body); }
    var ctl = typeof AbortController === 'function' ? new AbortController() : null;
    var timer = ctl ? setTimeout(function () { ctl.abort(); }, FETCH_TIMEOUT_MS) : null;
    try {
      var init = {
        method: 'POST', mode: 'cors', credentials: 'omit', referrerPolicy: 'no-referrer',
        redirect: 'error', cache: 'no-store',
        headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + settings.key },
        body: body
      };
      if (ctl) { init.signal = ctl.signal; }
      var r = await fetchFn(endpointURL(settings.endpoint), init);
      var status = typeof r.status === 'number' ? r.status : 0;
      if (status < 200 || status > 299) {
        try { if (r.body && r.body.cancel) { await r.body.cancel(); } } catch (e) { /* nothing to free */ }
        return { id: msg.id, status: status, body: '' };
      }
      var text = await readCapped(r, MAX_REPLY_BYTES);
      if (text === null || text.indexOf(settings.key) !== -1) { return fail; }
      return { id: msg.id, status: status, body: text };
    } catch (e) {
      return fail;
    } finally {
      if (timer) { clearTimeout(timer); }
    }
  }

  // acceptMessage admits only a message from the expected window at the
  // expected origin, carrying an object. The frame uses it for its parent
  // at the game origin and for the popup it opened at its own origin; the
  // popup uses it for its opener at its own origin.
  function acceptMessage(ev, origin, fromWin) {
    return !!ev && typeof origin === 'string' && origin !== '' && ev.origin === origin &&
      !!fromWin && ev.source === fromWin &&
      !!ev.data && typeof ev.data === 'object' && !Array.isArray(ev.data);
  }

  // createRelay is the frame's state and rules, apart from the DOM. env:
  // post(msg) to the game page, storage {get, set, remove}, fetchFn, and
  // now() for the request cap (Date.now by default). Its settings are held
  // here and nowhere else; they arrive only from the popup, by settings().
  function createRelay(env) {
    var settings = null;
    var account = '';
    var inflight = 0;
    var sentAt = []; // times of the requests fetched in the last minute
    var now = env.now || function () { return Date.now(); };

    function stored() { return account === '' ? null : env.storage.get(storageKey(account)); }

    function status() {
      var s = { type: 'status', ready: !!settings, locked: !settings && !!stored() };
      if (settings) { s.model = settings.model; }
      env.post(s);
    }

    function view() {
      if (account === '') { return 'none'; }
      return !settings && stored() ? 'unlock' : 'setup';
    }

    function fail(id) { env.post({ type: 'response', id: id, status: 0, body: '' }); }

    // overCap says whether one more fetch would pass the in-flight or
    // per-minute cap, and records it when not.
    function overCap() {
      var t = now();
      while (sentAt.length > 0 && t - sentAt[0] >= MINUTE_MS) { sentAt.shift(); }
      if (inflight >= MAX_INFLIGHT || sentAt.length >= MAX_PER_MINUTE) { return true; }
      sentAt.push(t);
      inflight++;
      return false;
    }

    function handle(data) {
      switch (data.type) {
        case 'hello': {
          var a = typeof data.account === 'string' ? data.account.trim() : '';
          if (a.length > MAX_ACCOUNT) { a = ''; }
          var changed = a.toLowerCase() !== account.toLowerCase();
          if (changed) {
            settings = null; // one account's key never serves another
          }
          account = a;
          status();
          return changed ? { hide: true } : null;
        }
        case 'setup': {
          var v = view();
          if (v === 'none') { env.post({ type: 'hide' }); return null; }
          return { show: v };
        }
        case 'request': {
          if (!isValidId(data.id)) { return null; }
          if (!settings || overCap()) {
            fail(data.id);
            return null;
          }
          var done = function (r) {
            inflight--;
            if (r) { env.post({ type: 'response', id: r.id, status: r.status, body: r.body }); } else { fail(data.id); }
          };
          return relayOne(env.fetchFn, settings, { id: data.id, body: data.body }).then(done, function () { done(null); });
        }
      }
      return null;
    }

    // popupState is what the popup needs to show its form: never the key,
    // only the account, which view, the current endpoint and model, and the
    // sealed blob when there is one to unlock.
    function popupState() {
      var v = view();
      return { type: 'popup-state', account: account, view: v,
        endpoint: settings ? settings.endpoint : '', model: settings ? settings.model : '',
        sealed: v === 'unlock' ? stored() : null };
    }

    // applySettings takes the popup's outcome: the key to use from now on,
    // and the sealed blob to keep (remember) or the order to keep nothing.
    function applySettings(data) {
      if (account === '') { return { type: 'popup-done', ok: false, message: 'Log in to the game first.' }; }
      if (!isAllowedEndpoint(data.endpoint) || !isValidKey(data.key) || !isValidModel(data.model) || data.model === '') {
        return { type: 'popup-done', ok: false, message: 'That key could not be used.' };
      }
      var note = '';
      if (data.remember === true) {
        if (!isSealedBlob(data.sealed)) { return { type: 'popup-done', ok: false, message: 'That key could not be saved.' }; }
        if (!env.storage.set(storageKey(account), data.sealed)) {
          note = 'This browser would not save it, so it is kept for this visit only.';
        }
      } else {
        env.storage.remove(storageKey(account));
      }
      settings = { endpoint: data.endpoint, key: data.key, model: data.model };
      status();
      env.post({ type: 'hide' });
      return { type: 'popup-done', ok: true, message: note };
    }

    function forget() {
      if (account !== '') { env.storage.remove(storageKey(account)); }
      settings = null;
      status();
    }

    // handlePopup takes one message from the popup and returns the reply to
    // post back to it, or null.
    function handlePopup(data) {
      switch (data.type) {
        case 'popup-hello': return popupState();
        case 'settings': return applySettings(data);
        case 'forget':
          forget();
          return { type: 'popup-done', ok: true, message: 'Your key is forgotten on this device.' };
      }
      return null;
    }

    return { handle: handle, handlePopup: handlePopup, forget: forget, view: view };
  }

  // createSetup is the popup's rules, apart from the DOM: it turns the form
  // into a settings message for the frame, sealing the key first when the
  // player wants it remembered, or opens a sealed blob with the passphrase.
  function createSetup(cryptoObj) {
    async function setup(f, account) {
      if (typeof account !== 'string' || account === '') { return { ok: false, message: 'Log in to the game first.' }; }
      var endpoint = typeof f.endpoint === 'string' ? f.endpoint.trim() : '';
      var model = typeof f.model === 'string' ? f.model.trim() : '';
      if (!isAllowedEndpoint(endpoint)) {
        return { ok: false, message: 'That endpoint is not allowed. Use an https address, or http on this computer.' };
      }
      if (!isValidModel(model) || model === '') { return { ok: false, message: 'Enter a model name.' }; }
      if (!isValidKey(f.key)) { return { ok: false, message: 'Enter your key.' }; }
      var msg = { type: 'settings', endpoint: endpoint, key: f.key, model: model, sealed: null, remember: false };
      if (f.remember) {
        if (typeof f.pass !== 'string' || f.pass.length < MIN_PASS) {
          return { ok: false, message: 'Choose a passphrase of at least eight characters.' };
        }
        try { msg.sealed = await seal(cryptoObj, f.pass, msg, account); } catch (e) {
          return { ok: false, message: 'This browser could not lock your key. Nothing was saved.' };
        }
        msg.remember = true;
      }
      return { ok: true, message: '', msg: msg };
    }

    async function unlock(pass, sealed, account) {
      if (!isSealedBlob(sealed)) { return { ok: false, message: 'There is no saved key to unlock.' }; }
      if (typeof pass !== 'string' || pass === '') { return { ok: false, message: 'Enter your passphrase.' }; }
      var opened;
      try { opened = await unseal(cryptoObj, pass, sealed, account); } catch (e) {
        return { ok: false, message: 'That passphrase did not open it.' };
      }
      return { ok: true, message: '', msg: { type: 'settings', endpoint: opened.endpoint, key: opened.key,
        model: opened.model, sealed: sealed, remember: true } };
    }

    return { setup: setup, unlock: unlock };
  }

  function isOrigin(s) {
    if (typeof s !== 'string' || s === '') { return false; }
    try { return new URL(s).origin === s; } catch (e) { return false; }
  }

  // boot wires whichever page this is. It does nothing outside a secure
  // context or on a page it does not know.
  function boot(win, doc) {
    if (!win || !doc || !win.isSecureContext || !doc.body || typeof doc.body.getAttribute !== 'function') { return; }
    switch (doc.body.getAttribute('data-page')) {
      case 'frame': bootFrame(win, doc); break;
      case 'setup': bootSetup(win, doc); break;
    }
  }

  // bootFrame wires the framed relay. It does nothing outside a frame or
  // without a game origin to answer to. The frame shows no input: the key
  // is typed in the popup it opens, and arrives from it.
  function bootFrame(win, doc) {
    if (win.parent === win) { return; }
    var meta = doc.querySelector('meta[name="game-origin"]');
    var gameOrigin = meta ? meta.getAttribute('content') : '';
    if (!isOrigin(gameOrigin)) { return; }
    var relayOrigin = win.location.origin;

    var el = function (id) { return doc.getElementById(id); };
    var setupView = el('setup'), unlockView = el('unlock');
    var statusEl = el('status'), unlockStatus = el('unlockstatus');

    var storage = {
      get: function (k) { try { return win.localStorage.getItem(k); } catch (e) { return null; } },
      set: function (k, v) { try { win.localStorage.setItem(k, v); return true; } catch (e) { return false; } },
      remove: function (k) { try { win.localStorage.removeItem(k); } catch (e) { /* nothing stored */ } }
    };
    var relay = createRelay({
      post: function (m) { win.parent.postMessage(m, gameOrigin); },
      storage: storage,
      fetchFn: win.fetch.bind(win)
    });

    function show(which) {
      setupView.hidden = which !== 'setup';
      unlockView.hidden = which !== 'unlock';
    }
    function hide() { show(''); win.parent.postMessage({ type: 'hide' }, gameOrigin); }

    var host = el('host');
    if (host) { host.textContent = win.location.host; }

    // The popup is opened by the frame itself, from a click inside the
    // frame, so its opener is this window and never the game page. One at a
    // time: a second click brings the open one forward.
    var popup = null;
    function openPopup() {
      if (popup && !popup.closed) {
        try { popup.focus(); } catch (e) { /* focus is a courtesy */ }
        return;
      }
      popup = win.open(relayOrigin + SETUP_PATH, '_blank', 'popup=yes,width=480,height=680');
      if (!popup) {
        var blocked = 'Your browser blocked the key window. Allow pop-ups for ' + win.location.host + ' and try again.';
        statusEl.textContent = blocked;
        unlockStatus.textContent = blocked;
      }
    }

    win.addEventListener('message', function (ev) {
      if (popup && acceptMessage(ev, relayOrigin, popup)) {
        var reply = relay.handlePopup(ev.data);
        if (reply) {
          popup.postMessage(reply, relayOrigin);
          if (reply.type === 'popup-done' && reply.ok) {
            // A forgotten key leaves the panel open on setup; a new or
            // unlocked key closes it (applySettings told the game page).
            show(ev.data.type === 'forget' ? 'setup' : '');
          }
        }
        return;
      }
      if (!acceptMessage(ev, gameOrigin, win.parent)) { return; }
      var r = relay.handle(ev.data);
      if (r && r.show) {
        statusEl.textContent = '';
        unlockStatus.textContent = '';
        show(r.show);
      } else if (r && r.hide) {
        hide(); // another account: close any open panel
      }
    });

    el('open').addEventListener('click', openPopup);
    el('unlockopen').addEventListener('click', openPopup);
    el('forget').addEventListener('click', function () {
      relay.forget();
      statusEl.textContent = 'Your key is forgotten on this device.';
    });
    el('unlockforget').addEventListener('click', function () {
      relay.forget();
      show('setup');
    });
    el('close').addEventListener('click', hide);
    el('unlockclose').addEventListener('click', hide);
  }

  // bootSetup wires the popup. It runs only as a top-level window with an
  // opener, and speaks only to that opener at its own origin: the frame
  // that opened it. Framed, or opened by anyone else, it shows nothing.
  function bootSetup(win, doc) {
    if (win.parent !== win || !win.opener) { return; }
    var opener = win.opener;
    var relayOrigin = win.location.origin;
    var setupLogic = createSetup(win.crypto);

    var el = function (id) { return doc.getElementById(id); };
    var waiting = el('waiting'), setupView = el('setup'), unlockView = el('unlock');
    var endpointIn = el('endpoint'), keyIn = el('key'), modelIn = el('model');
    var rememberIn = el('remember'), passIn = el('pass'), passLabel = el('passlabel');
    var unlockIn = el('unlockpass'), statusEl = el('status'), unlockStatus = el('unlockstatus');

    var account = '';
    var sealed = null;
    var busy = false;

    var originEl = el('origin');
    if (originEl) { originEl.textContent = win.location.host; }
    var ollama = el('ollama');
    if (ollama) {
      ollama.textContent = 'For a model on this computer, such as Ollama, allow this page\'s address (' +
        relayOrigin + ') in its settings, for Ollama in OLLAMA_ORIGINS.';
    }

    function clearSecrets() { keyIn.value = ''; passIn.value = ''; unlockIn.value = ''; }
    function show(which) {
      waiting.hidden = which !== 'waiting';
      setupView.hidden = which !== 'setup';
      unlockView.hidden = which !== 'unlock';
    }
    function post(msg) { opener.postMessage(msg, relayOrigin); }
    // leave severs the popup from its opener and closes it. Nothing typed
    // survives it.
    function leave() {
      clearSecrets();
      try { win.opener = null; } catch (e) { /* some browsers refuse; the window closes anyway */ }
      win.close();
    }

    show('waiting');

    win.addEventListener('message', function (ev) {
      if (!acceptMessage(ev, relayOrigin, opener)) { return; }
      var d = ev.data;
      switch (d.type) {
        case 'popup-state': {
          account = typeof d.account === 'string' ? d.account : '';
          sealed = isSealedBlob(d.sealed) ? d.sealed : null;
          if (account === '' || (d.view !== 'setup' && d.view !== 'unlock')) {
            waiting.textContent = 'Log in to the game first, then open this window again.';
            show('waiting');
            return;
          }
          if (typeof d.endpoint === 'string' && d.endpoint !== '') { endpointIn.value = d.endpoint; }
          if (typeof d.model === 'string' && d.model !== '') { modelIn.value = d.model; }
          show(d.view);
          break;
        }
        case 'popup-done': {
          busy = false;
          if (d.ok === true) { leave(); return; }
          var m = typeof d.message === 'string' ? d.message : 'That did not work.';
          statusEl.textContent = m;
          unlockStatus.textContent = m;
          break;
        }
      }
    });

    post({ type: 'popup-hello' });

    rememberIn.addEventListener('change', function () {
      passLabel.hidden = !rememberIn.checked;
      passIn.hidden = !rememberIn.checked;
      if (!rememberIn.checked) { passIn.value = ''; }
    });

    function useKey() {
      if (busy) { return; }
      busy = true;
      statusEl.textContent = 'Working...';
      var fields = { endpoint: endpointIn.value, key: keyIn.value, model: modelIn.value,
        remember: rememberIn.checked, pass: passIn.value };
      setupLogic.setup(fields, account).then(function (r) {
        fields = null;
        if (!r.ok) {
          statusEl.textContent = r.message;
          busy = false;
          return;
        }
        clearSecrets();
        post(r.msg); // the frame answers with popup-done
      });
    }

    function unlockKey() {
      if (busy) { return; }
      busy = true;
      unlockStatus.textContent = 'Working...';
      var pass = unlockIn.value;
      setupLogic.unlock(pass, sealed, account).then(function (r) {
        pass = null;
        clearSecrets();
        if (!r.ok) {
          unlockStatus.textContent = r.message;
          busy = false;
          return;
        }
        post(r.msg);
      });
    }

    function onEnter(fn) {
      return function (ev) { if (ev && ev.key === 'Enter') { ev.preventDefault(); fn(); } };
    }

    el('use').addEventListener('click', useKey);
    [endpointIn, keyIn, modelIn, passIn].forEach(function (input) { input.addEventListener('keydown', onEnter(useKey)); });
    el('unlockbtn').addEventListener('click', unlockKey);
    unlockIn.addEventListener('keydown', onEnter(unlockKey));
    el('unlockforget').addEventListener('click', function () {
      if (busy) { return; }
      busy = true;
      clearSecrets();
      post({ type: 'forget' });
    });
    el('cancel').addEventListener('click', leave);
    el('unlockcancel').addEventListener('click', leave);
  }

  return {
    ITER: ITER, MAX_REPLY_BYTES: MAX_REPLY_BYTES, MAX_INFLIGHT: MAX_INFLIGHT, MAX_PER_MINUTE: MAX_PER_MINUTE,
    SETUP_PATH: SETUP_PATH,
    isAllowedEndpoint: isAllowedEndpoint, endpointURL: endpointURL, storageKey: storageKey, isSealedBlob: isSealedBlob,
    seal: seal, unseal: unseal, relayOne: relayOne, acceptMessage: acceptMessage,
    createRelay: createRelay, createSetup: createSetup, boot: boot
  };
}));
