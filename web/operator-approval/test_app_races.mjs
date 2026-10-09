import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';

const source=readFileSync(new URL('./app.js',import.meta.url),'utf8');
const id='apr_12345678';
const turn=()=>new Promise(resolve=>setImmediate(resolve));
const pending={request_id:id,status:'pending',csrf_token:'csrf',decision_nonce:'nonce',step_up_required:false};

function harness(){
  const elements=new Map(),calls=[];
  const document={getElementById(id){
    if(!elements.has(id))elements.set(id,{hidden:true,disabled:false,textContent:'',value:'',handlers:{},
      reportValidity:()=>true,addEventListener(event,handler){this.handlers[event]=handler;}});
    return elements.get(id);
  }};
  const fetch=(url,opts={})=>new Promise(resolve=>calls.push({url,opts,reply(data,status=200){
    resolve({ok:status===200,status,json:async()=>data});
  }}));
  document.getElementById('step-up-password');
  vm.runInNewContext(source,{document,fetch,URL,Date,encodeURIComponent,
    location:{href:'https://operator.test/operator?request='+id,pathname:'/operator'}});
  return {elements,calls,click:name=>elements.get(name).handlers.click()};
}

test('approve/deny and refresh cannot race a decision or replace its proof',async()=>{
  const h=harness();h.calls[0].reply(pending);await turn();
  const deciding=h.click('approve');
  await h.click('deny');await h.click('refresh');
  assert.equal(h.calls.length,2,'Only the explicit first decision may be sent');
  assert.deepEqual(JSON.parse(h.calls[1].opts.body),{decision:'approve',decision_nonce:'nonce'});
  assert.equal(h.elements.get('refresh').disabled,true);
  h.calls[1].reply({request_id:id,status:'approved'});await deciding;
  assert.match(h.elements.get('status').textContent,/aprovada com sucesso/);
  assert.equal(h.elements.get('approve').disabled,true);
  assert.equal(h.elements.get('deny').disabled,true);
  assert.equal(h.elements.get('refresh').disabled,false);
});

test('a delayed pending GET cannot restore controls after logout',async()=>{
  const h=harness();h.calls[0].reply(pending);await turn();
  const reading=h.click('refresh'),logout=h.click('logout');
  h.calls[2].reply({status:'signed_out'});await logout;
  h.calls[1].reply({...pending,decision_nonce:'stale'});await reading;
  assert.match(h.elements.get('status').textContent,/Sessão encerrada/);
  assert.equal(h.elements.get('details').hidden,true);
  assert.equal(h.elements.get('login-panel').hidden,false);
  assert.equal(h.elements.get('approve').disabled,true);
  await h.click('approve');assert.equal(h.calls.length,3);
});

test('logout during step-up prevents the subsequent decision POST',async()=>{
  const h=harness();h.calls[0].reply({...pending,step_up_required:true});await turn();
  h.elements.get('step-up-password').value='synthetic-password';
  const deciding=h.click('approve'),logout=h.click('logout');
  assert.ok(h.calls[1].url.endsWith('/step-up'));
  h.calls[2].reply({status:'signed_out'});await logout;
  h.calls[1].reply({ok:true});await deciding;
  assert.equal(h.calls.filter(x=>x.url.endsWith('/decision')).length,0);
  assert.equal(h.elements.get('step-up-password').value,'');
  assert.match(h.elements.get('status').textContent,/Sessão encerrada/);
});

test('a delayed decision response cannot overwrite logout',async()=>{
  const h=harness();h.calls[0].reply(pending);await turn();
  const deciding=h.click('deny'),logout=h.click('logout');
  h.calls[2].reply({status:'signed_out'});await logout;
  h.calls[1].reply({request_id:id,status:'denied'});await deciding;
  assert.match(h.elements.get('status').textContent,/Sessão encerrada/);
  assert.equal(h.elements.get('details').hidden,true);
});

test('older GET cannot replace a newer terminal status',async()=>{
  const h=harness(),newer=h.click('refresh');
  h.calls[1].reply({request_id:id,status:'revoked',csrf_token:'csrf'});await newer;
  h.calls[0].reply(pending);await turn();
  assert.match(h.elements.get('status').textContent,/revogada/);
  assert.equal(h.elements.get('approve').disabled,true);
});

test('a decision acknowledgement for another request cannot claim success',async()=>{
  const h=harness();h.calls[0].reply(pending);await turn();
  const deciding=h.click('approve');
  h.calls[1].reply({request_id:'apr_other1234',status:'approved'});await deciding;
  assert.match(h.elements.get('status').textContent,/Broker não confirmou/);
  assert.doesNotMatch(h.elements.get('status').textContent,/aprovada com sucesso/);
  await h.click('approve');assert.equal(h.calls.length,2,'Fresh state required after an ambiguous reply');
});
