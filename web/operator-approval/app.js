'use strict';
// Every credential stays on the operator origin. The MCP bridge sees public
// readiness/status only; incoming postMessages never trigger a decision.
const $=id=>document.getElementById(id);
const requestId=new URL(location.href).searchParams.get('request');
const validId=/^apr_[a-zA-Z0-9_-]{8,100}$/;
const embedded=location.pathname==='/operator/embed';
const api=embedded?'/operator/embed/api':'/operator/api';
let csrf='',nonce='',stepUp=false,sessionCsrf='',loginBusy=false;
function status(message){$('status').textContent=message}
function notify(state){if(embedded&&window.parent!==window)window.parent.postMessage({type:'portico-operator',request_id:requestId,status:state},'*')}
async function readJSON(response){if(!response.ok){const errors={401:'Sessão não autenticada. Entre na Central segura.',403:'Operador sem autorização ou verificação recusada.',404:'Pedido indisponível para este operador.',409:'Decisão expirada, repetida ou sessão alterada. Consulte o estado.',428:'Confirme novamente a identidade para esta operação.',503:'Broker temporariamente indisponível. Consulte o estado antes de repetir.'};throw Error(errors[response.status]||'Não foi possível confirmar a solicitação.')}return response.json()}
async function load(){
 csrf='';nonce='';$('approve').disabled=true;$('deny').disabled=true;
 if(!requestId||!validId.test(requestId)){status('Abra um pedido válido pelo link fornecido pelo Pórtico.');return}
 try{
  const data=await readJSON(await fetch(api+'/approvals/'+encodeURIComponent(requestId),{credentials:'same-origin',cache:'no-store',headers:{Accept:'application/json'}}));
  if(data.request_id!==requestId)throw Error('O Broker não confirmou o pedido solicitado.');
  for(const k of ['request_id','node','subject','resource','access'])$(k).textContent=String(data[k]??'');
  $('operator').textContent=String(data.operator??'');
  $('ttl').textContent=String(data.ttl_label??'');
  $('expires').textContent=data.expires_at?new Date(data.expires_at).toLocaleString('pt-BR'):'';
  $('warning').hidden=!data.ceiling_wide;
  csrf=typeof data.csrf_token==='string'?data.csrf_token:'';
  sessionCsrf=csrf||sessionCsrf;
  nonce=typeof data.decision_nonce==='string'?data.decision_nonce:'';
  stepUp=data.step_up_required===true;
  $('step-up-panel').hidden=!stepUp;
  $('details').hidden=false;$('login-panel').hidden=true;
  const pending=data.status==='pending'&&!!csrf&&!!nonce;
  $('approve').disabled=!pending;$('deny').disabled=!pending;
  const final={approved:'Autorização confirmada pelo Broker.',denied:'Solicitação negada. Nenhuma permissão concedida.',expired:'Pedido ou permissão expirou.',cancelled:'Pedido cancelado.',revoked:'Permissão revogada.'};
  status(pending?'Confira a máquina, o solicitante, o recurso e o perfil antes de decidir.':(final[data.status]||'Pedido indisponível para decisão.'));
  notify(pending?'ready':data.status);
 }catch(e){status(e.message);$('details').hidden=true;$('login-panel').hidden=!e.message.includes('Sessão não autenticada')}
}
async function decide(decision){
 // The visible owner button is the explicit confirmation. Browser modal
 // dialogs are not available in the minimum MCP Apps iframe sandbox.
 if(!csrf||!nonce)return;
 $('approve').disabled=true;$('deny').disabled=true;
 try{
  if(decision==='approve'&&stepUp){
   const password=$('step-up-password').value;$('step-up-password').value='';
   await readJSON(await fetch(api+'/approvals/'+encodeURIComponent(requestId)+'/step-up',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json','X-CSRF-Token':csrf},body:JSON.stringify({decision_nonce:nonce,password})}));
  }
  const result=await readJSON(await fetch(api+'/approvals/'+encodeURIComponent(requestId)+'/decision',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json','X-CSRF-Token':csrf},body:JSON.stringify({decision,decision_nonce:nonce})}));
  if(result.status!==(decision==='approve'?'approved':'denied'))throw Error('O Broker não confirmou a decisão.');
  csrf='';nonce='';$('login-panel').hidden=true;$('details').hidden=false;$('step-up-panel').hidden=true;
  status(decision==='approve'?'Autorização aprovada com sucesso. Permissão concedida conforme prazo exibido.':'Solicitação negada com sucesso. Nenhuma permissão foi concedida.');
  notify(result.status);
  // Retain confirmed final state. No pending-only refetch may overwrite success.
 }catch(e){csrf='';nonce='';status(e.message+' Use Consultar estado antes de tentar novamente.')}
}
$('approve').addEventListener('click',()=>decide('approve'));
$('deny').addEventListener('click',()=>decide('deny'));
$('refresh').addEventListener('click',load);
$('logout').addEventListener('click',async()=>{if(!sessionCsrf)return;try{await readJSON(await fetch(api+'/logout',{method:'POST',credentials:'same-origin',headers:{'X-CSRF-Token':sessionCsrf}}));sessionCsrf='';csrf='';nonce='';$('details').hidden=true;$('login-panel').hidden=false;status('Sessão encerrada.')}catch(e){status(e.message)}});
async function login(){
 // A sandbox without allow-forms aborts native submission before the submit
 // event. Use explicit button/keyboard actions and same-origin fetch instead.
 if(loginBusy||!$('login-form').reportValidity())return;
 loginBusy=true;$('login-button').disabled=true;$('login-error').textContent='';
 const password=$('password').value;$('password').value='';
 try{await readJSON(await fetch(api+'/login',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json'},body:JSON.stringify({password})}));$('login-panel').hidden=true;await load()}
 catch(err){$('login-error').textContent=err.message}
 finally{loginBusy=false;$('login-button').disabled=false}
}
$('login-button').addEventListener('click',login);
$('password').addEventListener('keydown',e=>{if(e.key==='Enter'&&!e.isComposing){e.preventDefault();if(!e.repeat)void login()}});
$('login-form').addEventListener('submit',e=>{e.preventDefault();void login()});
load();
