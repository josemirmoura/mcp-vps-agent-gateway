// Browser-flow regression: a confirmed decision must not trigger a second
// GET against the pending-only API or display a misleading 404.
import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';

const source=readFileSync(new URL('./app.js',import.meta.url),'utf8');

async function scenario(decision){
  const elements=new Map();
  for(const id of ['status','details','login-panel','login-form','login','login-error',
    'approve','deny','request_id','node','subject','resource','access','ttl',
    'expires','warning','password','operator','step-up-panel','step-up-password','refresh','logout']){
    elements.set(id,{hidden:id==='details'||id==='login-panel',disabled:false,
      textContent:'',value:'',handlers:{},
      addEventListener(event,callback){this.handlers[event]=callback;}});
  }
  const calls=[];
  const fetch=async (url,opts={})=>{
    calls.push({url,opts});
    if(calls.length===1){
      return {ok:true,status:200,json:async()=>({
        request_id:'apr_12345678',node:'test',subject:'operator',
        resource:'/opt/test',access:'read',ttl_label:'300 segundos',
        expires_at:'2026-10-09T00:59:00Z',ceiling_wide:false,
        status:'pending',csrf_token:'test-csrf',decision_nonce:'nonce',step_up_required:false})};
    }
    if(calls.length===2 && opts.method==='POST'){
      return {ok:true,status:200,json:async()=>({
        status:decision==='approve'?'approved':'denied'})};
    }
    return {ok:false,status:404,json:async()=>({error:'not found'})};
  };
  const context={
    document:{getElementById:(id)=>elements.get(id)},
    location:{href:'https://operator.example.test/operator?request=apr_12345678'},
    URL,fetch,Date,encodeURIComponent,confirm:()=>true};
  vm.runInNewContext(source,context,{filename:'app.js'});
  await new Promise(resolve=>setImmediate(resolve));
  await elements.get(decision==='approve'?'approve':'deny').handlers.click();
  return {calls,elements};
}

for(const decision of ['approve','deny']){
  test('confirmed '+decision+' leaves final state visible with no pending refetch',async()=>{
    const {calls,elements}=await scenario(decision);
    assert.equal(calls.length,2,'post-decision GET must not happen');
    assert.equal(calls[1].opts.method,'POST');
    assert.equal(JSON.parse(calls[1].opts.body).decision,decision);
    assert.equal(elements.get('details').hidden,false);
    assert.equal(elements.get('approve').disabled,true);
    assert.equal(elements.get('deny').disabled,true);
    const text=elements.get('status').textContent;
    assert.match(text,decision==='approve'?/Autorização aprovada com sucesso/:/Solicitação negada com sucesso/);
    assert.doesNotMatch(text,/indisponível|inválida|não está mais pendente/);
  });
}

for (const action of ['click', 'Enter']) {
  test('sandbox login via '+action+' validates, clears password and refuses duplicate submission',async()=>{
    const elements=new Map();
    const calls=[];
    const messages=[];
    let valid=false,completeLogin;
    const document={getElementById(id){
      if(!elements.has(id))elements.set(id,{hidden:true,disabled:false,textContent:'',value:'',handlers:{},
        reportValidity:()=>valid,addEventListener(event,handler){this.handlers[event]=handler;}});
      return elements.get(id);
    }};
    const fetch=async(url,opts={})=>{
      calls.push({url,opts});
      if(url.endsWith('/login'))await new Promise(resolve=>{completeLogin=resolve});
      return {ok:false,status:url.endsWith('/login')?403:401};
    };
    const window={parent:{postMessage:message=>messages.push(message)}};
    vm.runInNewContext(source,{document,fetch,URL,Date,encodeURIComponent,window,
      location:{href:'https://operator.example.test/operator/embed?request=apr_12345678',pathname:'/operator/embed'}},{filename:'app.js'});
    await new Promise(resolve=>setImmediate(resolve));
    const trigger=()=>{
      if(action==='click')return elements.get('login').handlers.click();
      let prevented=false;
      elements.get('password').handlers.keydown({key:'Enter',isComposing:false,repeat:false,preventDefault(){prevented=true}});
      assert.equal(prevented,true,'Enter must not start native form navigation');
    };
    elements.get('password').value='synthetic-secret';
    trigger();
    assert.equal(calls.length,1,'Invalid input must not start authentication');
    valid=true;
    if(action==='Enter'){
      let prevented=false;
      elements.get('password').handlers.keydown({key:'Enter',isComposing:true,repeat:false,preventDefault(){prevented=true}});
      assert.equal(prevented,false,'Input-method composition must not trigger login');
      elements.get('password').handlers.keydown({key:'Enter',isComposing:false,repeat:true,preventDefault(){}});
      assert.equal(calls.length,1,'Composition and autorepeat must not submit credentials');
    }
    trigger();trigger();
    assert.equal(calls.length,2,'Only one explicit login may be in flight');
    assert.equal(calls[1].url,'/operator/embed/api/login');
    assert.equal(JSON.parse(calls[1].opts.body).password,'synthetic-secret');
    assert.equal(elements.get('password').value,'');
    assert.equal(elements.get('login').disabled,true);
    completeLogin();
    await new Promise(resolve=>setImmediate(resolve));
    assert.match(elements.get('login-error').textContent,/verificação recusada/);
    assert.equal(elements.get('login').disabled,false);
    assert.equal(calls.filter(call=>call.url.endsWith('/decision')).length,0);
    assert.ok(messages.every(message=>!JSON.stringify(message).includes('synthetic-secret')));
  });
}
