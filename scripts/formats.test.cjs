const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const path=require('node:path');

function setup({kind='movie',selected='dvd',legacy='',legacyKind=''}={}){
 const option=(value,types,label=value)=>({value,textContent:label,dataset:{mediaTypes:types},cloneNode(){return option(this.value,this.dataset.mediaTypes,this.textContent);}});
 const all=[option('dvd','movie tv game','DVD'),option('hardcover','book','Hardcover'),option('vinyl','album','Vinyl'),option('digital','movie tv album book game','Digital')];
 const blank=option('','','Not set');
 const select={value:selected,options:[blank],dataset:{legacyFormat:legacy,legacyMediaType:legacyKind},replaceChildren(...values){this.options=values;}};
 const type={value:kind,name:'media_type',closest:()=>form};
 const form={querySelector(selector){if(selector==='select[data-media-format]')return select;if(selector==='[name="media_type"]')return type;if(selector==='template[data-format-options]')return {content:{querySelectorAll:()=>all}};return null;}};
 const listeners={};
 vm.runInNewContext(fs.readFileSync(path.join(__dirname,'../internal/static/public/js/formats.js'),'utf8'),{document:{querySelectorAll:()=>[form],addEventListener:(name,fn)=>listeners[name]=fn},window:{addEventListener(){}}});
 return {select,type,change(value){type.value=value;listeners.change({target:type});},values:()=>select.options.map(o=>o.value)};
}

test('switching media types rebuilds options and clears an incompatible selection',()=>{
 const s=setup();assert.deepEqual(s.values(),['','dvd','digital']);assert.equal(s.select.value,'dvd');
 s.change('book');assert.deepEqual(s.values(),['','hardcover','digital']);assert.equal(s.select.value,'');
 s.select.value='hardcover';s.change('album');assert.deepEqual(s.values(),['','vinyl','digital']);assert.equal(s.select.value,'');
});

test('a relevant format remains selected and Any media exposes all formats',()=>{
 const s=setup({selected:'digital'});s.change('book');assert.equal(s.select.value,'digital');s.change('');assert.deepEqual(s.values(),['','dvd','hardcover','vinyl','digital']);
});

test('an existing legacy format is retained only for its original type',()=>{
 const s=setup({selected:'hardcover',legacy:'hardcover',legacyKind:'movie'});assert.deepEqual(s.values(),['','dvd','hardcover','digital']);assert.equal(s.select.value,'hardcover');
 s.change('tv');assert.deepEqual(s.values(),['','dvd','digital']);assert.equal(s.select.value,'');
});
