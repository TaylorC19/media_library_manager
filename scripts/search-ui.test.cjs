// Run with: node --test scripts/search-ui.test.cjs
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

class Element {
  constructor(tag = 'div') {
    this.tagName = tag; this.children = []; this.listeners = {};
    this.value = ''; this.textContent = ''; this.disabled = false; this.dataset = {}; this.attributes = {}; this.isConnected = true;
    this.open = true; this.href = 'http://localhost/en/search';
  }
  get href() { return this._href; }
  set href(value) { this._href = new URL(value, 'http://localhost').href; }
  setAttribute(name,value) { this.attributes[name] = value; }
  hasAttribute(name) { return name in this.attributes; }
  replaceChildren(...children) { this.children = children; }
  focus() { this.focused = true; }
  showModal() { this.open = true; }
  close() { this.open = false; this.emit('close'); }
  querySelector() { return this.children[0] || new Element(); }
  appendChild(child) { this.children.push(child); return child; }
  addEventListener(name, callback) { (this.listeners[name] ||= []).push(callback); }
  emit(name, event = {}) { for (const cb of this.listeners[name] || []) cb(event); }
  getAttribute(name) { return name === 'data-locale' ? 'en' : this[name]; }
  get innerHTML() { return this.html || ''; }
  set innerHTML(value) { this.html=value; this.children = []; }
  async play() {}
}
function descendants(element, tag) {
  return element.children.flatMap(child => [ ...(child.tagName === tag ? [child] : []), ...descendants(child, tag) ]);
}
function setup(options = {}) {
  const ids = Object.fromEntries([
    'scan-root', 'scan-video', 'scan-status', 'scan-results', 'scan-barcode-input', 'media_type',
    'search-bucket', 'barcode-search', 'scan-lookup-btn', 'scan-start-cam', 'scan-stop-cam',
    'scan-again-btn', 'scan-camera-hint', 'scan-fallback-search', 'q',
  ].map(id => [id, new Element()]));
  ids['search-bucket'].value = 'wishlist';
  ids.media_type.value = 'book';
  ids['scan-barcode-input'].value = '9781234567897';
  const document = {
    getElementById(id) { return ids[id] || null; },
    createElement(tag) { return new Element(tag); },
    createTextNode(text) { return Object.assign(new Element('#text'), { textContent: text }); },
    addEventListener() {},
  };
  const requests = [];
  const track = { stops: 0, stop() { this.stops++; } };
  const stream = { getTracks: () => [track] };
  const controls = { stops: 0, stop() { this.stops++; } };
  let decodes = 0;
  const context = {
    document, URL, Map,
    navigator: { mediaDevices: { getUserMedia: options.getUserMedia || (async () => stream) } },
    fetch: async (url, init) => {
      requests.push({url, body:JSON.parse(init.body)});
      if (options.fetch) return options.fetch(url, init);
      return { ok:true, json: async () => options.response || {
        barcode:'9781234567897',
        candidates:[{source:'provider',provider:'open_library',providerId:'OL1W',mediaType:'book',title:'Test Book'}],
        fallback:{manualQuery:'Test Book',mediaType:'book'},
      } };
    },
    requestAnimationFrame: () => 1, cancelAnimationFrame() {},
    addEventListener() {}, location: { origin:'http://localhost' },
    ZXingBrowser: {
      BarcodeFormat:{EAN_13:7,EAN_8:6,UPC_A:14,UPC_E:15},
      BrowserMultiFormatReader: class {
        // Match the API exported by the bundled ZXing browser library.
        async decodeFromVideoElement(video, callback) { decodes++; controls.callback = callback; return controls; }
      },
    },
  };
  context.window = context;
  vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../internal/static/public/js/scan.js'),'utf8'),context);
  return { ids, requests, track, controls, stream, decodes: () => decodes };
}
const flush = () => new Promise(resolve => setImmediate(resolve));

test('barcode results link to full previews without choosing a destination', async () => {
  const { ids, requests } = setup();
  ids['scan-lookup-btn'].emit('click'); await flush();
  assert.equal(requests[0].body.preferredMediaType, 'book');
  assert.equal(descendants(ids['scan-results'], 'form').length, 0);
  const link = descendants(ids['scan-results'], 'a')[0];
  const url = new URL(link.href);
  assert.equal(url.pathname, '/en/search/details');
  assert.equal(url.searchParams.get('provider'), 'open_library');
  assert.equal(url.searchParams.get('external_id'), 'OL1W');
  assert.equal(url.searchParams.get('bucket'), null);
});

test('local barcode matches open a metadata preview rather than saving immediately', async () => {
  const { ids } = setup({response:{barcode:'9781234567897', candidates:[{
    source:'local', mediaRecordId:'record1', title:'Test', linkedLibraryEntries:[],
  }]}});
  ids['scan-lookup-btn'].emit('click'); await flush();
  assert.equal(descendants(ids['scan-results'], 'form').length, 0);
  const url = new URL(descendants(ids['scan-results'], 'a')[0].href);
  assert.equal(url.searchParams.get('media_record_id'), 'record1');
});

test('ZXing fallback uses the bundled API and closing the disclosure releases camera and decoder',async () => {
  const {ids,track,controls,decodes} = setup();
  ids['scan-start-cam'].emit('click'); await flush();
  assert.equal(decodes(),1);
  ids['barcode-search'].open=false; ids['barcode-search'].emit('toggle');
  assert.equal(track.stops,1); assert.equal(controls.stops,1);
  assert.equal(ids['scan-video'].srcObject,null);
});

test('closing the disclosure while camera permission is pending disposes the eventual stream',async () => {
  let resolve;
  const {ids,stream,track,decodes} = setup({getUserMedia: () => new Promise(r => {resolve=r;})});
  ids['scan-start-cam'].emit('click');
  ids['barcode-search'].open=false; ids['barcode-search'].emit('toggle');
  resolve(stream); await flush();
  assert.equal(track.stops,1); assert.equal(decodes(),0);
  assert.equal(ids['scan-video'].srcObject,null);
});


function dialogSetup(fetchResponse) {
  const ids = Object.fromEntries(['search-detail-dialog','search-detail-content','search-detail-close','create-manually','media_type','q'].map(id => [id,new Element()]));
  ids['search-detail-dialog'].open=false;
  ids['search-detail-dialog'].dataset={loading:'Loading',error:'Failed',retry:'Retry'};
  ids['create-manually'].href='http://localhost/en/library/new';
  ids.media_type.value='book'; ids.q.value='A/B & C';
  const listeners={};
  const document={getElementById:id=>ids[id],createElement:tag=>new Element(tag),addEventListener:(name,cb)=>listeners[name]=cb};
  const requests=[];
  const context={document, URL, AbortController, window:{location:{origin:'http://localhost'}},fetch:async(url,options)=>{
    requests.push({url,options});
    return fetchResponse ? fetchResponse() : {ok:true,text:async()=>'<h2 id="search-detail-title">Full details</h2>'};
  }};
  vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../internal/static/public/js/search.js'),'utf8'),context);
  const trigger = new Element('a'); trigger.href='http://localhost/en/search/details?provider=open_library&external_id=OL1W&media_type=book';
  function click(extra={}) {
    let prevented=false;
    listeners.click({target:{closest:()=>trigger},button:0,preventDefault(){prevented=true;},...extra});
    return prevented;
  }
  return {ids,requests,trigger,click};
}

test('opening a result loads read-only details and closing restores focus', async () => {
  const {ids,requests,trigger,click}=dialogSetup();
  assert.equal(click(),true); await flush();
  assert.equal(ids['search-detail-dialog'].open,true);
  assert.equal(new URL(requests[0].url,'http://localhost').searchParams.get('fragment'),'1');
  assert.equal(requests[0].options.method,undefined); // Fetch defaults to GET, never POST.
  assert.match(ids['search-detail-content'].innerHTML,/Full details/);
  ids['search-detail-close'].emit('click');
  assert.equal(ids['search-detail-dialog'].open,false);
  assert.equal(trigger.focused,true);
  assert.equal(requests[0].options.signal.aborted,true);
});

test('closing during loading prevents a late response from reopening or replacing the dialog', async () => {
  let resolve;
  const {ids,click}=dialogSetup(()=>new Promise(r=>resolve=r));
  click(); ids['search-detail-close'].emit('click');
  resolve({ok:true,text:async()=>'<h2>Stale response</h2>'}); await flush();
  assert.equal(ids['search-detail-dialog'].open,false);
  assert.doesNotMatch(ids['search-detail-content'].innerHTML,/Stale response/);
});

test('preview errors offer retry, while modified clicks keep ordinary navigation', async () => {
  const {ids,requests,click}=dialogSetup(()=>({ok:false}));
  assert.equal(click({ctrlKey:true}),false); assert.equal(requests.length,0);
  click(); await flush();
  const content=ids['search-detail-content'];
  assert.equal(content.children[0].textContent,'Failed');
  assert.equal(content.children[1].textContent,'Retry');
  content.children[1].emit('click'); await flush();
  assert.equal(requests.length,2);
});

test('manual fallback retains the current query and type without a search destination selector', () => {
  const {ids}=dialogSetup();
  const url=new URL(ids['create-manually'].href);
  assert.equal(url.searchParams.get('q'),'A/B & C');
  assert.equal(url.searchParams.get('media_type'),'book');
  assert.equal(url.searchParams.get('bucket'),null);
});

test('Scan again restarts the camera and decoder after a successful read', async () => {
  const {ids,controls,decodes}=setup();
  ids['scan-start-cam'].emit('click'); await flush();
  controls.callback({getText:()=> '9781234567897'},null); await flush();
  assert.equal(ids['scan-video'].srcObject,null);
  assert.equal(ids['scan-again-btn'].disabled,false);
  ids['scan-again-btn'].emit('click'); await flush();
  assert.equal(decodes(),2);
  assert.notEqual(ids['scan-video'].srcObject,null);
});

test('Scan again ignores an older lookup that finishes after the new scan', async () => {
  const pending=[];
  const {ids,controls,requests}=setup({fetch:()=>new Promise(resolve=>pending.push(resolve))});
  ids['scan-start-cam'].emit('click'); await flush();
  controls.callback({getText:()=> '9781234567897'},null); await flush();
  ids['scan-again-btn'].emit('click'); await flush();
  controls.callback({getText:()=> '9787654321098'},null); await flush();
  assert.equal(requests.length,2);
  const response=title=>({ok:true,json:async()=>({barcode:'9787654321098',candidates:[{source:'provider',provider:'open_library',providerId:'OL1W',mediaType:'book',title}]})});
  pending[1](response('New scan')); await flush();
  pending[0](response('Old scan')); await flush();
  const titles=descendants(ids['scan-results'],'h3').map(element=>element.textContent);
  assert.equal(titles.includes('New scan'),true);
  assert.equal(titles.includes('Old scan'),false);
});
