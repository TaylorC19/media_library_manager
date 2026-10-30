const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

test('language links retain live filters, layout and page after HTMX and history navigation',()=>{
 const links=['en','ja'].map(locale=>({dataset:{localeTarget:locale},href:''}));
 const callbacks={};
 const window={location:{href:'http://localhost/en/?q=books&view=grid&page=2'},addEventListener:(event,fn)=>callbacks[event]=fn};
 const document={querySelectorAll:()=>links,addEventListener:(event,fn)=>callbacks[event]=fn};
 vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../internal/static/public/js/navigation.js'),'utf8'),{window,document,URL});
 assert.equal(links[1].href,'/ja/?q=books&view=grid&page=2');
 window.location.href='http://localhost/en/?q=A%2FB&format=paperback&view=list&page=3';
 callbacks['htmx:afterSettle']();
 assert.equal(links[1].href,'/ja/?q=A%2FB&format=paperback&view=list&page=3');
 window.location.href='http://localhost/ja/wishlist?q=game&view=grid';
 callbacks['htmx:historyRestore']();
 assert.equal(links[0].href,'/en/wishlist?q=game&view=grid');
 window.location.href='http://localhost/en/?q=restored';
 callbacks.popstate();
 assert.equal(links[1].href,'/ja/?q=restored');
});
