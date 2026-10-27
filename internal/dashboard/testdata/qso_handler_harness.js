// Browser-less harness for internal/dashboard/static/app.js.
//
// It provides just enough DOM and Leaflet surface for the dashboard script to
// load, then drives the real qso_logged handler and prints what it observed as
// JSON. The Go test (qso_logged_test.go) runs this with node and asserts on the
// result; nothing here is shipped.
//
// Run as: node qso_handler_harness.js <path-to-app.js>
'use strict';

const fs = require('fs');
const vm = require('vm');

// ── Element / DOM stubs ─────────────────────────────────────────────────────
function makeClassList(el) {
  const set = new Set();
  return {
    add: (...names) => names.forEach((n) => n && set.add(n)),
    remove: (...names) => names.forEach((n) => set.delete(n)),
    contains: (n) => set.has(n),
    toggle: (n, force) => {
      const on = force === undefined ? !set.has(n) : !!force;
      on ? set.add(n) : set.delete(n);
      return on;
    },
    get value() { return [...set].join(' '); },
    _set: set,
  };
}

const elements = new Map();

function makeElement(id) {
  const el = {
    id: id || '',
    tagName: (id || 'div').toUpperCase(),
    className: '',
    style: {},
    dataset: {},
    textContent: '',
    innerHTML: '',
    value: '',
    checked: false,
    hidden: false,
    disabled: false,
    src: '',
    href: '',
    children: [],
    childNodes: [],
    clientWidth: 720,
    clientHeight: 470,
    offsetWidth: 720,
    offsetHeight: 470,
    scrollTop: 0,
    scrollHeight: 470,
    appendChild(c) { this.children.push(c); return c; },
    removeChild(c) { this.children = this.children.filter((x) => x !== c); return c; },
    insertBefore(c) { this.children.unshift(c); return c; },
    remove() {},
    setAttribute() {},
    getAttribute() { return null; },
    removeAttribute() {},
    addEventListener() {},
    removeEventListener() {},
    querySelector() { return null; },
    querySelectorAll() { return []; },
    closest() { return null; },
    focus() {},
    blur() {},
    click() {},
    getBoundingClientRect() { return {x: 0, y: 0, width: 720, height: 470, top: 0, left: 0, right: 720, bottom: 470}; },
    getContext() { return null; }, // no WebGL -> the offline raster path
    play() {},
    pause() {},
  };
  el.classList = makeClassList(el);
  return el;
}

function getEl(id) {
  if (!elements.has(id)) elements.set(id, makeElement(id));
  return elements.get(id);
}

const documentStub = {
  documentElement: makeElement('html'),
  head: makeElement('head'),
  body: makeElement('body'),
  title: '',
  getElementById: getEl,
  querySelector(sel) { return getEl(sel.replace(/^#/, '')); },
  querySelectorAll() { return []; },
  createElement(tag) { return makeElement(tag); },
  createTextNode(t) { return {textContent: t}; },
  addEventListener() {},
  removeEventListener() {},
  readyState: 'complete',
  cookie: '',
};

// ── Window / browser globals ────────────────────────────────────────────────
const storage = new Map();
const sseHandlers = [];       // one entry per EventSource: {event: [fn]}
let markerCount = 0;          // L.circleMarker calls, i.e. markers created
let layerGroupsCleared = 0;

function makeLayerGroup() {
  const layers = [];
  const group = {
    _layers: layers,
    addLayer(l) { layers.push(l); return this; },
    addTo() { return this; },
    removeLayer(l) { const i = layers.indexOf(l); if (i >= 0) layers.splice(i, 1); return this; },
    clearLayers() { layers.length = 0; layerGroupsCleared++; return this; },
    hasLayer(l) { return layers.includes(l); },
  };
  return group;
}

function makeMap() {
  const bounds = {
    getSouth: () => 30, getNorth: () => 60, getWest: () => -10, getEast: () => 30,
    contains: () => true, extend: () => bounds,
  };
  const map = {
    _panes: {},
    options: {},
    createPane(name) { this._panes[name] = makeElement(name); return this._panes[name]; },
    getPane(name) { return this._panes[name] || makeElement(name); },
    setView() { return this; },
    setMaxZoom() { return this; },
    setMinZoom() { return this; },
    getMaxZoom() { return 18; },
    getZoom() { return 4; },
    getCenter() { return {lat: 50, lng: 20}; },
    getBounds() { return bounds; },
    latLngToContainerPoint() { return {x: 100, y: 100}; },
    containerPointToLatLng() { return {lat: 50, lng: 20}; },
    getSize() { return {x: 720, y: 470}; },
    invalidateSize() { return this; },
    on() { return this; },
    off() { return this; },
    once() { return this; },
    fire() { return this; },
    addLayer() { return this; },
    removeLayer() { return this; },
    hasLayer() { return true; },
    remove() { return this; },
    flyTo() { return this; },
    flyToBounds() { return this; },
    fitBounds() { return this; },
    setZoom() { return this; },
    whenReady(fn) { fn(); return this; },
    attributionControl: {setPrefix() {}},
    zoomControl: {setPosition() {}},
    dragging: {enable() {}, disable() {}},
    doubleClickZoom: {enable() {}, disable() {}},
    scrollWheelZoom: {enable() {}, disable() {}},
  };
  return map;
}

const L = {
  map: () => makeMap(),
  layerGroup: () => makeLayerGroup(),
  marker: () => ({addTo() { return this; }, bindTooltip() { return this; }, bindPopup() { return this; }, on() {}, setLatLng() {}, getLatLng() { return {lat: 0, lng: 0}; }, remove() {}}),
  circleMarker: () => { markerCount++; return {addTo() { return this; }, bindTooltip() { return this; }, bindPopup() { return this; }, on() {}, remove() {}}; },
  polyline: () => ({addTo() { return this; }, setStyle() { return this; }, remove() {}}),
  divIcon: (o) => ({options: o}),
  icon: (o) => ({options: o}),
  latLngBounds: () => ({extend() { return this; }, contains() { return true; }}),
  latLng: (a, b) => ({lat: a, lng: b}),
  imageOverlay: () => ({addTo() { return this; }, remove() {}}),
  tileLayer: () => ({addTo() { return this; }, on() { return this; }, remove() {}}),
  maplibreGL: () => ({addTo() { return this; }, getMaplibreMap() { return null; }, on() { return this; }, remove() {}}),
  control: {attribution: () => ({addTo() { return this; }}), layers: () => ({addTo() { return this; }})},
  CRS: {EPSG4326: 'EPSG:4326'},
  Util: {setOptions: (o, opts) => Object.assign(o, opts)},
  DomEvent: {stopPropagation() {}, preventDefault() {}},
  Browser: {retina: false},
};

const windowStub = {
  L,
  document: documentStub,
  location: {search: '', host: 'test.local', protocol: 'http:', href: 'http://test.local/', reload() {}},
  navigator: {onLine: true, userAgent: 'node-harness', language: 'en'},
  localStorage: {
    getItem: (k) => (storage.has(k) ? storage.get(k) : null),
    setItem: (k, v) => storage.set(k, String(v)),
    removeItem: (k) => storage.delete(k),
  },
  performance: {now: () => Date.now()},
  // Fake timers: nothing may fire, and the process must be able to exit.
  setTimeout: () => 0,
  clearTimeout: () => {},
  setInterval: () => 0,
  clearInterval: () => {},
  requestAnimationFrame: () => 0,
  cancelAnimationFrame: () => {},
  addEventListener: () => {},
  removeEventListener: () => {},
  matchMedia: () => ({matches: false, addEventListener() {}, addListener() {}}),
  getComputedStyle: () => ({getPropertyValue: () => ''}),
  Image: function Image() { this.src = ''; this.onload = null; this.onerror = null; },
  EventSource: function EventSource(url) {
    const handlers = {};
    sseHandlers.push(handlers);
    this.url = url;
    this.addEventListener = (name, fn) => { (handlers[name] = handlers[name] || []).push(fn); };
    this.removeEventListener = () => {};
    this.close = () => {};
    this.onopen = null;
    this.onerror = null;
    this.readyState = 0;
    this._handlers = handlers;
  },
  ResizeObserver: function ResizeObserver() { this.observe = () => {}; this.disconnect = () => {}; },
  MutationObserver: function MutationObserver() { this.observe = () => {}; this.disconnect = () => {}; },
  fetch: () => new Promise(() => {}), // never resolves: the snapshot path stays quiet
  console,
  Math, Date, JSON, parseInt, parseFloat, isNaN, Number, String, Object, Array, Set, Map,
  Error, Promise, RegExp, encodeURIComponent, decodeURIComponent,
  Alert: function Alert() {},
};

windowStub.window = windowStub;
windowStub.globalThis = windowStub;
windowStub.self = windowStub;

// ── Load the real script ────────────────────────────────────────────────────
const appPath = process.argv[2];
if (!appPath) {
  console.error('usage: node qso_handler_harness.js <app.js>');
  process.exit(2);
}
const sandbox = vm.createContext(windowStub);
vm.runInContext(fs.readFileSync(appPath, 'utf8'), sandbox, {filename: 'app.js'});

const hook = sandbox.__cqopsTest;
if (!hook) {
  console.error('app.js did not expose the test hook');
  process.exit(2);
}

// ── Drive the handler ───────────────────────────────────────────────────────
// Real map initialisation (production initMap with the stubbed Leaflet), so the
// handler's rebuild has somewhere to draw. WebGL is unavailable in this
// sandbox, which is the offline raster path the map already supports.
hook.initMap({isOnline: false, drawLines: true, maxLines: 250, maxMarkers: 200,
              highlightLastQSO: true, animateActivePath: false});

function runCase(activeCall, todayCount) {
  // Start from a clean slate: reset counters and the contact list.
  hook.reset();
  markerCount = 0;
  layerGroupsCleared = 0;

  const many = [];
  for (let i = 0; i < todayCount; i++) {
    many.push({call: 'SP' + i + 'ABC', band: '20m', mode: 'SSB', grid: 'JO90',
               lat: 50 + i * 0.1, lon: 15 + i * 0.1, country: 'Poland'});
  }
  if (many.length) hook.setToday(many);
  hook.setStation({callsign: 'SP9MOA', locator: 'JO90WE', lat: 50.06, lon: 19.94});

  // A contact being worked: the active layout with its own path on the map.
  if (activeCall) {
    hook.setActive({call: activeCall, band: '80m', mode: 'FT8', grid: 'KO73dg', country: 'Russia'});
  }
  const beforeActive = {
    activeGrid: hook.activeGrid(),
    state: documentStub.getElementById('app').classList.value,
  };

  markerCount = 0;
  layerGroupsCleared = 0;
  const saved = {call: 'DL1AA', band: '20m', mode: 'CW', grid: 'JN48', lat: 48.6, lon: 9.0, country: 'Germany'};
  hook.qsoLogged(saved);

  return {
    todayBefore: todayCount,
    beforeActiveGrid: beforeActive.activeGrid,
    beforeState: beforeActive.state,
    markers: markerCount,
    layerGroupsCleared,
    todayAfter: hook.today().length,
    calls: hook.today().map((q) => q.call),
    activeGridAfter: hook.activeGrid(),
    stateAfter: documentStub.getElementById('app').classList.value,
  };
}

const results = {
  emptyLog: runCase(null, 0),
  withContacts: runCase(null, 3),
  fromActiveContact: runCase('RN3YN', 3),
};
process.stdout.write(JSON.stringify(results));
