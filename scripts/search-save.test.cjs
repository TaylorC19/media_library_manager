const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const path=require('node:path');
const flush=()=>new Promise(resolve=>setImmediate(resolve));
const preview='<article data-save-bucket></article>';

function setup(){
 let document;
 function match(element,selector){
  const parts=selector.split(' ');
  if(parts.length>1){const last=parts.pop();return match(element,last)&&!!element.parent?.closest(parts.join(' '));}
  const tag=selector.match(/^[a-z][a-z0-9]*/i)?.[0];if(tag&&element.tag!==tag)return false;
  const cls=selector.match(/\.([\w-]+)/)?.[1];if(cls&&!element.className.split(' ').includes(cls))return false;
  const attr=selector.match(/\[([^=\]]+)(?:="([^"]*)")?\]/);if(attr){const actual=element.getAttribute(attr[1]);if(actual==null||(attr[2]!=null&&actual!==attr[2]))return false;}
  return true;
 }
 class E{
  constructor(tag='div'){this.tag=tag;this.children=[];this.attributes={};this.dataset={};this.listeners={};this.className='';this.disabled=false;this.open=false;this.isConnected=true;this._text='';}
  set textContent(value){this._text=value;this.children=[];}
  get textContent(){return this._text+this.children.map(c=>c.textContent).join('');}
  appendChild(child){child.parent=this;this.children.push(child);return child;}
  replaceChildren(...children){for(const c of this.children)c.isConnected=false;this.children=[];this._text='';children.forEach(c=>this.appendChild(c));}
  setAttribute(key,value){this.attributes[key]=value;if(key.startsWith('data-'))this.dataset[key.slice(5).replace(/-([a-z])/g,(_,c)=>c.toUpperCase())]=value;}
  getAttribute(key){if(key.startsWith('data-'))return this.dataset[key.slice(5).replace(/-([a-z])/g,(_,c)=>c.toUpperCase())]??null;return this.attributes[key]??this[key]??null;}
  hasAttribute(key){return this.getAttribute(key)!=null;}
  querySelectorAll(selector){return this.children.flatMap(c=>[...(match(c,selector)?[c]:[]),...c.querySelectorAll(selector)]);}
  querySelector(selector){if(selector==='#search-detail-title')return this.querySelector('h2');return this.querySelectorAll(selector)[0]||null;}
  closest(selector){return match(this,selector)?this:this.parent?.closest(selector)||null;}
  contains(child){return child===this||this.children.some(c=>c.contains(child));}
  after(element){const i=this.parent.children.indexOf(this);this.parent.children.splice(i+1,0,element);element.parent=this.parent;}
  focus(){document.activeElement=this;}
  addEventListener(name,fn){this.listeners[name]=fn;}
  showModal(){this.open=true;}
  close(){this.open=false;this.listeners.close?.();}
  get innerHTML(){return this._html||'';}
  set innerHTML(html){this._html=html;this.replaceChildren();if(html.includes('data-save-bucket'))buildActions(this);}
 }
 const root=new E();
 const ids=Object.fromEntries(['search-detail-dialog','search-detail-content','search-detail-close','create-manually','media_type','q'].map(id=>[id,new E()]));
 const dialog=ids['search-detail-dialog'],content=ids['search-detail-content'];dialog.appendChild(content);
 dialog.dataset={loading:'Loading',error:'Preview error',retry:'Retry',saving:'Saving',saveError:'Save failed',saved:'Saved',existing:'Already saved',openSaved:'Open saved item',collectionLabel:'Collection',wishlistLabel:'Wishlist',badgeCollection:'In Collection',badgeWishlist:'In Wishlist',canWrite:'true'};
 ids['create-manually'].href='http://localhost/en/library/new';ids.media_type.value='movie';ids.q.value='avatar';
 function buildActions(host){
  host.appendChild(new E('h2'));
  const status=host.appendChild(new E('p'));status.setAttribute('data-save-status','');
  for(const bucket of ['catalog','wishlist']){
   const slot=host.appendChild(new E());slot.setAttribute('data-save-bucket',bucket);
   const form=slot.appendChild(new E('form'));form.setAttribute('data-dialog-save','');form.action='http://localhost/media/import';form.fields={bucket,provider:'tmdb',external_id:'42',media_type:'movie',locale:'en'};
   const button=form.appendChild(new E('button'));button.type='submit';button.textContent='Add '+bucket;
  }
 }
 function result(kind){
  const link=root.appendChild(new E('a'));link.setAttribute('data-search-detail','');link.href=`http://localhost/en/search/details?provider=tmdb&external_id=42&media_type=${kind}&tmdb_kind=${kind}`;
  const body=link.appendChild(new E('span'));body.className='search-hit__body';const title=body.appendChild(new E('span'));title.className='search-hit__title';title.textContent='Result';return link;
 }
 const trigger=result('movie'),other=result('tv'),requests=[],listeners={};
 document={activeElement:null,getElementById:id=>ids[id]||null,createElement:tag=>new E(tag),querySelectorAll:selector=>root.querySelectorAll(selector),addEventListener:(name,fn)=>listeners[name]=fn};
 const window={location:{origin:'http://localhost',href:'http://localhost/en/search?q=avatar&view=grid'}};
 vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../internal/static/public/js/search.js'),'utf8'),{document,window,URL,URLSearchParams,Map,AbortController,FormData:class{constructor(form){this.values=form.fields;}[Symbol.iterator](){return Object.entries(this.values)[Symbol.iterator]();}},fetch(url,options){
  if(options?.method==='POST')return new Promise((resolve,reject)=>requests.push({url,options,resolve,reject}));
  return Promise.resolve({ok:true,text:async()=>preview});
 }});
 const click=link=>listeners.click({target:link,button:0,preventDefault(){}});
 const submit=bucket=>{const form=content.querySelector(`[data-save-bucket="${bucket}"]`).querySelector('form');if(form){form.querySelector('button').focus();listeners.submit({target:form,preventDefault(){}});}return form;};
 const success=(bucket='catalog')=>({ok:true,json:async()=>({entryId:'012345678901234567890123',bucket,url:'/en/library/012345678901234567890123',existing:false})});
 return {ids,content,dialog,trigger,other,requests,click,submit,success,window,document};
}

test('save stays in dialog, immediately updates only the matching badge, and offers Open saved item',async()=>{
 const s=setup();s.click(s.trigger);await flush();
 const form=s.submit('catalog');s.submit('catalog');
 assert.equal(s.requests.length,1);assert.equal(form.querySelector('button').disabled,true);
 assert.equal(s.requests[0].options.headers['X-Search-Dialog'],'1');
 assert.equal(s.requests[0].options.body.get('bucket'),'catalog');
 s.requests[0].resolve(s.success());await flush();
 assert.equal(s.dialog.open,true);assert.match(s.window.location.href,/search\?q=avatar/);
 assert.equal(s.trigger.querySelector('[data-saved-bucket="catalog"]').textContent,'In Collection');
 assert.equal(s.other.querySelector('[data-saved-bucket="catalog"]'),null);
 const slot=s.content.querySelector('[data-save-bucket="catalog"]');
 assert.equal(slot.querySelector('form'),null);assert.match(slot.querySelector('a').textContent,/Open saved item/);
 assert.equal(s.content.querySelector('[data-save-status]').textContent,'Saved');
 assert.equal(s.content.querySelector('[data-save-bucket="wishlist"]').querySelector('button').disabled,false);
});

test('failed saves preserve the form, show an error, and do not invent a badge',async()=>{
 const s=setup();s.click(s.trigger);await flush();const form=s.submit('wishlist');
 s.requests[0].resolve({ok:false});await flush();
 assert.equal(s.content.querySelector('[data-save-status]').textContent,'Save failed');
 assert.equal(form.querySelector('button').disabled,false);assert.equal(s.trigger.querySelector('.saved-badges'),null);
 s.submit('wishlist');assert.equal(s.requests.length,2);
});

test('closing a pending save does not cancel a committed save or reopen the dialog',async()=>{
 const s=setup();s.click(s.trigger);await flush();s.submit('catalog');s.dialog.close();
 s.requests[0].resolve(s.success());await flush();
 assert.equal(s.dialog.open,false);assert.equal(s.content.children.length,0);
 assert.equal(s.trigger.querySelector('[data-saved-bucket="catalog"]').textContent,'In Collection');
});

test('a late save acknowledgement cannot replace a different item dialog',async()=>{
 const s=setup();s.click(s.trigger);await flush();s.submit('catalog');
 s.click(s.other);await flush();s.requests[0].resolve(s.success());await flush();
 assert.ok(s.content.querySelector('[data-save-bucket="catalog"]').querySelector('form'));
 assert.equal(s.other.querySelector('.saved-badges'),null);
});

test('saving to both buckets keeps both badges without duplicates',async()=>{
 const s=setup();s.click(s.trigger);await flush();s.submit('catalog');s.requests[0].resolve(s.success());await flush();
 s.submit('wishlist');s.requests[1].resolve(s.success('wishlist'));await flush();
 assert.equal(s.trigger.querySelectorAll('.saved-badge').length,2);
 assert.equal(s.content.querySelectorAll('form').length,0);
});
