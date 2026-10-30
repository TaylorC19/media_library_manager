const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const flush = () => new Promise(resolve => setImmediate(resolve));

function setup() {
  const listeners = {}, windowListeners = {}, requests = [];
  const summary = {}, empty = {};
  const root = {
    dataset: {loading:'Searching',complete:'Complete',partial:'Partial'},
    querySelectorAll: () => slots,
    querySelector(selector) {
      if (selector === '[data-group-summary]') return summary;
      if (selector === '[data-groups-empty]') return empty;
      return slots.find(slot => slot.content.html.includes('data-has-results="true"')) || null;
    },
  };
  const slots = ['movie','tv','book'].map(kind => {
    const slot = {
      dataset:{searchGroupUrl:'/en/search/group?q=test&media_type='+kind,groupState:'pending',error:'Failed',retry:'Retry'},
      isConnected:true, message:{}, content:{html:'',
        set innerHTML(value) {this.html=value;},
        querySelector() { return this.html.includes('data-group-result') ? {} : null; },
        replaceChildren() {this.html='';},
      },
      closest: () => root,
      setAttribute(name,value) { this[name]=value; },
      querySelector(selector) {
        return {'[data-group-placeholder]':this.placeholder,'[data-group-message]':this.message,'[data-group-content]':this.content,'[data-search-group-retry]':this.retry}[selector] || null;
      },
    };
    slot.placeholder = {appendChild(button) {slot.retry=button;button.slot=slot;}};
    return slot;
  });
  const document = {
    querySelectorAll: () => slots.filter(slot => slot.isConnected && slot.dataset.groupState === 'pending'),
    addEventListener(name,fn) {listeners[name]=fn;},
    createElement: () => ({setAttribute(){},remove(){this.slot.retry=null;},closest(){return this.slot;}}),
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../internal/static/public/js/search-groups.js'),'utf8'),{
    document,AbortController,Map,
    window:{addEventListener(name,fn){windowListeners[name]=fn;}},
    fetch(url,options){return new Promise((resolve,reject)=>requests.push({url,options,resolve,reject}));},
  });
  return {slots,summary,empty,requests,listeners,windowListeners};
}
const response = (title,hasResults=true) => ({ok:true,text:async()=>`<section data-group-result data-has-results="${hasResults}">${title}</section>`});

test('fast groups render while slower requests remain pending; each group keeps its own slot',async()=>{
  const {slots,requests,summary,empty}=setup();
  assert.equal(requests.length,3);
  requests[2].resolve(response('Fast books'));await flush();
  assert.match(slots[2].content.html,/Fast books/);
  assert.equal(slots[2].dataset.groupState,'done');
  assert.equal(slots[0].dataset.groupState,'loading');
  assert.equal(summary.textContent,'Searching');
  requests[0].resolve(response('Movies'));requests[1].resolve(response('',false));await flush();
  assert.equal(summary.textContent,'Complete');assert.equal(empty.hidden,true);
  assert.match(slots[0].content.html,/Movies/);
});

test('failure retains completed groups and retries only the failed media type',async()=>{
  const {slots,requests,listeners,summary}=setup();
  requests[0].resolve(response('Movies'));requests[1].reject(new Error('offline'));requests[2].resolve(response('Books'));await flush();
  assert.equal(summary.textContent,'Partial');
  assert.equal(slots[1].message.textContent,'Failed');
  listeners.click({target:{closest:()=>slots[1].retry}});
  assert.equal(requests.length,4);
  assert.equal(requests[3].url,requests[1].url);
  requests[3].resolve(response('TV'));await flush();
  assert.match(slots[0].content.html,/Movies/);assert.match(slots[1].content.html,/TV/);
  assert.equal(summary.textContent,'Complete');
});

test('empty state appears only after every group finishes with no results',async()=>{
  const {requests,empty}=setup();
  requests[0].resolve(response('',false));requests[1].resolve(response('',false));await flush();
  assert.equal(empty.hidden,true);
  requests[2].resolve(response('',false));await flush();
  assert.equal(empty.hidden,false);
});

test('navigation aborts old requests and late responses cannot replace resumed results',async()=>{
  const {requests,slots,windowListeners}=setup();
  windowListeners.pagehide();
  assert.ok(requests.every(request=>request.options.signal.aborted));
  windowListeners.pageshow();assert.equal(requests.length,6);
  requests[3].resolve(response('Current movie'));await flush();
  requests[0].resolve(response('Old movie'));await flush();
  assert.match(slots[0].content.html,/Current movie/);
  assert.doesNotMatch(slots[0].content.html,/Old movie/);
});

test('fragment swaps cancel disconnected slots and cannot receive late content',async()=>{
  const {requests,slots,listeners}=setup();
  slots[0].isConnected=false;
  listeners['htmx:afterSwap']();
  assert.equal(requests[0].options.signal.aborted,true);
  requests[0].resolve(response('Stale'));await flush();
  assert.equal(slots[0].content.html,'');
  assert.equal(requests.length,3);
});

test('redirected login and malformed responses show retry instead of inserting a page',async()=>{
  const {requests,slots}=setup();
  requests[0].resolve({ok:true,redirected:true,text:async()=>'<form>Login</form>'});
  requests[1].resolve({ok:true,text:async()=>'<form>Invalid page</form>'});
  await flush();
  for(const slot of slots.slice(0,2)){
    assert.equal(slot.dataset.groupState,'error');assert.equal(slot.content.html,'');
  }
});
