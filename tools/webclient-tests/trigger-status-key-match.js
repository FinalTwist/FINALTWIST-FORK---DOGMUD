#!/usr/bin/env node
/*
 * Headless regression test for the web client's trigger engine matching a
 * "status" condition against Char.Conditions.
 *
 *   node tools/webclient-tests/trigger-status-key-match.js
 *
 * Zero dependencies. Not wired into CI — the web client has no JS test
 * infrastructure, and this exists because that gap let a real bug ship:
 *
 * THE BUG (found in code review, 2026-09-14)
 * -------------------------------------------
 * Char.Conditions is a name-keyed map. Slice 1b's buffs.DisplayName gives a
 * stacked bleed a live count in its DISPLAY label ("Bleeding (2)"), but the
 * map KEY stays the plain spec name ("Bleeding"; a repeat takes a "#n"
 * suffix, e.g. "Stoneskin#1"). evalTriggerCondition used to match a saved
 * "status has Bleeding" trigger against cond.name (falling back to the key
 * only when .name was absent), so the match went false exactly while the
 * player was bleeding hardest — and an "exclude" trigger fired instead. The
 * fix matches on the KEY with its "#n" suffix stripped, never on .name.
 *
 * This file parses the real function out of webclient-pure.html rather than
 * copying it, so it cannot drift from shipping code. It locates it by
 * function name, not line number.
 */
'use strict';

const fs = require('fs');
const path = require('path');

const HTML = path.join(__dirname, '..', '..', '_datafiles', 'html', 'public', 'webclient-pure.html');

// ---------------------------------------------------------------- extraction
function extract(src, startMarker, endMarker) {
  const s = src.indexOf(startMarker);
  const e = src.indexOf(endMarker, s + 1);
  if (s < 0) throw new Error('marker not found in webclient-pure.html: ' + startMarker);
  if (e < 0) throw new Error('end marker not found in webclient-pure.html: ' + endMarker);
  return src.slice(s, e);
}

const html = fs.readFileSync(HTML, 'utf8');
const code = extract(html, 'function evalTriggerCondition(', 'function fireTriggerCommands(');

global.GMCPStructs = {};

// Indirect eval — runs in global scope so the extracted function declaration
// becomes a global. A direct eval() under 'use strict' keeps it module-local
// and the call below would throw ReferenceError.
// eslint-disable-next-line no-eval
(0, eval)(code);

// ---------------------------------------------------------------- helpers
let fails = 0;
function check(name, cond, detail) {
  if (cond) console.log('  PASS  ' + name);
  else { fails++; console.log('  FAIL  ' + name + '\n          ' + detail); }
}

function statusCond(op, value) {
  // eslint-disable-next-line no-undef
  return evalTriggerCondition({ sourceKind: 'status', op: op, values: [value] }, []);
}

// The exact shape slice 1b produces: a stacked record's map key is the plain
// name, its .name display label carries the count; a repeated spec name gets
// a "#n" key suffix but keeps the plain .name.
GMCPStructs.Char = {
  Conditions: {
    'Bleeding': { name: 'Bleeding (2)', description: 'Losing blood.' },
    'Stoneskin#1': { name: 'Stoneskin', description: 'Skin like rock.' }
  }
};

console.log('SCENARIO — a stacked/duplicate-keyed condition matches on its KEY, not its label');

check('include "bleeding" matches the stacked record',
  statusCond('include', 'bleeding') === true,
  'a saved trigger for "Bleeding" must fire while the display label is "Bleeding (2)"');

check('exclude "bleeding" is false while bleeding (so it does NOT fire)',
  statusCond('exclude', 'bleeding') === false,
  'an exclude trigger must not fire just because the label grew a count');

check('include "stoneskin" matches a duplicate-keyed record by its stripped key',
  statusCond('include', 'stoneskin') === true,
  'Stoneskin#1 must match "stoneskin" (the key with its "#n" suffix stripped)');

check('include "bleeding (2)" (the display label) does NOT match',
  statusCond('include', 'bleeding (2)') === false,
  'the trigger UI can only ever save the plain condition name, never a label with a count');

console.log(fails === 0 ? '\nALL CHECKS PASSED' : '\n' + fails + ' CHECK(S) FAILED');
process.exit(fails ? 1 : 0);
