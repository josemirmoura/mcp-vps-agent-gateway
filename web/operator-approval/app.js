'use strict';
// Contract: GET /operator/api/approvals/:id returns authenticated JSON with
// request_id,node,subject,resource,access,ttl_label,expires_at,ceiling_wide,
// status,csrf_token. POST /operator/api/approvals/:id/decision accepts decision
// ('approve'|'deny') plus CSRF header. Backend MUST enforce identity, CSRF,
// policy, one-shot, expiry and audit. This UI grants no authority on its own.
const $=id=>document.getElementById(id);
const requestId=new URL(location.href).searchParams.get('request');
const validId=/^apr_[a-zA-Z0-9_-]{8,128}$/;
let csrf='';
function status(message){$('status').textContent=message}
async function readJSON(response){if(!response.ok){const errors={401:'Sessão não autenticada. Entre pelo portal seguro.',403:'Operador sem autorização para este pedido.',404:'Pedido não está mais pendente: pode ter sido concluído ou expirado.',409:'Pedido expirado ou já decidido. Confira o estado antes de tentar novamente.',503:'Broker temporariamente indisponível.'};throw Error(errors[response.status]||'Não foi possível confirmar a solicitação.')}return response.json()}
async function load(){
 if(!requestId||!validId.test(requestId)){status('Abra um pedido válido pelo link fornecido pelo Pórtico.');return}
 try{
  const data=await readJSON(await fetch('/operator/api/approvals/'+encodeURIComponent(requestId),{credentials:'same-origin',cache:'no-store',headers:{Accept:'application/json'}}));
  for(const k of ['request_id','node','subject','resource','access'])$(k).textContent=String(data[k]??'');
  $('ttl').textContent=String(data.ttl_label??'');
  $('expires').textContent=data.expires_at?new Date(data.expires_at).toLocaleString('pt-BR'):'';
  $('warning').hidden=!data.ceiling_wide;
  csrf=typeof data.csrf_token==='string'?data.csrf_token:'';
  $('details').hidden=false;
  const pending=data.status==='pending'&&!!csrf;
  $('approve').disabled=!pending;$('deny').disabled=!pending;
  status(pending?'Confira o recurso e o perfil antes de decidir.':'Este pedido não está disponível para decisão.');
 }catch(e){status(e.message);$('details').hidden=true;$('login-panel').hidden=!e.message.includes('Sessão não autenticada')}
}
async function decide(decision){
 if(!csrf||!confirm(decision==='approve'?'Confirmar autorização exatamente como exibida?':'Negar esta solicitação?'))return;
 $('approve').disabled=true;$('deny').disabled=true;
 try{
  const result=await readJSON(await fetch('/operator/api/approvals/'+encodeURIComponent(requestId)+'/decision',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json','X-CSRF-Token':csrf},body:JSON.stringify({decision})}));
  if(result.status!==(decision==='approve'?'approved':'denied'))throw Error('O Broker não confirmou a decisão.');
  csrf='';
  $('login-panel').hidden=true;
  $('details').hidden=false;
  status(decision==='approve'?'Autorização aprovada com sucesso. Permissão concedida conforme prazo exibido.':'Solicitação negada com sucesso. Nenhuma permissão foi concedida.');
  // A decisão remove o pedido da fila pendente. NÃO recarregar os detalhes:
  // GET retornaria 404 e substituiria a confirmação por um falso erro.
 }catch(e){
  csrf='';
  status(e.message+' Confira o estado do pedido antes de tentar novamente.');
 }
}
$('approve').addEventListener('click',()=>decide('approve'));
$('deny').addEventListener('click',()=>decide('deny'));
$('login-form').addEventListener('submit',async e=>{e.preventDefault();$('login-error').textContent='';try{const res=await fetch('/operator/api/login',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json'},body:JSON.stringify({password:$('password').value})});if(!res.ok)throw Error('Falha na autenticação.');$('password').value='';$('login-panel').hidden=true;await load()}catch(err){$('login-error').textContent=err.message}});
load();
