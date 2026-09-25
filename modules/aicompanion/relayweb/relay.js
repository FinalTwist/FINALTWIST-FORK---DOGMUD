// relay.js: the companion key relay. This page is the only place a player's
// own model key ever exists. It lives on its own origin (RelayOrigin), is
// framed by the game page, and talks to it only by postMessage:
//
//   game -> relay  {type:'hello', account}   who is logged in
//                  {type:'setup'}            show the key form (or unlock)
//                  {type:'request', id, body} one chat completions body
//   relay -> game  {type:'status', ready, model?, locked}
//                  {type:'response', id, status, body}
//                  {type:'hide'}
//
// The key never appears in any message, URL, attribute, log or error text.
// The endpoint a request goes to is the one the player stored, never one a
// message names. Served by modules/aicompanion/relaypage.go with a CSP that
// allows no inline script; tests in tools/jstest/companion-relay.test.js.

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

  // acceptMessage admits only a message the game page sent from the frame's
  // own parent, carrying an object.
  function acceptMessage(ev, gameOrigin, parentWin) {
    return !!ev && typeof gameOrigin === 'string' && gameOrigin !== '' && ev.origin === gameOrigin &&
      !!parentWin && ev.source === parentWin &&
      !!ev.data && typeof ev.data === 'object' && !Array.isArray(ev.data);
  }

  // createRelay is the relay's state and rules, apart from the DOM. env:
  // post(msg) to the game page, storage {get, set, remove}, cryptoObj,
  // fetchFn. Its settings are held here and nowhere else.
  function createRelay(env) {
    var settings = null;
    var account = '';

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
          return { show: v, endpoint: settings ? settings.endpoint : '', model: settings ? settings.model : '' };
        }
        case 'request': {
          if (!isValidId(data.id)) { return null; }
          if (!settings) {
            env.post({ type: 'response', id: data.id, status: 0, body: '' });
            return null;
          }
          return relayOne(env.fetchFn, settings, { id: data.id, body: data.body }).then(function (r) {
            env.post({ type: 'response', id: r.id, status: r.status, body: r.body });
          });
        }
      }
      return null;
    }

    async function setup(f) {
      if (account === '') { return { ok: false, message: 'Log in to the game first.' }; }
      var endpoint = typeof f.endpoint === 'string' ? f.endpoint.trim() : '';
      var model = typeof f.model === 'string' ? f.model.trim() : '';
      if (!isAllowedEndpoint(endpoint)) {
        return { ok: false, message: 'That endpoint is not allowed. Use an https address, or http on this computer.' };
      }
      if (!isValidModel(model) || model === '') { return { ok: false, message: 'Enter a model name.' }; }
      if (!isValidKey(f.key)) { return { ok: false, message: 'Enter your key.' }; }
      var next = { endpoint: endpoint, key: f.key, model: model };
      var note = '';
      if (f.remember) {
        if (typeof f.pass !== 'string' || f.pass.length < MIN_PASS) {
          return { ok: false, message: 'Choose a passphrase of at least eight characters.' };
        }
        var blob;
        try { blob = await seal(env.cryptoObj, f.pass, next, account); } catch (e) {
          return { ok: false, message: 'This browser could not lock your key. Nothing was saved.' };
        }
        if (!env.storage.set(storageKey(account), blob)) {
          note = 'This browser would not save it, so it is kept for this visit only.';
        }
      } else {
        env.storage.remove(storageKey(account));
      }
      settings = next;
      status();
      env.post({ type: 'hide' });
      return { ok: true, message: note };
    }

    async function unlock(pass) {
      var blob = stored();
      if (!blob) { return { ok: false, message: 'There is no saved key to unlock.' }; }
      if (typeof pass !== 'string' || pass === '') { return { ok: false, message: 'Enter your passphrase.' }; }
      var opened;
      try { opened = await unseal(env.cryptoObj, pass, blob, account); } catch (e) {
        return { ok: false, message: 'That passphrase did not open it.' };
      }
      settings = opened;
      status();
      env.post({ type: 'hide' });
      return { ok: true, message: '' };
    }

    function forget() {
      if (account !== '') { env.storage.remove(storageKey(account)); }
      settings = null;
      status();
    }

    return { handle: handle, setup: setup, unlock: unlock, forget: forget, view: view };
  }

  function isOrigin(s) {
    if (typeof s !== 'string' || s === '') { return false; }
    try { return new URL(s).origin === s; } catch (e) { return false; }
  }

  // boot wires the page. It does nothing outside a secure context, outside
  // a frame, or without a game origin to answer to.
  function boot(win, doc) {
    if (!win || !doc || !win.isSecureContext || win.parent === win) { return; }
    var meta = doc.querySelector('meta[name="game-origin"]');
    var gameOrigin = meta ? meta.getAttribute('content') : '';
    if (!isOrigin(gameOrigin)) { return; }

    var el = function (id) { return doc.getElementById(id); };
    var setupForm = el('setup'), unlockForm = el('unlock');
    var endpointIn = el('endpoint'), keyIn = el('key'), modelIn = el('model');
    var rememberIn = el('remember'), passIn = el('pass'), passLabel = el('passlabel');
    var unlockIn = el('unlockpass'), statusEl = el('status'), unlockStatus = el('unlockstatus');

    var storage = {
      get: function (k) { try { return win.localStorage.getItem(k); } catch (e) { return null; } },
      set: function (k, v) { try { win.localStorage.setItem(k, v); return true; } catch (e) { return false; } },
      remove: function (k) { try { win.localStorage.removeItem(k); } catch (e) { /* nothing stored */ } }
    };
    var relay = createRelay({
      post: function (m) { win.parent.postMessage(m, gameOrigin); },
      storage: storage,
      cryptoObj: win.crypto,
      fetchFn: win.fetch.bind(win)
    });

    function clearSecrets() { keyIn.value = ''; passIn.value = ''; unlockIn.value = ''; }
    function show(which) {
      setupForm.hidden = which !== 'setup';
      unlockForm.hidden = which !== 'unlock';
      if (which !== 'setup' && which !== 'unlock') { clearSecrets(); }
    }
    function hide() { show(''); win.parent.postMessage({ type: 'hide' }, gameOrigin); }

    var ollama = el('ollama');
    if (ollama) {
      ollama.textContent = 'For a model on this computer, such as Ollama, allow this page\'s address (' +
        win.location.origin + ') in its settings, for Ollama in OLLAMA_ORIGINS.';
    }

    win.addEventListener('message', function (ev) {
      if (!acceptMessage(ev, gameOrigin, win.parent)) { return; }
      var r = relay.handle(ev.data);
      if (r && r.show) {
        statusEl.textContent = '';
        unlockStatus.textContent = '';
        if (r.endpoint) { endpointIn.value = r.endpoint; }
        if (r.model) { modelIn.value = r.model; }
        show(r.show);
      } else if (r && r.hide) {
        hide(); // another account: close any open form
      }
    });

    rememberIn.addEventListener('change', function () {
      passLabel.hidden = !rememberIn.checked;
      passIn.hidden = !rememberIn.checked;
      if (!rememberIn.checked) { passIn.value = ''; }
    });

    var busy = false;
    setupForm.addEventListener('submit', function (ev) {
      ev.preventDefault();
      if (busy) { return; }
      busy = true;
      statusEl.textContent = 'Working...';
      var fields = { endpoint: endpointIn.value, key: keyIn.value, model: modelIn.value,
        remember: rememberIn.checked, pass: passIn.value };
      relay.setup(fields).then(function (r) {
        fields = null;
        clearSecrets();
        statusEl.textContent = r.message;
        if (r.ok) { show(''); }
        busy = false;
      });
    });

    unlockForm.addEventListener('submit', function (ev) {
      ev.preventDefault();
      if (busy) { return; }
      busy = true;
      unlockStatus.textContent = 'Working...';
      var pass = unlockIn.value;
      relay.unlock(pass).then(function (r) {
        pass = null;
        clearSecrets();
        unlockStatus.textContent = r.message;
        if (r.ok) { show(''); }
        busy = false;
      });
    });

    el('forget').addEventListener('click', function () {
      relay.forget();
      clearSecrets();
      statusEl.textContent = 'Your key is forgotten on this device.';
    });
    el('unlockforget').addEventListener('click', function () {
      relay.forget();
      clearSecrets();
      show('setup');
    });
    el('close').addEventListener('click', hide);
    el('unlockclose').addEventListener('click', hide);
  }

  return {
    ITER: ITER, MAX_REPLY_BYTES: MAX_REPLY_BYTES,
    isAllowedEndpoint: isAllowedEndpoint, endpointURL: endpointURL, storageKey: storageKey,
    seal: seal, unseal: unseal, relayOne: relayOne, acceptMessage: acceptMessage,
    createRelay: createRelay, boot: boot
  };
}));
