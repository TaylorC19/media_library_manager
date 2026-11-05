const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
function setup() {
 const events={},winEvents={},prompts=[];
 const form={dataset:{},elements:[{name:'notes',type:'textarea',value:'Original'}],method:'post',matches:()=>true,querySelectorAll:()=>[],target:''};
 const forms=[form];
 const document={body:{dataset:{unsavedMessage:'Discard changes?'}},querySelectorAll:()=>forms,addEventListener:(n,f)=>events[n]=f};
 const window={location:{href:'http://localhost/en/library/new',pathname:'/en/library/new',search:''},confirm:message=>{prompts.push(message);return window.answer;},answer:false,addEventListener:(n,f)=>winEvents[n]=f};
 vm.runInNewContext(fs.readFileSync('internal/static/public/js/unsaved.js','utf8'),{document,window,URL,WeakMap,JSON,Array,queueMicrotask,setTimeout:()=>{},MutationObserver:class {observe(){}}});
 const event=extra=>({defaultPrevented:false,preventDefault(){this.defaultPrevented=true;},stopImmediatePropagation(){this.stopped=true;},...extra});
 return {events,winEvents,prompts,form,forms,window,event};
}
test('unchanged and reverted forms do not warn; changed values protect reload',()=>{
 const s=setup();let e=s.event();s.winEvents.beforeunload(e);assert.equal(e.defaultPrevented,false);
 s.form.elements[0].value='Changed';e=s.event();s.winEvents.beforeunload(e);assert.equal(e.defaultPrevented,true);
 s.form.elements[0].value='Original';assert.equal(s.window.UnsavedChanges.confirmDiscard(),true);assert.equal(s.prompts.length,0);
});
test('cancel keeps navigation and logout blocked; accepted navigation prompts once',async()=>{
 const s=setup();s.form.elements[0].value='Changed';
 const link={href:'http://localhost/en/settings'};
 let e=s.event({target:{closest:()=>link},button:0});s.events.click(e);assert.equal(e.defaultPrevented,true);
 e=s.event({target:{method:'post',matches:()=>false}});s.events.submit(e);assert.equal(e.defaultPrevented,true);
 s.window.answer=true;e=s.event({target:{closest:()=>link},button:0});s.events.click(e);await Promise.resolve();
 e=s.event();s.winEvents.beforeunload(e);assert.equal(e.defaultPrevented,false);
 s.winEvents.pageshow();e=s.event();s.winEvents.beforeunload(e);assert.equal(e.defaultPrevented,true);
});
test('valid native save proceeds; AJAX save failure retains protection',async()=>{
 const s=setup();s.form.elements[0].value='Changed';let e=s.event({target:s.form});s.events.submit(e);assert.equal(e.defaultPrevented,false);
 e.preventDefault();await Promise.resolve();e=s.event();s.winEvents.beforeunload(e);assert.equal(e.defaultPrevented,true);
 const saved=s.window.UnsavedChanges.snapshot(s.form);s.form.elements[0].value='Edited during request';s.window.UnsavedChanges.saved(s.form,saved);
 assert.equal(s.window.UnsavedChanges.confirmDiscard(),false);
 s.form.elements[0].value='Changed';assert.equal(s.window.UnsavedChanges.confirmDiscard(),true);
});
test('checkbox changes, dynamic forms and HTMX replacements are protected',()=>{
 const s=setup();const form={elements:[{name:'remove',type:'checkbox',checked:false}],querySelectorAll:()=>[],dataset:{}};s.forms.push(form);s.window.UnsavedChanges.initialize();
 form.elements[0].checked=true;let e=s.event({detail:{target:{querySelectorAll:()=>[form]}}});s.events['htmx:beforeSwap'](e);assert.equal(e.defaultPrevented,true);
 s.window.UnsavedChanges.saved(form);assert.equal(s.window.UnsavedChanges.confirmDiscard(),true);
});
