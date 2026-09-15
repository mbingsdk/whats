"use client";
import {useCallback,useEffect,useRef,useState} from "react";
import {useRouter} from "next/navigation";
import {IdentityError,request,type Page} from "../lib/identity-api";
import {draftKey,loadDraft,saveDraft,invalidateDrafts,purgeDrafts,type Draft} from "../lib/inbox-drafts";
import {type Organization,type User} from "./ui";
type Conversation={id:string;display_name:string;identity_value:string;sending_phone:string;status:string;priority:string;handoff_state:string;assigned_team_id:string|null;assigned_member_id:string|null;assignment_revision:number;revision:number;inbound_sequence:number;unread_count?:number;manual_unread?:boolean;window_expires_at:string|null;window_confidence:string;server_now:string};
type Message={id:string;direction:string;message_type:string;text_body:string;content:Record<string,unknown>;provider_at:string|null;received_at:string;delivery_state:string;processing_state:string;error_code:string|null};
type Note={mentioned_me:boolean;id:string;author_member_id:string;body:string;revision:number;redacted_at:string|null;created_at:string};
type Thread={message_has_more:boolean;message_next_cursor:string|null;permissions:string[];conversation:Conversation[];messages:Message[];notes:Note[];presence:{member_id:string;name:string}[]};
type Setup={member_id:string;session_id:string;teams:{id:string;name:string}[];members:{id:string;name:string;team_id:string|null}[];settings:{sending_enabled:boolean;billing_currency_state:string}[];live_test_configured:boolean};
type Decision={allowed:boolean;code:string;window:string;pricing:string;billing_currency_state:string;confirmation_required:boolean;financial_reservation_required:boolean};
type Preflight={decision:Decision;authorization?:string;expires_at?:string;policy_basis?:string};
type Intent={id:string;state:string;error_code?:string;conversation_id?:string};
const endpoint="/api/v1";
function human(v:string){return v.replaceAll("_"," ").toLowerCase();}
function WindowBadge({conversation:c,now}:{conversation:Conversation;now:number}){
 const expiry=c.window_expires_at?Date.parse(c.window_expires_at):0;
 const delta=expiry-now,known=c.window_confidence==="AUTHENTICATED_TEXT"&&expiry>0;
 const state=!known?"UNKNOWN":delta<=0?"EXPIRED":delta<=900000?"EXPIRING_SOON":"ACTIVE";
 return <div className={"window-badge "+state.toLowerCase()}><strong>Service window: {human(state)}</strong>{known&&delta>0&&<span> · {Math.floor(delta/3600000)}h {Math.floor(delta%3600000/60000)}m remaining</span>}<small>{known?"Based on authenticated inbound text. Final checks run when sending.":"No verified window evidence. A reply is blocked."}</small></div>;
}
function MessageContent({m}:{m:Message}){
 if(m.message_type==="TEXT")return <p>{m.text_body}</p>;
 if(["IMAGE","VIDEO","AUDIO","DOCUMENT","STICKER"].includes(m.message_type))return <p>{human(m.message_type)} attachment · metadata only{typeof m.content.filename==="string"?" · "+m.content.filename:""}{typeof m.content.caption==="string"?": "+m.content.caption:""}</p>;
 if(m.message_type==="LOCATION")return <p>Location: {String(m.content.latitude??"unknown")}, {String(m.content.longitude??"unknown")} {typeof m.content.name==="string"?m.content.name:""}</p>;
 if(m.message_type==="REACTION")return <p>Reaction: {typeof m.content.emoji==="string"?m.content.emoji:"removed"}</p>;
 if(m.message_type==="INTERACTIVE")return <p>Interactive response: {JSON.stringify(m.content)}</p>;
 if(m.message_type==="CONTACTS")return <p>Shared contact information: {JSON.stringify(m.content)}</p>;
 if(m.message_type==="TEMPLATE")return <p>Template event: {JSON.stringify(m.content)}</p>;
 return <p>Unsupported message type. The event is retained for review.</p>;
}
export default function Inbox({org,user}:{org:Organization;user:User}){
 const router=useRouter(),[setup,setSetup]=useState<Setup|null>(null),[list,setList]=useState<Conversation[]>([]),[next,setNext]=useState<string|null>(null);
 const [listCursor,setListCursor]=useState(""),[priorityFilter,setPriorityFilter]=useState(""),[unreadOnly,setUnreadOnly]=useState(false),[olderMessages,setOlderMessages]=useState<Message[]>([]),[olderCursor,setOlderCursor]=useState<string|null>(null);
 const [selected,setSelected]=useState(""),[thread,setThread]=useState<Thread|null>(null),[query,setQuery]=useState(""),[status,setStatus]=useState("");
 const [error,setError]=useState(""),[busy,setBusy]=useState(false),[mode,setMode]=useState<"reply"|"note">("reply"),[draft,setDraft]=useState<Draft>({text:"",updatedAt:0}),[restored,setRestored]=useState(false);
 const [preflight,setPreflight]=useState<Preflight|null>(null),[intent,setIntent]=useState<Intent|null>(null),[mention,setMention]=useState(""),[now,setNow]=useState(()=>Date.now()),[offset,setOffset]=useState(0);
 const selection=useRef(""),scope=useRef(0),read=useRef(""),draftScope=useRef("");
 const c=thread?.conversation[0];
 const key=selected?draftKey(user.id,user.user_id,org.id,selected,mode):"";
 const can=(permission:string)=>thread?.permissions.includes(permission)??false;
 const failure=useCallback((e:unknown)=>{setError(e instanceof Error?e.message:"Request failed.");if(e instanceof IdentityError&&e.status===401){invalidateDrafts();router.replace("/login");}},[router]);
 const refresh=useCallback(async()=>{
  const generation=scope.current;
  const page=await request<Page<Conversation>>(endpoint+"/conversations?"+new URLSearchParams({q:query,status,cursor:listCursor,priority:priorityFilter,unread:unreadOnly?"true":""}));
  if(scope.current!==generation)return;setList(page.items);setNext(page.next_cursor);
  const id=selection.current;if(!id)return;
  try{
   const data=await request<Thread>(endpoint+"/conversations/"+id);
   if(scope.current!==generation||selection.current!==id)return;
   setThread(data);const row=data.conversation[0];if(row)setOffset(Date.parse(row.server_now)-Date.now());
  }catch(e){if(e instanceof IdentityError&&(e.status===404||e.status===403)){scope.current++;selection.current="";setSelected("");setThread(null);invalidateDrafts();}throw e;}
 },[query,status,listCursor,priorityFilter,unreadOnly]);
 useEffect(()=>{let active=true;void request<Setup>(endpoint+"/inbox").then(v=>{if(active)setSetup(v);}).catch(failure);return()=>{active=false;};},[failure]);
 useEffect(()=>{void refresh().catch(failure);},[refresh,failure,selected]);
 useEffect(()=>{
  const stream=new EventSource(endpoint+"/inbox/events");let pending=false;
  stream.addEventListener("inbox.invalidate",()=>{if(!pending){pending=true;void refresh().catch(failure).finally(()=>{pending=false;});}});
  const fallback=setInterval(()=>{void refresh().catch(failure);},15000);
  const keepAlive=setInterval(()=>{void request(endpoint+"/auth/session").catch(failure);},120000);
  return()=>{stream.close();clearInterval(fallback);clearInterval(keepAlive);};
 },[refresh,failure]);
 useEffect(()=>{const timer=setInterval(()=>setNow(Date.now()+offset),1000);return()=>clearInterval(timer);},[offset]);
 useEffect(()=>{
  const handle=(e:StorageEvent)=>{if(e.key==="waba.identity-invalidated"){purgeDrafts(sessionStorage);setDraft({text:"",updatedAt:Date.now()});setThread(null);selection.current="";setSelected("");router.replace("/login");}};
  window.addEventListener("storage",handle);return()=>window.removeEventListener("storage",handle);
 },[router]);
 useEffect(()=>{
  if(!key||!c||c.id!==selected||draftScope.current===key)return;
  draftScope.current=key;const stored=loadDraft(sessionStorage,key);setDraft(stored??{text:"",updatedAt:Date.now()});setRestored(!!stored?.text);setIntent(stored?.clientKey?{id:stored.intentId??"",state:"SUBMISSION_UNCERTAIN"}:null);setPreflight(null);
  if(stored?.clientKey){void request<Intent[]>(endpoint+"/outbound-intents?client_key="+encodeURIComponent(stored.clientKey)).then(items=>{if(draftScope.current===key&&items.length)setIntent(items[0]);}).catch(failure);}
 },[key,c,selected,failure]);
 useEffect(()=>{
  if(!c||!selected||read.current===selected+":"+c.inbound_sequence)return;
  read.current=selected+":"+c.inbound_sequence;void request(endpoint+"/conversations/"+selected+"/read","POST",{watermark:c.inbound_sequence,manual_unread:false}).catch(failure);
 },[c,selected,failure]);
 useEffect(()=>{
  if(!selected||mode!=="reply"||!draft.text)return;
  const ping=()=>void request(endpoint+"/conversations/"+selected+"/presence","POST",{}).catch(failure);ping();const timer=setInterval(ping,10000);return()=>clearInterval(timer);
 },[selected,mode,draft.text,failure]);
 useEffect(()=>{
  if(!c||mode!=="reply"||!draft.text.trim()||intent)return;
  let active=true;const timer=setTimeout(()=>{void request<Preflight>(endpoint+"/pricing/preflight","POST",{conversation_id:c.id,assignment_revision:c.assignment_revision,text:draft.text,message_type:"TEXT",category:"SERVICE"}).then(v=>{if(active)setPreflight(v);}).catch(e=>{if(active)failure(e);});},500);
  return()=>{active=false;clearTimeout(timer);};
 },[c,mode,draft.text,intent,failure]);
 const edit=(text:string)=>{const value={text,updatedAt:Date.now()};setDraft(value);saveDraft(sessionStorage,key,value);setRestored(false);setPreflight(null);};
 const act=async(fn:()=>Promise<void>)=>{setBusy(true);setError("");try{await fn();await refresh();}catch(e){failure(e);await refresh().catch(failure);}finally{setBusy(false);}};
 const patch=(body:Record<string,unknown>)=>act(async()=>{if(c){const current=await request<Thread>(endpoint+"/conversations/"+c.id);await request(endpoint+"/conversations/"+c.id,"PATCH",{revision:current.conversation[0].revision,...body});}});
 const choose=(id:string)=>{scope.current++;selection.current=id;setSelected(id);setThread(null);setOlderMessages([]);setOlderCursor(null);setDraft({text:"",updatedAt:0});setIntent(null);setPreflight(null);setRestored(false);};
 const send=()=>act(async()=>{
  if(!c||!draft.text.trim()||intent)return;
  const payload={conversation_id:c.id,assignment_revision:c.assignment_revision,text:draft.text,message_type:"TEXT",category:"SERVICE"};
  const checked=await request<Preflight>(endpoint+"/pricing/preflight","POST",payload);setPreflight(checked);
  if(!checked.decision.allowed||!checked.authorization)return;
  const clientKey=draft.clientKey??crypto.randomUUID(),saved={...draft,clientKey,updatedAt:Date.now()};
  setDraft(saved);saveDraft(sessionStorage,key,saved);setIntent({id:"",state:"SUBMITTING"});
  try{
   const result=await request<Intent>(endpoint+"/outbound-intents","POST",{...payload,client_idempotency_key:clientKey,authorization:checked.authorization});
   setIntent(result);saveDraft(sessionStorage,key,{...saved,intentId:result.id});
  }catch(e){
   setIntent({id:"",state:"SUBMISSION_UNCERTAIN"});
   if(e instanceof IdentityError&&e.status>=400&&e.status<500){setIntent(null);saveDraft(sessionStorage,key,{text:draft.text,updatedAt:Date.now()});setDraft({text:draft.text,updatedAt:Date.now()});throw e;}
   const found=await request<Intent[]>(endpoint+"/outbound-intents?client_key="+encodeURIComponent(clientKey));
   if(found.length)setIntent(found[0]);else if(e instanceof IdentityError&&e.status>=400&&e.status<500){saveDraft(sessionStorage,key,{text:draft.text,updatedAt:Date.now()});setDraft({text:draft.text,updatedAt:Date.now()});throw e;}else{setIntent({id:"",state:"SUBMISSION_UNCERTAIN"});throw e;}
  }
 });
 useEffect(()=>{
  if(!intent||(!intent.id&&!draft.clientKey))return;let active=true;
  const timer=setInterval(()=>{void request<Intent[]>(intent.id?endpoint+"/outbound-intents/"+intent.id:endpoint+"/outbound-intents?client_key="+encodeURIComponent(draft.clientKey??"")).then(items=>{if(active&&items[0])setIntent(items[0]);}).catch(failure);},2000);
  return()=>{active=false;clearInterval(timer);};
 },[intent,draft.clientKey,failure]);
 const note=()=>act(async()=>{if(!c)return;await request(endpoint+"/conversations/"+c.id+"/notes","POST",{body:draft.text,mentions:mention?[mention]:[]});edit("");setMention("");});
 return <section className="inbox-shell" aria-label="Conversation workspace">
 <p className="inbox-policy">Controlled text replies only. Paid sends and templates are closed. Company billing currency: {setup?.settings[0]?.billing_currency_state??"UNKNOWN"}.</p>
 {error&&<p role="alert" className="error">{error}</p>}
 <div className="inbox-columns"><aside className="conversation-list">
 <label>Search conversations<input value={query} onChange={e=>{setQuery(e.target.value);setListCursor("");}} placeholder="Name, number or message"/></label>
 <label>Status<select aria-label="Conversation status filter" value={status} onChange={e=>{setStatus(e.target.value);setListCursor("");}}><option value="">All statuses</option>{["OPEN","SNOOZED","RESOLVED"].map(x=><option key={x}>{x}</option>)}</select></label>
 <label>Priority filter<select aria-label="Priority filter" value={priorityFilter} onChange={e=>{setPriorityFilter(e.target.value);setListCursor("");}}><option value="">All priorities</option>{["LOW","NORMAL","HIGH","URGENT"].map(x=><option key={x}>{x}</option>)}</select></label>
 <label><input type="checkbox" checked={unreadOnly} onChange={e=>{setUnreadOnly(e.target.checked);setListCursor("");}}/> Unread only</label>
 {list.length===0&&<p>No conversations in your permitted scope.</p>}
 {list.map(row=><button className={"conversation-row "+(selected===row.id?"selected":"")} key={row.id} onClick={()=>choose(row.id)}><strong>{row.display_name||row.identity_value}</strong><span>{row.identity_value}</span><small>Via {row.sending_phone} · {human(row.status)} · {human(row.priority)}</small>{!!row.unread_count&&<span className="unread">{row.unread_count} unread</span>}{row.manual_unread&&<small>Marked unread</small>}</button>)}
 {listCursor&&<button onClick={()=>setListCursor("")}>Newest conversations</button>}
 {next&&<button onClick={()=>setListCursor(next)}>Next page</button>}
 </aside><section className="conversation-thread" aria-label="Message thread">
 {!c?<p>Select a conversation to view messages and reply.</p>:<><div className="thread-heading"><h2>{c.display_name||c.identity_value}</h2><p>{c.identity_value} · Sending phone: {c.sending_phone}</p><WindowBadge conversation={c} now={now}/></div>
 <ol className="message-list">{Array.from(new Map([...(thread?.messages??[]),...olderMessages].map(m=>[m.id,m])).values()).reverse().map(m=><li key={m.id} className={"message "+m.direction.toLowerCase()}><small>{human(m.direction)} · {human(m.message_type)} · {new Date(m.provider_at??m.received_at).toLocaleString()}</small><MessageContent m={m}/><small>{human(m.delivery_state)}{m.processing_state==="BLOCKED"?" · blocked":""}{m.error_code?" · "+human(m.error_code):""}</small></li>)}</ol>
 {(olderMessages.length?olderCursor:thread?.message_next_cursor)&&<button disabled={busy} onClick={()=>void act(async()=>{const id=c.id;const page=await request<Thread>(endpoint+"/conversations/"+id+"?before="+encodeURIComponent(olderMessages.length?olderCursor??"":thread?.message_next_cursor??""));if(selection.current===id){setOlderMessages(v=>[...v,...page.messages]);setOlderCursor(page.message_next_cursor);}})}>Older messages</button>}
 {!!thread?.presence.length&&<p className="presence">{thread.presence.map(p=>p.name).join(", ")} is composing. This is advisory; sending is not locked.</p>}
 <div className="composer"><div className="composer-tabs"><button aria-pressed={mode==="reply"} onClick={()=>setMode("reply")}>Reply</button><button aria-pressed={mode==="note"} onClick={()=>setMode("note")}>Internal note</button></div>
 {restored&&<p role="status">Draft restored for this conversation.</p>}
 <label>{mode==="reply"?"Text reply":"Internal note — visible only to authorized colleagues"}<textarea aria-label={mode==="reply"?"Text reply":"Internal note"} value={draft.text} maxLength={mode==="reply"?4096:8000} rows={4} disabled={busy||!!intent} onChange={e=>edit(e.target.value)}/></label>
 {mode==="reply"?<><div className="pricing-guard" aria-live="polite"><strong>Pricing Guard</strong><p>{preflight?human(preflight.decision.code):draft.text.trim()?"Checking current authorization…":"Write a text reply to check eligibility."}</p>{preflight&&<small>{human(preflight.decision.pricing)} · Window: {human(preflight.decision.window)} · Currency: {preflight.decision.billing_currency_state}{preflight.decision.allowed?" · No charge confirmation or monetary reservation required.":""}</small>}</div>
 {intent?<div role="status" className="notice"><strong>Send intent: {human(intent.state)}</strong>{intent.error_code&&<p>{human(intent.error_code)}</p>}{["UNCERTAIN","SUBMISSION_UNCERTAIN"].includes(intent.state)?<p>Delivery is uncertain. Do not resend this message. An operator must reconcile the provider evidence.</p>:["ACCEPTED","FAILED","BLOCKED"].includes(intent.state)?<button onClick={()=>{setIntent(null);edit("");}}>Compose a new reply</button>:<p>Waiting for dispatch and provider evidence.</p>}</div>:<button disabled={busy||!can("messages.send")||!draft.text.trim()||!preflight?.decision.allowed} onClick={()=>void send()}>Send text reply</button>}
 <small>Attachments and template sends are unavailable in this sprint.</small></>:<><label>Mention colleague<select aria-label="Mention colleague" value={mention} onChange={e=>setMention(e.target.value)}><option value="">No mention</option>{Array.from(new Map(setup?.members.map(m=>[m.id,m])).values()).map(m=><option key={m.id} value={m.id}>{m.name}</option>)}</select></label><button disabled={busy||!can("notes.write")||!draft.text.trim()} onClick={()=>void note()}>Add internal note</button></>}
 </div>
 <section aria-label="Internal notes"><h3>Internal notes</h3>{thread?.notes.map(n=><article key={n.id} className="internal-note"><small>{setup?.members.find(m=>m.id===n.author_member_id)?.name??"Colleague"} · {new Date(n.created_at).toLocaleString()}</small>{n.mentioned_me&&<strong>You were mentioned</strong>}<p>{n.redacted_at?"Note redacted":n.body}</p>{!n.redacted_at&&n.author_member_id===org.member_id&&can("notes.write")&&<button onClick={()=>{const body=window.prompt("Edit internal note",n.body);if(body!==null)void act(async()=>{await request(endpoint+"/notes/"+n.id,"PATCH",{revision:n.revision,body});});}}>Edit note</button>}{!n.redacted_at&&can("notes.redact")&&<button onClick={()=>{if(window.confirm("Redact this internal note? Its content will be removed."))void act(async()=>{await request(endpoint+"/notes/"+n.id+"/redact","POST",{revision:n.revision});});}}>Redact note</button>}</article>)}</section>
 </>}
 </section><aside className="conversation-details" aria-label="Conversation controls">{c&&<><h3>Conversation</h3>
 <label>Status<select disabled={busy||!can("conversations.manage")} value={c.status} onChange={e=>{const value=e.target.value;if(value==="SNOOZED")void patch({status:value,snoozed_until:new Date(Date.now()+3600000).toISOString()});else void patch({status:value});}}>{["OPEN","RESOLVED","SNOOZED"].map(x=><option key={x}>{x}</option>)}</select></label><small>Snooze pauses the conversation for one hour.</small>
 <label>Priority<select aria-label="Priority" disabled={busy||!can("conversations.manage")} value={c.priority} onChange={e=>void patch({priority:e.target.value})}>{["LOW","NORMAL","HIGH","URGENT"].map(x=><option key={x}>{x}</option>)}</select></label>
 <label>Handoff<select aria-label="Handoff" disabled={busy||!can("conversations.manage")} value={c.handoff_state} onChange={e=>void patch({handoff_state:e.target.value})}><option value="WAITING_AGENT">Waiting for an agent</option><option value="HUMAN">Human handling</option>{c.handoff_state==="BOT"&&<option value="BOT">Bot foundation (inactive)</option>}</select></label>
 <label>Assigned team<select aria-label="Assigned team" value={c.assigned_team_id??""} disabled={busy||!can("conversations.assign")} onChange={e=>void act(async()=>{await request(endpoint+"/conversations/"+c.id+"/assignment","PUT",{assignment_revision:c.assignment_revision,team_id:e.target.value||null,member_id:null});})}><option value="">Unassigned team</option>{setup?.teams.map(t=><option key={t.id} value={t.id}>{t.name}</option>)}</select></label>
 <label>Assigned colleague<select aria-label="Assigned colleague" value={c.assigned_member_id??""} disabled={busy||!can("conversations.assign")} onChange={e=>void act(async()=>{await request(endpoint+"/conversations/"+c.id+"/assignment","PUT",{assignment_revision:c.assignment_revision,team_id:c.assigned_team_id,member_id:e.target.value||null});})}><option value="">Unassigned colleague</option>{Array.from(new Map(setup?.members.filter(m=>!c.assigned_team_id||m.team_id===c.assigned_team_id).map(m=>[m.id,m])).values()).map(m=><option key={m.id} value={m.id}>{m.name}</option>)}</select></label>
 <button disabled={busy} onClick={()=>void act(async()=>{await request(endpoint+"/conversations/"+c.id+"/read","POST",{watermark:c.inbound_sequence,manual_unread:true});})}>Mark unread</button>
 </>}</aside></div></section>;
}
